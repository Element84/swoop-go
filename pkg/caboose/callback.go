package caboose

import (
	"context"
	"fmt"
	"log"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/states"
	"github.com/element84/swoop-go/pkg/utils"
)

type Callbacks []*config.Callback
type CallbackMap map[string]map[states.FinalState]Callbacks

func MapConfigCallbacks(sc *config.SwoopConfig) CallbackMap {
	var cm = CallbackMap{}

	for wfName, wf := range sc.Workflows {
		cm[wfName] = map[states.FinalState]Callbacks{}

		for state := range states.FinalStates {
			cm[wfName][state] = Callbacks{}
		}

		for _, cb := range wf.Callbacks {
			for _, state := range *cb.When {
				cm[wfName][state] = append(cm[wfName][state], cb)
			}
		}
	}

	return cm
}

func (cm CallbackMap) Lookup(wfName string, status states.FinalState) (Callbacks, bool) {
	wf, ok := cm[wfName]
	if !ok {
		return Callbacks{}, false
	}

	cba, ok := wf[status]
	if !ok {
		return Callbacks{}, false
	}

	return cba, true
}

type CallbackExecutor struct {
	ctx  context.Context
	s3   *s3.SwoopS3
	conn db.Conn
}

func NewCallbackExecutor(ctx context.Context, s3 *s3.SwoopS3, conn db.Conn) *CallbackExecutor {
	return &CallbackExecutor{ctx, s3, conn}
}

func (cbx *CallbackExecutor) extractParams(
	parameters *config.CallbackParameters,
	validator func(params any) error,
	data *map[string]any,
) (*map[string]any, error) {
	params := map[string]any{}
	for paramName, param := range *parameters {
		if param.HasPath() {
			result := param.GetJsonPath(data).([]any)
			// GetJsonPath always returns an array even if only one match
			if len(result) == 0 {
				return nil, fmt.Errorf("failed to extract value for parameter '%s'", paramName)
			} else if len(result) == 1 {
				params[paramName] = result[0]
			} else {
				params[paramName] = result
			}
		} else {
			params[paramName] = param.Value
		}
	}

	err := validator(params)
	if err != nil {
		return nil, fmt.Errorf("callback parameters did not validate: %s", err)
	}

	return &params, nil
}

func (cbx *CallbackExecutor) insertCallback(
	name string,
	handlerName string,
	handlerType config.HandlerType,
	wfUuid uuid.UUID,
) (uuid.UUID, error) {
	cbUuid, err := db.NewCallbackAction(
		name,
		handlerName,
		handlerType.String(),
		wfUuid,
	).Insert(cbx.ctx, cbx.conn)

	if err != nil {
		return uuid.UUID{}, err

	}
	if cbUuid.IsNil() {
		return uuid.UUID{}, fmt.Errorf("no uuid returned for callback action")
	}

	return cbUuid, nil
}

func (cbx *CallbackExecutor) failCallback(cbUuid uuid.UUID, err error) error {
	log.Printf("ERROR: %s", err)
	event := &db.Event{
		ActionUuid: cbUuid,
		Status:     states.Failed,
		ErrorMsg:   err.Error(),
	}

	err = event.Insert(cbx.ctx, cbx.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cbx *CallbackExecutor) insertAndFailCallback(
	name string,
	handlerName string,
	handlerType config.HandlerType,
	wfUuid uuid.UUID,
	err error,
) error {
	cbUuid, _err := cbx.insertCallback(name, handlerName, handlerType, wfUuid)
	if _err != nil {
		return _err
	}

	return cbx.failCallback(cbUuid, err)
}

func (cbx *CallbackExecutor) processCallback(
	cb *config.Callback, wfProps *WorkflowProperties, data *map[string]any,
) error {
	cbUuid, err := cbx.insertCallback(cb.Name, cb.HandlerName, cb.Handler.Type, wfProps.Uuid)
	if err != nil {
		return err
	}

	params, err := cbx.extractParams(cb.Parameters, cb.ValidateParams, data)
	if err != nil {
		// an error extracting params is fatal and should not be retried
		// so we insert a failure for the callback and return early
		return cbx.failCallback(cbUuid, err)
	}

	err = cbx.s3.PutCallbackParams(cbx.ctx, cbUuid, params)
	if err != nil {
		return err
	}

	return nil
}

func (cbx *CallbackExecutor) ProcessCallbacks(cbs Callbacks, wfProps *WorkflowProperties) error {
	if len(cbs) == 0 {
		return nil
	}

	jsonProps, err := utils.Jsonify(wfProps)
	if err != nil {
		// this really should not happen
		return err
	}

	input, err := cbx.s3.GetInput(cbx.ctx, wfProps.Uuid)
	if err != nil {
		// TODO: seems like we need to handle obviously-not-retryable
		//       errors differently than those that could be retried.
		return err
	}

	output := map[string]any{}
	if states.ActionState(wfProps.Status) == states.Successful {
		_output, err := cbx.s3.GetOutput(cbx.ctx, wfProps.Uuid)
		if err != nil {
			// TODO: seems like we need to handle obviously-not-retryable
			//       errors differently than those that could be retried.
			return err
		}
		output = _output.(map[string]any)
	}

	data := map[string]any{
		"input":    input,
		"output":   output,
		"workflow": jsonProps,
	}

	for _, callback := range cbs {
		switch callback.Type {
		case config.SingleCallback:
			err := cbx.processCallback(callback, wfProps, &data)
			if err != nil {
				// if we get an error back here it is possibly a transient problem
				// we return it to bubble it up to the general workflow retry mechanism
				return fmt.Errorf(
					"workflow '%s' callback '%s' failed: %s",
					wfProps.Name,
					callback.Name,
					err,
				)
			}
		case config.PerFeatureCallback:
			features, ok := output["features"].([]any)
			if !ok {
				// this is not a retryable error -- output won't change
				err := fmt.Errorf(
					"workflow '%s' callback '%s' is type '%s' but extracting output features failed",
					wfProps.Name,
					callback.Name,
					callback.Type,
				)

				err = cbx.insertAndFailCallback(
					callback.Name,
					callback.HandlerName,
					callback.Handler.Type,
					wfProps.Uuid,
					err,
				)
				if err != nil {
					return fmt.Errorf(
						"workflow '%s' callback '%s' failed to fail: %s",
						wfProps.Name,
						callback.Name,
						err,
					)
				}

				continue
			}
			for featIdx, feature := range features {
				// TODO: need to filter features with callback's filter
				data["feature"] = feature
				err := cbx.processCallback(callback, wfProps, &data)
				if err != nil {
					// if we get an error back here it is possibly a transient problem
					// we return it to bubble it up to the general workflow retry mechanism
					return fmt.Errorf(
						"workflow '%s' callback '%s' for feature index '%d' failed: %s",
						wfProps.Name,
						callback.Name,
						featIdx,
						err,
					)
				}
			}
		default:
			// this is not a retryable error, so we shouldn't return it as one
			// we really shouldn't get here, at least not with callbacks coming
			// from the config file
			err := fmt.Errorf(
				"workflow '%s' callback '%s' has unknown callback type '%s'",
				wfProps.Name,
				callback.Name,
				callback.Type,
			)

			err = cbx.insertAndFailCallback(
				callback.Name,
				callback.HandlerName,
				callback.Handler.Type,
				wfProps.Uuid,
				err,
			)
			if err != nil {
				return fmt.Errorf(
					"workflow '%s' callback '%s' failed to fail: %s",
					wfProps.Name,
					callback.Name,
					err,
				)
			}

			continue
		}
	}

	return nil
}
