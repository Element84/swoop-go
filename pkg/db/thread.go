package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"

	"github.com/element84/swoop-go/pkg/states"
)

type Thread struct {
	Uuid        uuid.UUID
	HandlerName string
	LockId      int
}

func GetProcessableThreads(
	ctx context.Context,
	conn Conn,
	handlerName string,
	limit int,
	ignored []uuid.UUID,
) ([]*Thread, error) {
	rows, _ := conn.Query(
		ctx,
		`SELECT
			t.action_uuid as uuid,
			t.handler_name as handlername,
			t.lock_id as lockid
		FROM swoop.get_processable_actions(
			_ignored_action_uuids => $1,
			_handler_names => $2,
			_limit => $3
		) as pas
		JOIN swoop.thread as t using (action_uuid)`,
		ignored,
		[]string{handlerName},
		limit,
	)
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Thread])
}

func (t *Thread) Unlock(ctx context.Context, conn Conn) error {
	return UnlockThread(ctx, conn, t.LockId)
}

func UnlockThread(ctx context.Context, conn Conn, lockId int) error {
	_, err := conn.Exec(
		ctx,
		"SELECT swoop.unlock_thread($1)",
		lockId,
	)
	return err
}

func (t *Thread) InsertQueuedEvent(ctx context.Context, conn Conn) error {
	return (&Event{
		ActionUuid: t.Uuid,
		Status:     states.Queued,
	}).Insert(ctx, conn)
}

func (t *Thread) InsertSuccessfulEvent(ctx context.Context, conn Conn) error {
	return (&Event{
		ActionUuid: t.Uuid,
		Status:     states.Successful,
	}).Insert(ctx, conn)
}

func (t *Thread) InsertFailedEvent(ctx context.Context, conn Conn, errorMsg string) error {
	return (&Event{
		ActionUuid: t.Uuid,
		Status:     states.Failed,
		ErrorMsg:   errorMsg,
	}).Insert(ctx, conn)
}

func (t *Thread) InsertRetriesExhaustedEvent(ctx context.Context, conn Conn, errorMsg string) error {
	return (&Event{
		ActionUuid: t.Uuid,
		Status:     states.RetriesExhausted,
		ErrorMsg:   errorMsg,
	}).Insert(ctx, conn)
}

func (t *Thread) InsertBackoffEvent(
	ctx context.Context,
	conn Conn,
	retrySeconds int,
	errorMsg string,
) error {
	return (&Event{
		ActionUuid:   t.Uuid,
		Status:       states.Backoff,
		ErrorMsg:     errorMsg,
		RetrySeconds: retrySeconds,
	}).Insert(ctx, conn)
}
