package conductor

import (
	"context"
	"io"
	"log"

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

		resp, err := hc.MakeRequest(ctx, request)

		readBody := func() (*string, error) {
			reader, err := request.GetBody()
			if err != nil {
				return nil, err
			}

			b, err := io.ReadAll(reader)
			if err != nil {
				return nil, err
			}

			body := string(b)

			return &body, nil
		}

		logRequest := func() error {
			body, err := readBody()
			if err != nil {
				log.Printf("failed to read request body: %s", err)
			}

			return hc.s3.PutCallbackHttp(
				ctx,
				thread.Uuid,
				map[string]any{
					"request": map[string]any{
						"method": request.Method,
						"url":    request.URL.String(),
						"proto":  request.Proto,
						"header": request.Header,
						"body":   body,
						"host":   request.Host,
					},
					"response": resp,
				},
			)
		}

		// TODO: LOGGING LIKE THIS COULD LEAK SECRETS!!!
		// We should consider having a way to:
		//   - toggle logging on/off (off by default)
		//   - redact certain info (sounds hard/impossible)
		//   - ???
		_err := logRequest()
		if _err != nil {
			log.Printf("failed to write http data: %s", _err)
		}

		return err
	}
	return HandleActionWrapper(ctx, conn, thread, hc.isAsync, handleFn)
}
