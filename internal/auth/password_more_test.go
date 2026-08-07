package auth_test

import (
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/auth"
)

func TestCheckPasswordRejectsMalformedHashes(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		"not-a-hash",
		"bcrypt$v=19$m=1,t=1,p=1$YWJj$YWJj",
		"argon2id$v=18$m=65536,t=1,p=2$YWJjZGVmZ2hpamtsbW5vcA$YWJjZGVmZ2hpamtsbW5vcGFiY2RlZmdoaWprbG1ub3A",
		"argon2id$v=19$bad$YWJj$YWJj",
		"argon2id$v=19$m=65536,t=1,p=2$@@@@$YWJj",
		"argon2id$v=19$m=65536,t=1,p=2$YWJjZGVmZ2hpamtsbW5vcA$@@@@",
		"argon2id$v=19$x=1,t=1,p=2$YWJjZGVmZ2hpamtsbW5vcA$YWJjZGVmZ2hpamtsbW5vcGFiY2RlZmdoaWprbG1ub3A",
	}
	for _, hash := range cases {
		if auth.CheckPassword(hash, "password") {
			t.Fatalf("CheckPassword(%q) = true, want false", hash)
		}
	}
}
