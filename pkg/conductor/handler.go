package conductor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
)

const (
	// TODO: make this a top-level config param
	pollInterval = 600 * time.Second
	// TODO: handler parameter, related to rate limiting and max concurrency
	batchSize = 100
)

type nothing struct{}

func HandleActionWrapper(
	ctx context.Context,
	conn db.Conn,
	thread *db.Thread,
	isAsyncAction bool,
	handleFn func() error,
) error {
	var err error
	// TODO: need a test to verify we don't leak locks
	defer thread.Unlock(ctx, conn)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if isAsyncAction {
		err = thread.InsertQueuedEvent(ctx, tx)
	} else {
		err = thread.InsertSuccessfulEvent(ctx, tx)
	}
	if err != nil {
		return err
	}

	handleError := func(_err error) error {
		err := tx.Rollback(ctx)
		if err != nil {
			return err
		}

		// TODO: need to handle backoff and retries, not just jump to failed
		// TODO: we don't have a way to track attempts and therefore can't calculate backoff
		return thread.InsertFailedEvent(ctx, conn, _err.Error())
	}

	err = handleFn()
	if err != nil {
		_err := handleError(err)
		if _err != nil {
			return fmt.Errorf(
				"error running action: %s; while handling that error encountered another: %s",
				err,
				_err,
			)
		}
		return err
	}

	return tx.Commit(ctx)
}

type HandlerClient interface {
	HandleAction(ctx context.Context, conn db.Conn, thread *db.Thread) error
}

type Handler struct {
	name       string
	isNotified chan nothing
	conf       *config.Handler
	client     HandlerClient
}

/*
SOME INCOHERENT AND LIKELY MISLEADING NOTES:

rate limit (configuredLimit) is actions per Second
- failure halves limit to Min(limit, Every(maxBackoff))
- success doubles limit to Max(limit, configuredLimit)
  - we don't actually want to update the rate on every completion
  - periodic rate update method takes successCount - failureCount to derive new limit

also need a maxConcurrency specifying how many actions can be in process at any given time
- defaults to Max(int(configuredLimit), 1)
- queryLimit is Min(maxConcurrency - currentActions, availableTokens)

  - we can work out the rate limiting with https://pkg.go.dev/golang.org/x/time/rate

  - we can work out the concurrency limit with https://pkg.go.dev/golang.org/x/sync/semaphore

  - except we can't see the capacity, so we have a gap here...

  - query is not thread-safe, but is a private method and is only called internally in one place

  - Bad example code:
    queryLimit := 0
    for {
    // This is dumb, we only need to
    queryLimit = Min(maxConLimiter.Tokens(), maxConcurrency - currentActions)
    if queryLmit >= 1 {
    break
    }
    time.Sleep(100 * time.Milisecond)
    }

    err := Limiter.WaitN(ctx, queryLimit)

MAYBE BETTER IDEA:

We have a "semaphore" and a token bucket. The semaphore needs to return us
the max number of "resources" available, same with the bucket and number of
tokens, but we block on either until that number is at least 1. Then we
take the min of those values and request that allocation from each.

See the following resources:
  - https://cs.opensource.google/go/x/sync/+/refs/tags/v0.3.0:semaphore/semaphore.go
  - https://cs.opensource.google/go/x/time/+/master:rate/rate.go

Can we combine the bucket and semaphore into a single object? Like the
limit is adjusted based on the number of reserved, i.e., limit =
max_limit - allocated?  Like a token bucket that you have to release as
well as request? <-- YES, THIS
*/
func NewHandlerFromConfig(ctx context.Context, conf *config.Handler) (*Handler, error) {
	var client HandlerClient
	switch conf.Type {
	case config.ArgoWorkflows:
		c, err := NewArgoClient(ctx, conf.ArgoConf, conf.Workflows)
		if err != nil {
			return nil, fmt.Errorf("failed making client: %s", err)
		}
		client = c
	default:
		return nil, fmt.Errorf("unsupported handler type: '%s'", conf.Type)
	}

	return &Handler{
		name:       conf.Name,
		isNotified: make(chan nothing, 1),
		conf:       conf,
		client:     client,
	}, nil
}

func (h *Handler) GetName() string {
	return h.name
}

func (h *Handler) Notify() {
	// TODO: batch requests until either count or timer reaches threshold before notifying
	h.NotifyNow()
}

func (h *Handler) NotifyNow() {
	select {
	case h.isNotified <- nothing{}:
		log.Printf("handler %s: received notification", h.name)
	default:
		// notification already pending, nothing to do
	}
}

func (h *Handler) query(ctx context.Context, conn db.Conn, limit int) ([]*db.Thread, error) {
	return db.GetProcessableThreads(ctx, conn, h.name, limit, []uuid.UUID{})
}

func (h *Handler) poller(ctx context.Context) {
	// polling is implemented simply by periodic "notification"
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			h.NotifyNow()
		}
	}
}

func (h *Handler) Run(ctx context.Context, conn db.Conn) error {
	// escape hatch when context is done
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	select {
	case <-ctx.Done():
		return nil
	case <-h.isNotified:
	}

	threads, err := h.query(ctx, conn, batchSize)
	if err != nil {
		return err
	}

	// TODO: test case for this condition
	if len(threads) == batchSize {
		// if we got as many records as we asked for then we suspect we have
		// more records to process and we notify so we'll immediately query again
		h.NotifyNow()
	}

	// We could handle the threads in a goroutine to unblock and increase
	// throughput. But then we have to track all actionUuids using this
	// connection, and ensure they are filtered from queries. The expense of
	// queries then increases, and at a increasing volumes will actually begin
	// to reduce throughput.
	//
	// Perhaps the best solution is ultimately to have a worker pool that can
	// do the heavy lifiting for handlers. That could possibly be a worker pool
	// per handler, to prevent other handlers being blocked, or maybe it is a
	// global pool to keep load more predicatable. Either way, such a model
	// would offer a means to process a block of records to completion in a
	// blocking fashion, but allow multiple blocks to execute in parallel to
	// help mitigate the impact of slow requests.
	var wg sync.WaitGroup
	for _, thread := range threads {
		thread := thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := h.client.HandleAction(ctx, conn, thread)
			if err != nil {
				log.Printf("handler %s: failed to process thread %s: %s", h.name, thread.Uuid, err)
				return
			}
			log.Printf("handler %s: successfully processed thread %s", h.name, thread.Uuid)
		}()
	}

	wg.Wait()
	return nil
}

func (h *Handler) Start(ctx context.Context, dbConf *db.ConnectConfig) error {
	// TODO: need some way to re-enter this, say when we get a db connection error or something
	//       wait.Until maybe?
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// force polling on start
	h.NotifyNow()

	conn, err := dbConf.Connect(ctx)
	defer conn.Close(ctx)

	if err != nil {
		return err
	}

	go h.poller(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err = h.Run(ctx, conn)
		if err != nil {
			return err
		}
	}
}
