package config

import (
	"errors"
	"fmt"
)

type Workflows map[string]*Workflow

func (wfs *Workflows) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p Workflows

	err := unmarshal((*p)(wfs))
	if err != nil {
		return err
	}

	for wfId, wf := range *wfs {
		wf.setId(wfId)
	}

	return nil
}

func (wfs Workflows) setHandlers(handlers Handlers) error {
	for _, wf := range wfs {
		err := wf.setHandlers(handlers)
		if err != nil {
			return err
		}
	}

	return nil
}

type Workflow struct {
	Id          string
	handler     *Handler
	HandlerName string              `yaml:"handler"`
	Callbacks   Callbacks           `yaml:"callbacks"`
	ArgoOpts    *ArgoWorkflowOpts   `yaml:"argoOpts,omitempty"`
	CirrusOpts  *CirrusWorkflowOpts `yaml:"cirrusOpts,omitempty"`
}

func (wf *Workflow) setId(id string) error {
	if wf.ArgoOpts != nil {
		err := wf.ArgoOpts.SetWorkflowIdLabel(id)
		if err != nil {
			return err
		}
	}

	wf.Id = id

	return nil
}

func (wf *Workflow) GetHandler() *Handler {
	return wf.handler
}

func (wf *Workflow) setHandlers(handlers Handlers) error {
	err := wf.setHandler(handlers)
	if err != nil {
		return fmt.Errorf("workflow '%s': %s", wf.Id, err)
	}

	err = wf.Callbacks.setHandlers(handlers)
	if err != nil {
		return fmt.Errorf("workflow '%s': %s", wf.Id, err)
	}

	return nil
}

func (wf *Workflow) setHandler(handlers Handlers) error {
	handler, ok := handlers[wf.HandlerName]
	if !ok {
		return fmt.Errorf(
			"cannot resolve handler '%s'",
			wf.HandlerName,
		)
	}

	switch handler.Type {
	case ArgoWorkflows:
		if wf.ArgoOpts == nil {
			return errors.New("an argo workflow must define 'argoOpts'")
		}
	case Cirrus:
		if wf.CirrusOpts == nil {
			return errors.New("a cirrus workflow must define 'cirrusOpts'")
		}
	default:
		return fmt.Errorf("not a valid workflow handler type: '%s'", handler.Type)
	}

	wf.handler = handler
	handler.Workflows = append(handler.Workflows, wf)

	return nil
}
