package station

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
)

type deviceStoreFake struct {
	mu              sync.Mutex
	station         Station
	secret          string
	observedVersion *int
	observedBuild   string
}

func (f *deviceStoreFake) authenticate(_ context.Context, secret string) (*Station, error) {
	if secret != f.secret {
		return nil, ErrUnauthorized
	}
	station := f.station
	return &station, nil
}

func (f *deviceStoreFake) observeClient(
	_ context.Context,
	_ int64,
	secret string,
	version *int,
	build string,
) error {
	if secret != f.secret {
		return ErrUnauthorized
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.observedVersion = version
	f.observedBuild = build
	return nil
}

func (f *deviceStoreFake) observation() (*int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.observedVersion, f.observedBuild
}

type checkinSubmitterFake struct {
	submission CheckinSubmission
}

type brandingDeliveryFake struct {
	locationID int64
	kind       string
}

func (f *brandingDeliveryFake) DeliverStationBackground(
	w http.ResponseWriter,
	_ *http.Request,
	locationID int64,
) error {
	f.locationID = locationID
	f.kind = "background"
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (f *brandingDeliveryFake) DeliverStationLogo(
	w http.ResponseWriter,
	_ *http.Request,
	locationID int64,
) error {
	f.locationID = locationID
	f.kind = "logo"
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (f *checkinSubmitterFake) SubmitStationCheckin(
	_ context.Context,
	submission CheckinSubmission,
) (*CheckinReceipt, error) {
	f.submission = submission
	return &CheckinReceipt{
		ID:         31,
		PersonID:   submission.PersonID,
		LocationID: submission.LocationID,
		Direction:  submission.Direction,
	}, nil
}

func TestCheckinUsesStationLocation(t *testing.T) {
	store := &deviceStoreFake{
		station: Station{ID: 7, LocationID: 11, Enabled: true},
		secret:  "test-station-secret",
	}
	checkins := &checkinSubmitterFake{}
	server, err := newServer(store, Dependencies{
		Locations: locationProviderFake{},
		Checkins:  checkins,
	}, "test", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	t.Cleanup(server.Close)

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("person_id", "23")
	_ = form.WriteField("location_id", "999")
	_ = form.WriteField("direction", "check_in")
	if err := form.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	router := chi.NewRouter()
	server.RegisterRoutes(router, router)
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		deviceBasePath+"/checkins",
		body,
	)
	request.Header.Set("Authorization", "Bearer "+store.secret)
	request.Header.Set("Content-Type", form.FormDataContentType())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
	if checkins.submission.StationID != 7 || checkins.submission.LocationID != 11 ||
		checkins.submission.PersonID != 23 {
		t.Fatalf("submission = %#v", checkins.submission)
	}
}

func TestBrandingUsesStationLocation(t *testing.T) {
	store := &deviceStoreFake{
		station: Station{ID: 7, LocationID: 11, Enabled: true},
		secret:  "test-station-secret",
	}
	branding := &brandingDeliveryFake{}
	server, err := newServer(store, Dependencies{Branding: branding}, "test", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	t.Cleanup(server.Close)

	router := chi.NewRouter()
	server.RegisterRoutes(router, router)
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		deviceBasePath+"/configuration/background",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+store.secret)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
	if branding.locationID != 11 || branding.kind != "background" {
		t.Fatalf("branding location/kind = %d/%s", branding.locationID, branding.kind)
	}
}

func TestConnectRejectsIncompatibleProtocolAndRecordsBuild(t *testing.T) {
	store := &deviceStoreFake{
		station: Station{ID: 7, LocationID: 11, Enabled: true},
		secret:  "test-station-secret",
	}
	server, err := newServer(store, Dependencies{}, "server-build", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	t.Cleanup(server.Close)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, deviceBasePath+"/connect", nil)
	request.Header.Set("Authorization", "Bearer "+store.secret)
	request.Header.Set("Sec-WebSocket-Protocol", "woodgate-station.v2")
	request.Header.Set(AppBuildHeader, "2.0.0+4")
	recorder := httptest.NewRecorder()
	server.connect(recorder, request)

	if recorder.Code != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUpgradeRequired)
	}
	if recorder.Header().Get(ProtocolHeader) != Subprotocol {
		t.Fatalf("required protocol = %q", recorder.Header().Get(ProtocolHeader))
	}
	version, build := store.observation()
	if version == nil || *version != 2 || build != "2.0.0+4" {
		t.Fatalf("observed version/build = %v/%q", version, build)
	}
}

func TestConnectSendsHelloAndConfigurationChange(t *testing.T) {
	store := &deviceStoreFake{
		station: Station{ID: 7, Name: "Reception", LocationID: 11, Enabled: true},
		secret:  "test-station-secret",
	}
	server, err := newServer(store, Dependencies{}, "server-build", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	t.Cleanup(server.Close)

	router := chi.NewRouter()
	server.RegisterRoutes(router, router)
	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+store.secret)
	headers.Set(AppBuildHeader, "2.0.0+4")
	connection, response, err := websocket.Dial( //nolint:bodyclose // websocket.Dial owns the response body.
		ctx,
		"ws"+strings.TrimPrefix(httpServer.URL, "http")+deviceBasePath+"/connect",
		&websocket.DialOptions{HTTPHeader: headers, Subprotocols: []string{Subprotocol}},
	)
	if err != nil {
		if response != nil {
			t.Fatalf("Dial: %v (status %d)", err, response.StatusCode)
		}
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close(websocket.StatusNormalClosure, "") })

	_, helloData, err := connection.Read(ctx)
	if err != nil {
		t.Fatalf("read hello: %v", err)
	}
	var hello serverMessage
	if err := json.Unmarshal(helloData, &hello); err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	if hello.Type != messageHello || hello.Station == nil || hello.Station.ID != 7 ||
		hello.Station.LocationID != 11 || hello.ProtocolVersion != ProtocolVersion {
		t.Fatalf("hello = %#v", hello)
	}

	server.ConfigurationChanged(7, 29)
	_, changedData, err := connection.Read(ctx)
	if err != nil {
		t.Fatalf("read configuration change: %v", err)
	}
	var changed serverMessage
	if err := json.Unmarshal(changedData, &changed); err != nil {
		t.Fatalf("decode configuration change: %v", err)
	}
	if changed.Type != messageConfigurationChanged {
		t.Fatalf("configuration change = %#v", changed)
	}

	presence, err := json.Marshal(clientMessage{Type: messagePresence})
	if err != nil {
		t.Fatalf("encode presence: %v", err)
	}
	if err := connection.Write(ctx, websocket.MessageText, presence); err != nil {
		t.Fatalf("write presence: %v", err)
	}
}
