package domain_test

import (
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestRoleValid(t *testing.T) {
	if !domain.RoleAdministrator.Valid() {
		t.Fatal("Administrator should be valid")
	}
	if domain.Role("SecretsManager").Valid() {
		t.Fatal("unknown role must be invalid")
	}
}
