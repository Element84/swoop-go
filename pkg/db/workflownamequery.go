package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
)

type WorkflowNameQuery struct {
	WorkflowUuid uuid.UUID
}

func (wnq *WorkflowNameQuery) Exec(ctx context.Context, conn Conn) (string, error) {
	var name string

	err := conn.QueryRow(
		ctx,
		`SELECT
		  action_name
		FROM swoop.action
		WHERE
		  action_uuid = $1`,
		wnq.WorkflowUuid,
	).Scan(&name)
	if err != nil {
		return "", err
	}

	return name, nil
}
