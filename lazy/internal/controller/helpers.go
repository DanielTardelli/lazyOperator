package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// For converting the requested relationships to hash for uniqueness
func RelationshipHash(source tardellicomauv1alpha1.AutoMapperDefinitionGVK, result tardellicomauv1alpha1.AutoMapperDefinitionGVK,
	basis string, basisVars []string) string {
	var basisVarsConcat string
	// _ is for unused vars in Go
	for _, str := range basisVars {
		basisVarsConcat += str
	}

	concatenated := source.Kind + "|" + result.Kind + "|" + basis + "|" + basisVarsConcat
	hash := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(hash[:])
}

// Factory method for getting object constructor from resource type
func DynamicObjectRetrieve(c client.Client, name, namespace, group, version, kind string) *unstructuredv1.Unstructured {
	gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}

	// Create unstructured object
	obj := unstructuredv1.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	key := client.ObjectKey{Namespace: namespace, Name: name}

	if err := c.Get(context.TODO(), key, &obj); err != nil {
		return nil
	} else {
		return &obj
	}
}
