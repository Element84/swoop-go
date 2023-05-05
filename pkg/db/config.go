package db

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Pass     string
	Name     string
	UrlExtra string
}

// TODO support parameter overrides for testing, maybe
func (db *DatabaseConfig) Url() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s%s",
		db.User,
		db.Pass,
		db.Host,
		db.Port,
		db.Name,
		db.UrlExtra,
	)
}

func (db *DatabaseConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&db.Host,
		"database-host",
		"",
		"swoop database host (required; SWOOP_DATABASE_HOST)",
	)
	cobra.MarkFlagRequired(fs, "database-host")
	fs.IntVar(
		&db.Port,
		"database-port",
		5432,
		"swoop database port (SWOOP_DATABASE_PORT)",
	)
	fs.StringVar(
		&db.User,
		"database-user",
		"",
		"swoop database user (required; SWOOP_DATABASE_USER)",
	)
	cobra.MarkFlagRequired(fs, "database-user")
	fs.StringVar(
		&db.Pass,
		"database-password",
		"",
		"swoop database password (required; SWOOP_DATABASE_PASSWORD)",
	)
	cobra.MarkFlagRequired(fs, "database-password")
	fs.StringVar(
		&db.Name,
		"database-name",
		"",
		"swoop database name (required; SWOOP_DATABASE_NAME)",
	)
	cobra.MarkFlagRequired(fs, "database-name")
	fs.StringVar(
		&db.UrlExtra,
		"database-url-extra",
		"",
		"swoop database url extra parameters (SWOOP_DATABASE_URL_EXTRA)",
	)
}
