package cmd

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// TODO: this binds _all_ flags, even those from other sources, like k8s
// note that we then have 1) undocumented env vars and 2) possible name collisions
//
// how do we better document supported env vars (auto-doc env var names in help?)?
// how do we namespace flags (can flagsets be nested?)?
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
