package bundle

import (
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// CAPI cluster resource
// KIND kubeadm config
func DetectServiceSubnetRange(b Bundle) (string, error) {
	list, err := LoadResourcesFromFile(b, filepath.Join(b.Layout().ClusterResources(), "pods", "kube-system.json"))
	if err != nil {
		return "", err
	}

	for i := range list.Items {
		if !isKubeApiserverPod(&list.Items[i]) {
			continue
		}

		return parseIpRangeArg(&list.Items[i])
	}

	return "", nil
}

func parseIpRangeArg(u *unstructured.Unstructured) (string, error) {
	pod := &corev1.Pod{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &pod); err != nil {
		return "", err
	}

	for _, c := range pod.Spec.Containers {
		if c.Name != "kube-apiserver" {
			continue
		}

		for _, arg := range c.Command {
			if strings.HasPrefix(arg, "--service-cluster-ip-range=") {
				return strings.TrimPrefix(arg, "--service-cluster-ip-range="), nil
			}
		}
	}

	return "", nil
}

func isKubeApiserverPod(u *unstructured.Unstructured) bool {
	if !strings.HasPrefix(u.GetName(), "kube-apiserver-") {
		return false
	}

	labels := u.GetLabels()
	return labels["component"] == "kube-apiserver"
}