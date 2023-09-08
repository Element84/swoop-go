package regexp

import (
	"regexp"
)

type Regexp struct {
	*regexp.Regexp
}

func Compile(expr string) (*Regexp, error) {
	rx, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	return &Regexp{rx}, nil
}

func (r *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var p string

	err := unmarshal(&p)
	if err != nil {
		return err
	}

	rx, err := Compile(p)
	if err != nil {
		return err
	}
	*r = *rx

	return nil
}
