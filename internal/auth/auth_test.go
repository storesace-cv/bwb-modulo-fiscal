package auth_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
)

func TestDevStaticAuthenticate(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef"
	forbidden := "fedcba9876543210fedcba9876543210"
	a, err := auth.NewDevStatic(auth.DevStaticConfig{
		Token:          token,
		ScopeID:        "scope-dev",
		ForbiddenToken: forbidden,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	p, err := a.Authenticate(ctx, req)
	if err != nil || p.ScopeID != "scope-dev" {
		t.Fatalf("got %#v %v", p, err)
	}

	req.Header.Set("Authorization", "Bearer "+forbidden)
	_, err = a.Authenticate(ctx, req)
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("err = %v", err)
	}

	req.Header.Set("Authorization", "Bearer wrong-token-wrong-token-wrong!!")
	_, err = a.Authenticate(ctx, req)
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}

	req.Header.Del("Authorization")
	_, err = a.Authenticate(ctx, req)
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestDevStaticRejectsWrongTokenDifferentLengths(t *testing.T) {
	token := strings.Repeat("a", 32)
	a, err := auth.NewDevStatic(auth.DevStaticConfig{Token: token, ScopeID: "scope-dev"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, wrong := range []string{"x", strings.Repeat("b", 16), strings.Repeat("c", 31), strings.Repeat("d", 64), strings.Repeat("e", 128)} {
		req, _ := http.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+wrong)
		_, err := a.Authenticate(ctx, req)
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("len=%d err=%v", len(wrong), err)
		}
	}
}

func TestNewDevStaticRejectsShortToken(t *testing.T) {
	_, err := auth.NewDevStatic(auth.DevStaticConfig{Token: "short", ScopeID: "s"})
	if err == nil {
		t.Fatal("expected error")
	}
}
