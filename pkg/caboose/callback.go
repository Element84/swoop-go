package caboose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/states"
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

func (cm CallbackMap) lookup(wfName string, status states.FinalState) (Callbacks, bool) {
	wf, ok := cm[wfName]
	if !ok {
		return nil, false
	}

	cba, ok := wf[status]
	if !ok {
		return nil, false
	}

	return cba, true
}

type CallbackExecutor struct {
	ctx  context.Context
	s3   *s3.S3Driver
	conn db.Conn
}

func NewCallbackExecutor(ctx context.Context, s3 *s3.S3Driver, conn db.Conn) *CallbackExecutor {
	return &CallbackExecutor{ctx, s3, conn}
}

// TODO: probably extract these to a wrapper around the driver,
//
//	so we can use the put method for the workflow resource
func (cbx *CallbackExecutor) getJsonFromObject(key string) (any, error) {
	stream, err := cbx.s3.Get(cbx.ctx, key)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	stat, err := stream.Stat()
	if err != nil {
		return nil, err
	}

	contentBytes := make([]byte, stat.Size)
	b, err := stream.Read(contentBytes)
	if int64(b) != stat.Size && err != nil {
		return nil, err
	}

	var j any
	err = json.Unmarshal(contentBytes, &j)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (cbx *CallbackExecutor) putJsonIntoObject(key string, j any) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(j)
	if err != nil {
		return err
	}

	opts := &s3.PutOptions{
		// allows us to preview in the minio console
		// application/json would be more appropriate but can't be previewed
		ContentType: "text/plain",
	}

	err = cbx.s3.Put(cbx.ctx, key, b, int64(b.Len()), opts)
	if err != nil {
		return err
	}

	return nil
}

func (cbx *CallbackExecutor) getInput(workflowUUID uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/input.json", workflowUUID)
	return cbx.getJsonFromObject(key)
}

func (cbx *CallbackExecutor) getOutput(workflowUUID uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/output.json", workflowUUID)
	return cbx.getJsonFromObject(key)
}

func (cbx *CallbackExecutor) putCallbackParams(callbackUUID uuid.UUID, json any) error {
	key := fmt.Sprintf("callbacks/%s/parameters.json", callbackUUID)
	return cbx.putJsonIntoObject(key, json)
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

func (cbx *CallbackExecutor) processCallback(
	cb *config.Callback, wfProps *WorkflowProperties, data *map[string]any,
) error {
	cbUuid, err := db.NewCallbackAction(
		cb.Name,
		cb.HandlerName,
		cb.Handler.Type,
		wfProps.Uuid,
	).Insert(cbx.ctx, cbx.conn)
	if err != nil {
		return err
	}
	if cbUuid.IsNil() {
		return fmt.Errorf("no uuid returned for callback action")
	}

	params, err := cbx.extractParams(cb.Parameters, cb.ValidateParams, data)
	if err != nil {
		// an error validating params is fatal and should not be retried
		// so we insert a failure for the callback and return early
		event := &db.Event{
			ActionUuid: cbUuid,
			Status:     states.Failed,
			ErrorMsg:   err.Error(),
		}
		err := event.Insert(cbx.ctx, cbx.conn)
		if err != nil {
			return err
		}
		return nil
	}

	err = cbx.putCallbackParams(cbUuid, params)
	if err != nil {
		return err
	}

	return nil
}

func (cbx *CallbackExecutor) ProcessCallbacks(cbs Callbacks, wfProps *WorkflowProperties) error {
	input, err := cbx.getInput(wfProps.Uuid)
	if err != nil {
		return err
	}

	output, err := cbx.getOutput(wfProps.Uuid)
	if err != nil {
		return err
	}

	data := map[string]any{
		"input":  input,
		"output": output,
	}

	for _, callback := range cbs {
		switch callback.Type {
		case config.SingleCallback:
			err := cbx.processCallback(callback, wfProps, &data)
			if err != nil {
				// TODO
				return err
			}
		case config.PerFeatureCallback:
			// TODO: probably need a safer way to handle this json query
			features, ok := input.(map[string]any)["features"].([]any)
			if !ok {
				// TODO
				return fmt.Errorf("failed to get features")
			}
			for _, feature := range features {
				data["feature"] = feature
				err := cbx.processCallback(callback, wfProps, &data)
				if err != nil {
					return err
					// TODO
				}
			}
		default:
			return fmt.Errorf("Unknown callback type: '%s'", callback.Type)
		}
	}

	return nil
}
