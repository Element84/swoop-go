package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	test "github.com/element84/swoop-go/pkg/utils/testing"
)

func TestNamespaceAndConfigFlags(
	ctx context.Context,
	t testing.TB,
	prefix string,
) *genericclioptions.ConfigFlags {
	configFlags := genericclioptions.NewConfigFlags(false)
	kc := test.PathFromRoot(t, "kubeconfig.yaml")
	namespace := strings.ToLower(prefix + t.Name())
	configFlags.KubeConfig = &kc
	configFlags.Namespace = &namespace

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		t.Fatalf("failed to build k8s config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to get k8s client: %s", err)
	}

	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	t.Cleanup(func() {
		_ = clientset.CoreV1().Namespaces().Delete(
			context.Background(), namespace, metav1.DeleteOptions{},
		)
	})

	_, err = clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to get make k8s test namespace: %s", err)
	}

	return configFlags
}
