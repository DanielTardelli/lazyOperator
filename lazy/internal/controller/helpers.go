package controller

import (
	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// for taking two unstructured objects and converting to standardised object
// for comparison
func toStatusResource(src, res *unstructured.Unstructured) (*tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings, error) {
	convert := func(obj *unstructured.Unstructured) tardellicomauv1alpha1.AutoMapperRelationshipStatusResource {
		return tardellicomauv1alpha1.AutoMapperRelationshipStatusResource{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
			GVK: tardellicomauv1alpha1.AutoMapperDefinitionGVK{
				Group:   obj.GetObjectKind().GroupVersionKind().Group,
				Version: obj.GetObjectKind().GroupVersionKind().Version,
				Kind:    obj.GetObjectKind().GroupVersionKind().Kind,
			},
		}
	}

	return &tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings{
		Source: convert(src),
		Result: convert(res),
	}, nil
}

// for merging labels that already exist on an unstructured object with the labels
// we require for our relationship and resource location
func safeMergeLabels(customLabels, srcLabels map[string]string) map[string]string {
	if srcLabels == nil {
		return customLabels
	}

	for key, val := range customLabels {
		_, exists := srcLabels[key]
		if !exists {
			srcLabels[key] = val
		}
	}

	return srcLabels
}
