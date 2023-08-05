package db

import (
	"context"
	"fmt"
	"strings"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectConfig struct {
	// we could in the future support more config parameters, see
	// https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Config
	Database *string
}

func (c ConnectConfig) String() string {
	params := []string{}

	if c.Database != nil {
		params = append(params, fmt.Sprintf("dbname=%s", *c.Database))
	}

	return strings.Join(params[:], " ")
}

func (conf *ConnectConfig) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(conf.String())
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	return pgxpool.NewWithConfig(ctx, config)
}
