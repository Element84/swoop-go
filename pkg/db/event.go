package db

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"

	scontext "github.com/element84/swoop-go/pkg/context"
	"github.com/element84/swoop-go/pkg/states"
)

type Event struct {
	ActionUuid   uuid.UUID
	Time         time.Time
	Status       states.ActionState
	ErrorMsg     string
	RetrySeconds int
}

func (s *Event) Insert(ctx context.Context, conn Conn) error {
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
	var (
		err          error
		retrySeconds *int
		appName      *string
	)

	if s.RetrySeconds != 0 {
		retrySeconds = &s.RetrySeconds
	}

	app, _err := scontext.GetApplicationName(ctx)
	if _err == nil {
		appName = &app
	}

	if s.Time.IsZero() {
		_, err = conn.Exec(
			ctx,
			`INSERT INTO swoop.event (
				event_time,
				action_uuid,
				status,
				error,
				retry_seconds,
				event_source
			) VALUES (
				now(),
				$1,
				$2,
				$3,
				$4,
				$5
			) ON CONFLICT DO NOTHING`,
			s.ActionUuid,
			s.Status,
			s.ErrorMsg,
			retrySeconds,
			appName,
		)
	} else {
		_, err = conn.Exec(
			ctx,
			`INSERT INTO swoop.event (
				action_uuid,
				event_time,
				status,
				error,
				retry_seconds,
				event_source
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6
			) ON CONFLICT DO NOTHING`,
			s.ActionUuid,
			s.Time,
			s.Status,
			s.ErrorMsg,
			retrySeconds,
			appName,
		)
	}
	return err
}
