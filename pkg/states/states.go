package states

import (
	"fmt"
	"strings"
)

type WorkflowState string

const (
	Running    WorkflowState = "RUNNING"
	Successful WorkflowState = "SUCCESSFUL"
	Failed     WorkflowState = "FAILED"
	Canceled   WorkflowState = "CANCELED"
	TimedOut   WorkflowState = "TIMED_OUT"
	Invalid    WorkflowState = "INVALID"
)

var WorkflowStates = map[WorkflowState]struct{}{
	Running:    {},
	Successful: {},
	Failed:     {},
	Canceled:   {},
	TimedOut:   {},
	Invalid:    {},
}

func (fs WorkflowState) String() string {
	return string(fs)
}

func Parse(s string) (WorkflowState, error) {
	ws := WorkflowState(strings.ToUpper(s))

	_, ok := WorkflowStates[ws]
	if !ok {
		return "", fmt.Errorf("unknown workflow state: '%s'", s)
	}

	return ws, nil
}

type FinalState WorkflowState

var FinalStates = map[FinalState]struct{}{
	FinalState(Successful): {},
	FinalState(Failed):     {},
	FinalState(Canceled):   {},
	FinalState(TimedOut):   {},
	FinalState(Invalid):    {},
}

func ParseFinalState(s string) (FinalState, error) {
	fs := FinalState(strings.ToUpper(s))

	_, ok := FinalStates[fs]
	if !ok {
		return "", fmt.Errorf("not a final workflow state: '%s'", s)
	}

	return fs, nil
}

func (fs *FinalState) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var state string

	err := unmarshal(&state)
	if err != nil {
		return err
	}

	*fs, err = ParseFinalState(state)
	if err != nil {
		return err
	}

	return nil
}
