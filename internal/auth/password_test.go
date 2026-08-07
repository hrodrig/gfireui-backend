package auth_test

import (
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/auth"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned an empty hash")
	}
	if !auth.CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("CheckPassword() = false, want true")
	}
	if auth.CheckPassword(hash, "wrong password") {
		t.Fatal("CheckPassword() = true for the wrong password")
	}
}
