package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/mhrabovcin/troubleshoot-live/pkg/utils"
)

// LoadResourcesFromFile tries to k8s API resources from a given file. It supports
// resources stored as List kind, YAML array of separate resources, JSON array of
// resources and JSON stored item list without TypeMeta information.
// The result will be returned as `UnstructuredList` but the items could be missing
// GVK information. It is up to caller to add GVK to each item before further
// processing.
func LoadResourcesFromFile(bundle afero.Fs, path string) (*unstructured.UnstructuredList, error) {
	data, err := afero.ReadFile(bundle, path)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(path, ".json") {
		return parseJSONList(data, path)
	}

	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		list := &unstructured.UnstructuredList{}
		err := yaml.Unmarshal(data, list)
		if err == nil {
			return list, nil
		}

		items := []unstructured.Unstructured{}
		if iErr := yaml.Unmarshal(data, &items); err != nil {
			errs := []error{err, iErr}
			return nil, fmt.Errorf("failed to load resources from YAML file %q with errors: %w", path, errors.Join(errs...))
		}
		list.Items = items
		return list, nil
	}

	return nil, fmt.Errorf("unsupported data format")
}

func parseJSONList(data []byte, path string) (*unstructured.UnstructuredList, error) {
	list := &unstructured.UnstructuredList{}
	// Format:
	// - stored as unstructured.UnstructedList and items contain GVK info
	err := json.Unmarshal(data, list)
	if err == nil {
		return list, nil
	}
	errs := []error{err}

	// Format:
	// - no GVK info in objects
	// [ {}, {}, ... {} ]
	items := []map[string]any{}
	if secondErr := json.Unmarshal(data, &items); secondErr != nil {
		errs = append(errs, secondErr)
	} else {
		for _, item := range items {
			list.Items = append(list.Items, unstructured.Unstructured{Object: item})
		}
		return list, nil
	}

	// Format:
	// - similar to unstructured list but objects do not contain GVK info
	// {
	//	"metadata": { ... }
	//  "items": [ {}, {}, ... {}]
	// }
	//
	untypedList := struct {
		Items []map[string]any `json:"items"`
	}{}
	if thirdErr := json.Unmarshal(data, &untypedList); thirdErr != nil {
		errs = append(errs, thirdErr)
	} else {
		for _, item := range untypedList.Items {
			list.Items = append(list.Items, unstructured.Unstructured{Object: item})
		}
		return list, nil
	}

	for i := range errs {
		errs[i] = utils.MaxErrorString(errs[i], 200)
	}

	return nil, fmt.Errorf("failed to load resources from JSON file %q with errors: %w", path, errors.Join(errs...))
}

// cmOrSecret represents a special data structure that troubleshoot uses for
// storing secrets and configmaps.
type cmOrSecret struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	// configmaps include data but secrets data are not included in the bundle.
	Data map[string]string `json:"data,omitempty"`
}

// LoadConfigMap loads configmap data from special struct that support-bundle
// uses to store CMs in.
func LoadConfigMap(bundle afero.Fs, path string) (*unstructured.Unstructured, error) {
	data, err := afero.ReadFile(bundle, path)
	if err != nil {
		return nil, err
	}

	cmStruct := cmOrSecret{}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &cmStruct); err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(data, &cmStruct); err != nil {
			return nil, err
		}
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmStruct.Name,
			Namespace: cmStruct.Namespace,
		},
		Data: cmStruct.Data,
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: u}, nil
}

// LoadSecret loads secret from special struct that support-bundle
// uses to store Secrets in. It leaves the data empty.
func LoadSecret(bundle afero.Fs, path string) (*unstructured.Unstructured, error) {
	data, err := afero.ReadFile(bundle, path)
	if err != nil {
		return nil, err
	}

	secretData := cmOrSecret{}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &secretData); err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(data, &secretData); err != nil {
			return nil, err
		}
	}

	cm := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretData.Name,
			Namespace: secretData.Namespace,
		},
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: u}, nil
}
