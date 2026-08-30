package ctxkeys

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodgate/internal/directory"
)

func TestUserRoundTrip(t *testing.T) {
	want := &directory.User{ID: 42, Email: "person@example.invalid", AccessEnabled: true}
	ctx := WithUser(context.Background(), want)

	got, ok := User(ctx)
	if !ok || got != want {
		t.Fatalf("User = %#v, %t", got, ok)
	}
	if current := CurrentUserID(ctx); current == nil || *current != want.ID {
		t.Fatalf("CurrentUserID = %v", current)
	}
}

func TestRequireUserRejectsAnonymousContext(t *testing.T) {
	if _, err := RequireUser(context.Background()); err == nil {
		t.Fatal("RequireUser accepted an anonymous context")
	}
}
