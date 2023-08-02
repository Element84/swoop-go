package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
)

type CallbackAction struct {
	callbackName string
	handlerName  string
	handlerType  string
	workflowUuid uuid.UUID
}

func NewCallbackAction(
	callbackName string,
	handlerName string,
	handlerType string,
	workflowUuid uuid.UUID,
) *CallbackAction {
	return &CallbackAction{
		callbackName,
		handlerName,
		handlerType,
		workflowUuid,
	}
}

func (cba *CallbackAction) Insert(ctx context.Context, conn Conn) (uuid.UUID, error) {
	var id uuid.UUID

	err := conn.QueryRow(
		ctx,
		`INSERT INTO swoop.action (
			action_type,
			action_name,
			handler_name,
			handler_type,
			parent_uuid
		) VALUES (
			'callback',
			$1,
			$2,
			$3,
			$4
		) RETURNING action_uuid`,
		cba.callbackName,
		cba.handlerName,
		cba.handlerType,
		cba.workflowUuid,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
