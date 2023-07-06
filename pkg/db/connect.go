package db

import (
	"context"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	// we could in the future support more config parameters, see
	// https://pkg.go.dev/github.com/jackc/pgx/v4/pgxpool#Config
	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	return pgxpool.NewWithConfig(ctx, config)
}
