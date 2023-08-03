package caboose

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/states"
)

type WorkflowProperties struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Uuid       uuid.UUID
	Name       string
	Status     states.WorkflowState
	ErrorMsg   string
}

func (p *WorkflowProperties) ToStartEvent() *db.Event {
	return &db.Event{
		ActionUuid: p.Uuid,
		Time:       p.StartedAt,
		Status:     states.Running,
	}
}

func (p *WorkflowProperties) ToEndEvent() *db.Event {
	return &db.Event{
		ActionUuid: p.Uuid,
		Time:       p.FinishedAt,
		Status:     p.Status,
		ErrorMsg:   p.ErrorMsg,
	}
}

func (p *WorkflowProperties) LookupName(ctx context.Context, conn db.Conn) error {
	if p.Name != "" {
		// nothing to do if it is already set
		return nil
	}

	if p.Uuid.IsNil() {
		return fmt.Errorf("cannot lookup workflow execution name with nil uuid")
	}

	name, err := (&db.WorkflowNameQuery{
		WorkflowUuid: p.Uuid,
	}).Exec(ctx, conn)
	if err != nil {
		return err
	}

	p.Name = name
	return nil
}
