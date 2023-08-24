package context

import (
	"context"
)

func MergeCancel(ctx, cancelCtx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	stop := context.AfterFunc(cancelCtx, func() {
		cancel(context.Cause(cancelCtx))
	})
	return ctx, func() {
		stop()
		cancel(context.Canceled)
	}
}
