package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gopkg.in/yaml.v3"
)

type testServer struct {
	Server       *httptest.Server
	StatusCode   int
	ResponseBody string
}

func (ts *testServer) handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(ts.StatusCode)
	fmt.Fprintln(w, ts.ResponseBody)
}

func mkTestServer(t testing.TB, statusCode int, respBody string) *testServer {
	ts := &testServer{
		StatusCode:   statusCode,
		ResponseBody: respBody,
	}
	ts.Server = httptest.NewServer(http.HandlerFunc(ts.handler))
	t.Cleanup(ts.Server.Close)
	return ts
}

var conf = `
url: "https://{{.secrets.minioUser}}:{{.secrets.minioPassword}}@our-minio:9000"
method: "POST"
body: |
  {
    "fixed": "a_value",
    "name": "{{ .parameters.workflowName -}}",
    "date": "{{ .parameters.executionTime -}}"
  }
headers:
  Authorization: "Basic {{ .secrets.user }} {{ .secrets.password}}"
  Content-Type: "application/json"
  X-Workflow-Name: "{{ .parameters.workflowName }}"
followRedirects: false
responses:
  - statusCode: 404
    message: ".*timed out.*"
    result: Fatal
  - statusCode: 404
    message: "no problem!"
    result: success
  - statusCode: 401
    jsonPath: '[?($.int == 1)]'
    result: success
  - statusCode: 400
    jsonPath: '[?($.error =~ "(?i)^.*transient.*$")]'
    result: error
  - statusCode: 400
    result: fatal
  - statusCode: 400
    result: SUCCESS
  - statusCode: 201
    result: fatal
`

func Test_Client(t *testing.T) {
	ctx := context.Background()
	secrets := map[string]any{
		"minioUser":     "Franz",
		"minioPassword": "don'tstealmypassword",
		"user":          "Kafka",
		"password":      "please!please!please!",
	}
	// TODO: load this via json to support nesting, then try referencing nested keys
	parameters := map[string]any{
		"workflowName":  "some_workflow",
		"executionTime": "2023-04-04 04:40:04.444T00:00",
	}
	data := map[string]any{
		"secrets":    secrets,
		"parameters": parameters,
	}

	hr := &Client{}

	err := yaml.Unmarshal([]byte(conf), hr)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	req, err := hr.NewRequest(data)
	if err != nil {
		t.Fatalf("error templating request: %s", err)
	}

	// TODO: validate request was templated correctly
	t.Logf("%+v", req)

	ts := mkTestServer(t, 0, "")
	tsUrl, err := url.Parse(ts.Server.URL)
	if err != nil {
		t.Fatalf("test server url '%s' is bad: %s", ts.Server.URL, err)
	}
	req.URL = tsUrl

	for _, test := range []struct {
		respCode int
		respBody string
		expected RequestResult
	}{
		{200, "", Success},
		{201, "", Fatal},
		{400, `{"error": "nothing"}`, Fatal},
		{400, `{"detail": "some message", "error": "twas TRANSIENT"}`, Error},
		{404, "", Error},
		{404, "no problem!", Success},
		{404, "some thing about a timed out request...", Fatal},
		{401, `{"noint": null}`, Error},
		{401, `{"int": 10}`, Error},
		{401, `{"int": 1}`, Success},
	} {
		t.Run(
			"",
			func(t *testing.T) {
				ts.StatusCode = test.respCode
				ts.ResponseBody = test.respBody

				err := hr.MakeRequest(ctx, req)
				result, ok := RequestResultFromError(err)
				if !ok {
					t.Logf("%+v", err)
					t.Fatalf("failed to make request: %s", err)
				}

				if result != test.expected {
					t.Fatalf("expected request result to be '%s', got '%s'", test.expected, result)
				}
			},
		)
	}
}
