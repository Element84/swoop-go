package db

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ActionUUID uuid.UUID
	EventTime  time.Time
	Status     string
	ErrorMsg   string
}

func (s *Event) Insert(ctx context.Context, db *pgxpool.Pool) (pgconn.CommandTag, error) {
	/*
		// We could do something like this if we wanted to prevent events being inserted for
		// unknown workflows. In reality, however, the current risk of not checking seems low.
		// If we did want this check, then it might make more sense as a trigger on event insert,
		// or perhaps a foreign key relation to action might be better (but runs into complications
		// with partitioning). For now we'll keep this here as a reference, in case we want it.
		var actionExists bool
		err := db.QueryRow(
			ctx,
			"SELECT exists(SELECT 1 from swoop.action where action_uuid = $1)",
			s.actionUUID,
		).Scan(&actionExists)

		if err != nil {
			// returning nil here doesn't work, we need a CommandTag
			return nil, err
		} else if !actionExists {
			// returning nil here doesn't work, we need a CommandTag
			return nil, fmt.Errorf("Cannot insert event, unknown action UUID: '%s'", s.actionUUID)
		}
	*/

	return db.Exec(
		ctx,
		`INSERT INTO swoop.event (
		    action_uuid,
			event_time,
			status,
			error,
			event_source
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			'swoop-caboose'
		) ON CONFLICT DO NOTHING`,
		s.ActionUUID,
		s.EventTime,
		s.Status,
		s.ErrorMsg,
	)
}
