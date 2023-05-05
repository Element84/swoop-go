package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/element84/swoop-go/pkg/config"
)

func bindFlags(fs *pflag.FlagSet, v *viper.Viper) {
	fs.VisitAll(func(f *pflag.Flag) {
		bindFlag(fs, v, f)
	})
}

func bindFlag(fs *pflag.FlagSet, v *viper.Viper, f *pflag.Flag) {
	// if a flag hasn't changed and viper has a value...
	if !f.Changed && v.IsSet(f.Name) {
		// ...we set the flag from viper, which also
		// passes the value to any bound variables
		fs.Set(f.Name, fmt.Sprintf("%v", v.Get(f.Name)))
	}
}

func databaseFlags(fs *pflag.FlagSet) *config.DatabaseConfig {
	db := &config.DatabaseConfig{}
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
	return db
}
