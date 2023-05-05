package cmd

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
