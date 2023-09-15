package jsonpath

import (
	"github.com/ohler55/ojg/jp"
)

type JsonPath struct {
	*jp.Expr
}

func ParseString(expr string) (*JsonPath, error) {
	jsonPath, err := jp.ParseString(expr)
	if err != nil {
		return nil, err
	}
	return &JsonPath{&jsonPath}, nil
}

func (jp *JsonPath) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	err := unmarshal(&s)
	if err != nil {
		return err
	}

	j, err := ParseString(s)
	if err != nil {
		return err
	}
	*jp = *j

	return nil
}
