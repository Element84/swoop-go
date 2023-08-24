package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gofrs/uuid/v5"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

const SwoopWorkflowIdLabelName = "swoop.element84.com/workflowId"

var paramNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+[-a-zA-Z0-9_]*$`)

type K8sOptions struct {
	Kubeconfig      string                     `yaml:"kubeconfig"`
	ConfigOverrides *clientcmd.ConfigOverrides `yaml:"configOverrides"`
}

type ArgoConf struct {
	InstanceId string      `yaml:"instanceId"`
	K8sOptions *K8sOptions `yaml:"k8sOptions"`
	// TODO: should this support "global" options? labels, service account, annotations?
}

func (ac *ArgoConf) GetConfig() clientcmd.ClientConfig {
	lrs := clientcmd.NewDefaultClientConfigLoadingRules()
	lrs.DefaultClientConfig = &clientcmd.DefaultClientConfig
	lrs.ExplicitPath = ac.K8sOptions.Kubeconfig
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		lrs,
		ac.K8sOptions.ConfigOverrides,
	)
}

func (ac *ArgoConf) GetNamespace() (string, error) {
	// if we support more than just a k8s connection here then this needs to be fancier
	// see https://github.com/argoproj/argo-workflows/blob/master/cmd/argo/commands/client/conn.go#L76
	namespace, _, err := ac.GetConfig().Namespace()
	return namespace, err
}

type Labels labels.Set

func validateLabel(key, val string) error {
	errors := validation.IsQualifiedName(key)
	if len(errors) > 0 {
		return fmt.Errorf("key is not valid: '%s'", errors)
	}

	errors = validation.IsValidLabelValue(val)
	if len(errors) > 0 {
		return fmt.Errorf("value is not valid: '%s'", errors)
	}

	return nil
}

func (l Labels) Validate() error {
	for label, val := range l {
		err := validateLabel(label, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l Labels) Get(key string) (string, bool) {
	val, ok := l[key]
	return val, ok
}

func (l Labels) Add(key, val string) (Labels, error) {
	err := validateLabel(key, val)
	if err != nil {
		return l, err
	}

	if l == nil {
		l = make(Labels)
	}

	l[key] = val
	return l, nil
}

func (l Labels) String() string {
	return labels.Set(l).String()
}

func (l *Labels) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p Labels

	err := unmarshal((*p)(l))
	if err != nil {
		return err
	}

	err = l.Validate()
	if err != nil {
		return err
	}

	return nil
}

type ArgoWorkflowParameters map[string]string

func (p ArgoWorkflowParameters) StringArray() (params []string) {
	for name, val := range p {
		params = append(
			params,
			// replace to escape `"` chars in val string
			fmt.Sprintf(`%s="%s"`, name, strings.ReplaceAll(val, `"`, `\"`)),
		)
	}

	sort.StringSlice(params).Sort()
	return params
}

func (p ArgoWorkflowParameters) Validate() error {
	for param := range p {
		if !paramNameRegex.MatchString(param) {
			return fmt.Errorf(
				"parameter name invalid: '%s'; valid chars are alphanumeric, '_', or '-'",
				param,
			)
		}
	}
	return nil
}

func (p *ArgoWorkflowParameters) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type t ArgoWorkflowParameters

	err := unmarshal((*t)(p))
	if err != nil {
		return err
	}

	err = p.Validate()
	if err != nil {
		return err
	}

	return nil
}

type ArgoTemplate struct {
	Kind string
	Name string
}

func (at *ArgoTemplate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	err := unmarshal(&s)
	if err != nil {
		return err
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf(
			"template name '%s' is malformed; should be `kind/name` like 'workflowtemplate/mirror-workflow'",
			s,
		)
	}

	at.Kind = parts[0]
	at.Name = parts[1]

	return nil
}

type ArgoWorkflowOpts struct {
	Template       *ArgoTemplate          `yaml:"template"`
	Parameters     ArgoWorkflowParameters `yaml:"parameters"`
	ServiceAccount string                 `yaml:"serviceAccount"`
	Labels         Labels                 `yaml:"labels"`
	Annotations    Labels                 `yaml:"annotations"`
}

func (awo *ArgoWorkflowOpts) SetWorkflowIdLabel(id string) error {
	labels, err := awo.Labels.Add(SwoopWorkflowIdLabelName, id)
	if err != nil {
		return err
	}
	awo.Labels = labels
	return nil
}

type ArgoSubmitOptsGenerator func(wfUuid uuid.UUID, priority int) *wfv1.SubmitOpts

func (awo *ArgoWorkflowOpts) SubmitOptsGenerator() (ArgoSubmitOptsGenerator, error) {
	_, ok := awo.Labels.Get(SwoopWorkflowIdLabelName)
	if !ok {
		return nil, fmt.Errorf(
			"workflow label unset; cannot make SubmitOpts generator until SetWorkflowIdLabel(id string) is called",
		)
	}

	params := awo.Parameters.StringArray()
	sa := awo.ServiceAccount
	labels := awo.Labels.String()
	anno := awo.Annotations.String()
	return func(wfUuid uuid.UUID, priority int) *wfv1.SubmitOpts {
		prio := int32(priority)
		return &wfv1.SubmitOpts{
			Name:           wfUuid.String(),
			Parameters:     params,
			ServiceAccount: sa,
			Labels:         labels,
			Annotations:    anno,
			Priority:       &prio,
		}
	}, nil
}

func (awo *ArgoWorkflowOpts) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p ArgoWorkflowOpts

	err := unmarshal((*p)(awo))
	if err != nil {
		return err
	}

	if awo.Template == nil {
		return errors.New("'template' is a required property and must be defined")
	}

	return nil
}
