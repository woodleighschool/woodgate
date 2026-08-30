package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodgate/internal/directory"
)

func TestCurrentUserRequiresAccessEnabled(t *testing.T) {
	sessions, ctx := loadedSession(t)
	sessions.Put(ctx, sessionUserIDKey, int64(42))
	service := &Service{
		users:    staticUsers{user: &directory.User{ID: 42}},
		sessions: sessions,
	}

	if _, err := service.CurrentUser(ctx); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentUser() error = %v, want ErrNotAuthenticated", err)
	}
}

func TestLoginStartsSessionForAccessEnabledUser(t *testing.T) {
	const password = "correct horse battery staple"
	hash, err := directory.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	sessions, ctx := loadedSession(t)
	service := &Service{
		users: staticUsers{user: &directory.User{
			ID:            42,
			Email:         "person@example.invalid",
			PasswordHash:  hash,
			AccessEnabled: true,
		}},
		sessions:  sessions,
		dummyHash: hash,
	}

	user, err := service.Login(ctx, LoginParams{
		Email:    " person@example.invalid ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if user.ID != 42 {
		t.Fatalf("Login() user ID = %d, want 42", user.ID)
	}
	if got := sessions.GetInt64(ctx, sessionUserIDKey); got != 42 {
		t.Fatalf("session user ID = %d, want 42", got)
	}
}

func loadedSession(t *testing.T) (*scs.SessionManager, context.Context) {
	t.Helper()
	sessions := scs.New()
	sessions.Store = memstore.New()
	ctx, err := sessions.Load(t.Context(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	return sessions, ctx
}

type staticUsers struct {
	user *directory.User
}

func (s staticUsers) Get(context.Context, int64) (*directory.User, error) {
	return s.user, nil
}

func (s staticUsers) GetAccount(context.Context, int64) (*directory.Account, error) {
	return &directory.Account{User: *s.user}, nil
}

func (s staticUsers) GetLoginByEmail(context.Context, string) (*directory.User, error) {
	return s.user, nil
}

func (s staticUsers) GetSSOByEmail(context.Context, string) (*directory.User, error) {
	return s.user, nil
}
