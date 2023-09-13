package conductor

import (
	"context"

	"github.com/element84/swoop-go/pkg/config/http"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/errors"
	"github.com/element84/swoop-go/pkg/s3"
)

type httpClient struct {
	*http.Client
	s3      *s3.SwoopS3
	isAsync bool
}

// TODO: we'll need to pass the secrets object on through here
func newSyncHttpClient(client *http.Client, s3 *s3.SwoopS3) *httpClient {
	return &httpClient{client, s3, false}
}

func (hc *httpClient) HandleAction(ctx context.Context, conn db.Conn, thread *db.Thread) error {
	handleFn := func() error {
		// IDEA: we can cache the loaded params on error for the next retry in an LRU cache
		// TODO: this assumes that http actions are only used for callbacks
		//   -> maybe need to use a single actions prefix in the bucket?
		//   -> or look at action type and use that to parameterize the prefix?
		params, err := hc.s3.GetCallbackParams(ctx, thread.Uuid)
		if err != nil {
			return errors.NewRequestError(err, false)
		}

		request, err := hc.NewRequest(map[string]any{
			"uuid":       thread.Uuid,
			"parameters": params,
		})
		if err != nil {
			return errors.NewRequestError(err, false)
		}

		return hc.MakeRequest(ctx, request)
		// TODO: save response to object storage
		//   -> need to return it from make request in some way I guess
		//   -> I guess we need a response model that has what we care about
		//      (oh, I think we have that!) and can be dumped to json
	}
	return HandleActionWrapper(ctx, conn, thread, hc.isAsync, handleFn)
}
