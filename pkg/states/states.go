package states

import (
	"fmt"
	"strings"
)

type ActionState string

const (
	Pending          ActionState = "PENDING"
	Queued           ActionState = "QUEUED"
	Running          ActionState = "RUNNING"
	Successful       ActionState = "SUCCESSFUL"
	Failed           ActionState = "FAILED"
	Canceled         ActionState = "CANCELED"
	TimedOut         ActionState = "TIMED_OUT"
	Backoff          ActionState = "BACKOFF"
	Invalid          ActionState = "INVALID"
	RetriesExhausted ActionState = "RETRIES_EXHAUSTED"
	Info             ActionState = "INFO"
)

var ActionStates = map[ActionState]struct{}{
	Pending: {},
	Queued: {},
	Running: {},
	Successful: {},
	Failed: {},
	Canceled: {},
	TimedOut: {},
	Backoff: {},
	Invalid: {},
	RetriesExhausted: {},
	Info: {},
}

func (fs ActionState) String() string {
	return string(fs)
}

func Parse(s string) (ActionState, error) {
	ws := ActionState(strings.ToUpper(s))

	_, ok := ActionStates[ws]
	if !ok {
		return "", fmt.Errorf("unknown action state: '%s'", s)
	}

	return ws, nil
}

type WorkflowState ActionState

var WorkflowStates = map[WorkflowState]struct{}{
	WorkflowState(Running):    {},
	WorkflowState(Successful): {},
	WorkflowState(Failed):     {},
	WorkflowState(Canceled):   {},
	WorkflowState(TimedOut):   {},
	WorkflowState(Invalid):    {},
}

func ParseWorkflowState(s string) (WorkflowState, error) {
	ws := WorkflowState(strings.ToUpper(s))

	_, ok := WorkflowStates[ws]
	if !ok {
		return "", fmt.Errorf("unknown workflow state: '%s'", s)
	}

	return ws, nil
}

type FinalState ActionState

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
