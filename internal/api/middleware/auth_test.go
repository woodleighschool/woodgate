package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/woodleighschool/woodgate/internal/api/ctxkeys"
	"github.com/woodleighschool/woodgate/internal/auth"
	"github.com/woodleighschool/woodgate/internal/directory"
)

type fakeAuthenticator struct {
	user *directory.User
	err  error
}

func (fake *fakeAuthenticator) Authenticate(context.Context) (*directory.User, error) {
	return fake.user, fake.err
}

func TestRequireHTTPAuthAttachesSessionUser(t *testing.T) {
	authenticator := &fakeAuthenticator{user: &directory.User{ID: 42, AccessEnabled: true}}
	handler := RequireHTTPAuth(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := ctxkeys.User(req.Context())
		if !ok || user.ID != 42 {
			t.Fatalf("user = %#v, %t", user, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestRequireHTTPAuthRejectsMissingSession(t *testing.T) {
	handler := RequireHTTPAuth(&fakeAuthenticator{err: auth.ErrNotAuthenticated})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler ran") }),
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestRequireHTTPAuthTreatsLookupFailureAsServerError(t *testing.T) {
	handler := RequireHTTPAuth(&fakeAuthenticator{err: errors.New("database unavailable")})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler ran") }),
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
}
