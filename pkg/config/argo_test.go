package config

import (
	//"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
	"gopkg.in/yaml.v3"

	test "github.com/element84/swoop-go/pkg/utils/testing"
	//"github.com/element84/swoop-go/pkg/utils/testing/k8s"
)

func mkArgoConfYaml(kubeconfig, namespace string) string {
	return fmt.Sprintf(`
instanceId: argoInstance
k8sOptions:
  kubeconfig: %s
  configOverrides:
    context:
      namespace: "%s"
`,
		kubeconfig,
		namespace,
	)
}

func Test_ArgoConf(t *testing.T) {
	//ctx := context.Background()
	expectedNs := "some-random-namespace-name"
	conf := mkArgoConfYaml(
		test.PathFromRoot(t, "kubeconfig.yaml"),
		expectedNs,
	)
	//_ = k8s.TestNamespaceAndConfigFlags(ctx, t, "testing-k8s-")

	ac := &ArgoConf{}

	err := yaml.Unmarshal([]byte(conf), ac)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	namespace, err := ac.GetNamespace()
	if err != nil {
		t.Fatalf("failed getting namespace: %s", err)
	}

	if namespace != expectedNs {
		t.Fatalf("expected namespace '%s', got '%s'", expectedNs, namespace)
	}
}

func Test_Labels(t *testing.T) {
	yml := `
label1: value1
label2: value2
`

	l := Labels{}

	check := func(indicies []int) {
		for _, i := range indicies {
			val, ok := l.Get(fmt.Sprintf("label%d", i))
			if !ok {
				t.Errorf("label not found: label%d", i)
			}
			if val != fmt.Sprintf("value%d", i) {
				t.Errorf("expected 'value%d' for label%d, got '%s'", i, i, val)
			}
		}
	}

	err := yaml.Unmarshal([]byte(yml), &l)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	check([]int{1, 2})

	l, err = l.Add("label3", "value3")
	if err != nil {
		t.Fatal("failed to add label3")
	}

	l, err = l.Add("bad$label", "val")
	if err == nil {
		t.Fatal("should have failed due to bad label")
	}

	l, err = l.Add("label", "bad$val")
	if err == nil {
		t.Fatal("should have failed due to bad value")
	}

	check([]int{1, 2, 3})

	l, err = l.Add("label1", "newValue")
	if err != nil {
		t.Fatal("failed to update label1")
	}

	expected := "label1=newValue,label2=value2,label3=value3"
	str := l.String()
	if str != expected {
		t.Fatalf("expected String to return '%s', got '%s'", expected, str)
	}
}

func Test_LabelsBad(t *testing.T) {
	yml := `
label$1: value1
label2: value2
`
	err := yaml.Unmarshal([]byte(yml), &Labels{})
	if err == nil {
		t.Fatal("should have errored parsing yaml with bad label, but didn't")
	}
}

func Test_LabelsEmpty(t *testing.T) {
	yml := ""
	l := &Labels{}

	val := l.String()
	if val != "" {
		t.Fatalf("empty labels not empty string: '%s'", val)
	}

	err := yaml.Unmarshal([]byte(yml), l)
	if err != nil {
		t.Fatalf("errored parsing yaml: %s", err)
	}

	val = l.String()
	if val != "" {
		t.Fatalf("empty labels not empty string: '%s'", val)
	}
}

func Test_ArgoWorkflowParameters(t *testing.T) {
	expected := `parameter1="value1",parameter2="value\"with\"quotes"`
	yml := `
parameter1: value1
parameter2: value"with"quotes
`
	p := &ArgoWorkflowParameters{}

	val := p.StringArray()
	if len(val) != 0 {
		t.Fatalf("string array should be empty, got %s", val)
	}

	err := yaml.Unmarshal([]byte(yml), p)
	if err != nil {
		t.Fatalf("errored parsing yaml: %s", err)
	}

	val2 := strings.Join(p.StringArray(), ",")
	if val2 != expected {
		t.Fatalf("expected '%s', got '%s'", expected, val2)
	}
}

func Test_ArgoWorkflowParametersBad(t *testing.T) {
	yml := `
paramet%%er1: value1
parameter2: value"with"quotes
`

	err := yaml.Unmarshal([]byte(yml), &ArgoWorkflowParameters{})
	if err == nil {
		t.Fatal("should have errored parsing yaml with bad parameter name, but didn't")
	}
}

func Test_ArgoTemplate(t *testing.T) {
	kind := "workflowtemplate"
	name := "mirror-workflow"
	yml := fmt.Sprintf("%s/%s", kind, name)
	at := ArgoTemplate{}

	err := yaml.Unmarshal([]byte(yml), &at)
	if err != nil {
		t.Fatalf("errored parsing yaml: %s", err)
	}

	if at.Kind != kind {
		t.Fatalf("template kind '%s' expected, got '%s'", kind, at.Kind)
	}

	if at.Name != name {
		t.Fatalf("template name '%s' expected, got '%s'", name, at.Name)
	}
}

func Test_ArgoTemplateBad(t *testing.T) {
	badtemplates := []string{
		"badtemplate",
		"",
	}

	for _, template := range badtemplates {
		t.Run(
			template,
			func(t *testing.T) {
				err := yaml.Unmarshal(
					[]byte(fmt.Sprintf(`template: "%s"`, template)),
					&struct{ Template ArgoTemplate }{})
				if err == nil {
					t.Fatal("should have errored parsing yaml, but didn't")
				}
			},
		)
	}
}

// TODO: more test cases
func Test_ArgoWorkflowOpts1(t *testing.T) {
	yml := `
template: workflowtemplate/mirror-workflow
parameters:
  parameter1: value
labels:
  somelabel: somevalue
annotations:
  someannotation: somevalue
`
	awo := &ArgoWorkflowOpts{}

	err := yaml.Unmarshal([]byte(yml), awo)
	if err != nil {
		t.Fatalf("errored parsing yaml: %s", err)
	}

	_, err = awo.SubmitOptsGenerator()
	if err == nil {
		t.Fatal("should have errored calling SubmitOptsGenerator() per no workflow ID set")
	}

	_ = awo.SetWorkflowIdLabel("workflow")

	sog, err := awo.SubmitOptsGenerator()
	if err != nil {
		t.Fatalf("failed running SubmitOptsGenerator(): %s", err)
	}

	wfUuid := uuid.Must(uuid.NewV4())
	prio := 100
	so := sog(wfUuid, prio)

	if so.Name != wfUuid.String() {
		t.Fatalf("submitopts name should be '%s', got '%s'", wfUuid, so.Name)
	}

	params := `parameter1="value"`
	if len(so.Parameters) != 1 || so.Parameters[0] != params {
		t.Fatalf("submitopts params should be '%s', got '%s'", params, so.Parameters)
	}

	sa := ""
	if so.ServiceAccount != sa {
		t.Fatalf("submitopts serivce account should be '%s', got '%s'", sa, so.ServiceAccount)
	}

	labels := "somelabel=somevalue,swoop.element84.com/workflowId=workflow"
	if so.Labels != labels {
		t.Fatalf("submitopts labels should be '%s', got '%s'", labels, so.Labels)
	}

	anno := "someannotation=somevalue"
	if so.Annotations != anno {
		t.Fatalf("submitopts annotations should be '%s', got '%s'", anno, so.Annotations)
	}

	if int(*so.Priority) != prio {
		t.Fatalf("submitopts priority should be '%d', got '%d'", prio, so.Priority)
	}
}

func Test_ArgoWorkflowOpts2(t *testing.T) {
	yml := `
template: workflowtemplate/mirror-workflow
parameters:
  parameter1: value
annotations:
  someannotation: somevalue
`
	awo := &ArgoWorkflowOpts{}

	err := yaml.Unmarshal([]byte(yml), awo)
	if err != nil {
		t.Fatalf("errored parsing yaml: %s", err)
	}

	_, err = awo.SubmitOptsGenerator()
	if err == nil {
		t.Fatal("should have errored calling SubmitOptsGenerator() per no workflow ID set")
	}

	_ = awo.SetWorkflowIdLabel("workflow")

	labels := "swoop.element84.com/workflowId=workflow"
	if awo.Labels.String() != labels {
		t.Fatalf("labels should be '%s', got '%s'", labels, awo.Labels)
	}

	_, err = awo.SubmitOptsGenerator()
	if err != nil {
		t.Fatalf("failed running SubmitOptsGenerator(): %s", err)
	}
}
