package config

import (
	"fmt"

	"github.com/element84/swoop-go/pkg/states"
	"github.com/element84/swoop-go/pkg/utils"
)

type CallbackWhen []states.FinalState

func (cw *CallbackWhen) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		vals    = []string{}
		when    = []states.FinalState{}
		notWhen = []states.FinalState{}
	)

	err := unmarshal(&vals)
	if err != nil {
		return err
	}

	for _, val := range vals {
		var not = false

		if val[0] == '!' {
			not = true
			val = val[1:]
		}

		p, err := states.ParseFinalState(val)
		if err != nil {
			// unknown state
			return err
		}

		if not {
			notWhen = append(notWhen, p)
		} else {
			when = append(when, p)
		}
	}

	if len(notWhen) == 0 {
		*cw = when
	} else {
		// validate no conflicts between when and notWhen
		for _, state := range when {
			if utils.Contains(notWhen, state) {
				return fmt.Errorf("conflicting state values: '%s' and '!%s'", state, state)
			}
		}

		// append all not notWhen states to when
		for state := range states.FinalStates {
			if !utils.Contains(notWhen, state) {
				when = append(when, state)
			}
		}

		// dedup when into cw
		var all = map[states.FinalState]struct{}{}
		for _, state := range when {
			if _, ok := all[state]; !ok {
				all[state] = struct{}{}
				*cw = append(*cw, state)
			}
		}
	}

	return nil
}
