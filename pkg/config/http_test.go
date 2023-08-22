package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

var conf = `
url: https://{secrets.minio-user}:{secrets.minio-password}@our-minio:9000
method: POST
body: |
  {
    "fixed": "a_value",
    "name": "{{ .parameters.workflowName -}}",
    "date": "{{ .parameters.feature.properties.datetime -}}"
  }
headers:
  Authorization: "Basic {{ .secrets.user }} {{ .secrets.password}}"
  Content-Type: "application/json"
  X-Workflow-Name: "{{ .parameters.workflowName }}"
followRedirects: false
responses:
  - status: 400
    message: ".*timed out.*"
    fatal: false
  - status: 400
    fatal: true
`

func Test_HttpRequest(t *testing.T) {
	secrets := map[string]any{
		"minio-user":     "Franz",
		"minio-password": "don'tstealmypassword",
		"user":           "Kafka",
		"password":       "please!please!please!",
	}
	parameters := map[string]any{
		"workflowName":  "some_workflow",
		"executionTime": "2023-04-04 04:40:04.444T00:00",
	}
	//data := map[string]any{
	_ = map[string]any{
		"secrets":    secrets,
		"parameters": parameters,
	}

	hr := &HttpRequest{}

	err := yaml.Unmarshal([]byte(conf), &hr)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}
}
