package caboose

import (
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/states"
)

type WorkflowProperties struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Uuid       uuid.UUID
	// TODO template name is argo specific, what is the generic?
	TemplateName string
	Status       states.WorkflowState
	ErrorMsg     string
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
