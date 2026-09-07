//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"

	"github.com/woodleighschool/woodgate/internal/appkey"
	"github.com/woodleighschool/woodgate/internal/config"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestNativeKeysCannotAuthenticateBrowserRoutes(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := directory.NewStore(database)
	var adminID int64
	if err := database.QueryRow(ctx, `SELECT id FROM authz_roles WHERE key = 'admin'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	user, err := directory.NewUserService(store).Create(ctx, directory.UserCreate{
		Email: "user@example.invalid", Password: "correct-password", RoleIDs: []int64{adminID},
	})
	if err != nil {
		t.Fatal(err)
	}
	const personalKey = "synthetic-personal-bearer-credential"
	if err := directory.NewAuthnStore(store).SetAPIKey(ctx, user.ID, personalKey); err != nil {
		t.Fatal(err)
	}
	key, err := appkey.NewStore(database).Create(ctx, appkey.Mutation{Name: "Native terminal", AllLocations: true})
	if err != nil {
		t.Fatal(err)
	}
	baseURL := startAuthBoundaryApplication(t, database)
	client := &http.Client{Timeout: 10 * time.Second}
	login, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/session", strings.NewReader(`{"email":"user@example.invalid","password":"correct-password"}`))
	if err != nil {
		t.Fatal(err)
	}
	login.Header.Set("Content-Type", "application/json")
	response, err := client.Do(login)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	cookies := response.Cookies()
	if response.StatusCode != http.StatusOK || len(cookies) != 1 {
		t.Fatalf("browser login = %d, cookies = %d", response.StatusCode, len(cookies))
	}
	for _, tc := range []struct {
		name       string
		nativeKey  string
		bearer     string
		cookie     bool
		identified bool
	}{
		{name: "native header", nativeKey: key.APIKey},
		{name: "native bearer", bearer: key.APIKey},
		{name: "personal header", nativeKey: personalKey},
		{name: "personal bearer", bearer: personalKey, identified: true},
		{name: "browser cookie", cookie: true, identified: true},
		{name: "browser cookie with native header", nativeKey: key.APIKey, cookie: true, identified: true},
		{name: "personal bearer with native header", nativeKey: key.APIKey, bearer: personalKey, identified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range []string{"/api/session", "/api/users", "/api/app-keys"} {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("X-Api-Key", tc.nativeKey)
				if tc.bearer != "" {
					req.Header.Set("Authorization", "Bearer "+tc.bearer)
				}
				if tc.cookie {
					req.AddCookie(cookies[0])
				}
				assertBrowserAuthentication(t, client, req, tc.identified, user.ID)
			}
		})
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Api-Key", key.APIKey)
	response, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("valid native credential was rejected by native route: %d", response.StatusCode)
	}
}

func assertBrowserAuthentication(t *testing.T, client *http.Client, req *http.Request, identified bool, userID int64) {
	t.Helper()
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	want := http.StatusUnauthorized
	if req.URL.Path == "/api/session" || identified {
		want = http.StatusOK
	}
	if response.StatusCode != want {
		t.Fatalf("%s = %d, want %d: %s", req.URL.Path, response.StatusCode, want, body)
	}
	if req.URL.Path != "/api/session" {
		return
	}
	var session struct {
		User *authn.Principal `json:"user"`
	}
	if err := json.Unmarshal(body, &session); err != nil {
		t.Fatal(err)
	}
	if identified && (session.User == nil || session.User.ID != userID) {
		t.Fatalf("session did not resolve user identity: %s", body)
	}
	if !identified && session.User != nil {
		t.Fatalf("native credential authenticated browser session: %s", body)
	}
}

func startAuthBoundaryApplication(t *testing.T, database *pgxpool.Pool) string {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	baseURL := "http://" + listener.Addr().String()
	cfg := config.Config{
		Listen: listener.Addr().String(), ServerURL: baseURL, ClientIPSource: config.ClientIPSourceRemoteAddr,
		StorageKind: "file", StorageFileRoot: t.TempDir(), StorageCapabilityKey: strings.Repeat("a", 64),
		StorageTransferTTL: 15 * time.Minute,
	}
	logger := slog.New(slog.DiscardHandler)
	sessions, sessionStore := newSessions(database, cfg, logger)
	t.Cleanup(sessionStore.StopCleanup)
	objects, err := bloby.New(t.Context(), blobydb.New(database), storageConfig(cfg), logger)
	if err != nil {
		t.Fatal(err)
	}
	app, err := buildApplication(t.Context(), cfg, database, sessions, logger, objects)
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- app.server.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.server.Shutdown(ctx); err != nil {
			t.Errorf("shutdown composed server: %v", err)
		}
		select {
		case err := <-stopped:
			if err != nil {
				t.Errorf("serve composed application: %v", err)
			}
		case <-ctx.Done():
			t.Error("composed server did not stop")
		}
	})
	return baseURL
}
