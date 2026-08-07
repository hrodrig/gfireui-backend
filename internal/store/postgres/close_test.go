package postgres_test

import (
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/store/postgres"
)

func TestCloseNilStore(t *testing.T) {
	t.Parallel()
	var store *postgres.Store
	store.Close()
}
