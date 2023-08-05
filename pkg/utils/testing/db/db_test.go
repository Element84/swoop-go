package db

import (
	"context"
	"testing"
)

func TestSwoopDB(t *testing.T) {
	ctx := context.Background()
	db := NewTestingDB(t, "")
	db.Create(ctx)
	db.LoadFixture(ctx, "base_01")
}
