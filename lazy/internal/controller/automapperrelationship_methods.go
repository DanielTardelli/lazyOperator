/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"strings"

	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
	errorsv1 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// for tmp use then to use proper crd structs
type AutoMapperTmpMap struct {
	Source unstructured.Unstructured
	Result *unstructured.Unstructured
	Error  error
}

func getRelationship(r *AutoMapperDefinitionReconciler, ctx context.Context, name, namespace string) (*tardellicomauv1alpha1.AutoMapperRelationship, error) {
	cr := &tardellicomauv1alpha1.AutoMapperRelationship{}

	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := r.Get(ctx, namespacedName, cr); err != nil {
		return nil, err
	}

	return cr, nil
}

func AutoMapperRelationshipGetUnstructuredResource(obj tardellicomauv1alpha1.AutoMapperRelationshipStatusResource, r *AutoMapperDefinitionReconciler, ctx context.Context) (*unstructured.Unstructured, error) {
	// Form GVK
	gvk := schema.GroupVersionKind{
		Group:   obj.GVK.Group,
		Version: obj.GVK.Version,
		Kind:    obj.GVK.Kind,
	}

	// Form key
	key := types.NamespacedName{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}

	rsrc := &unstructured.Unstructured{}
	rsrc.SetGroupVersionKind(gvk)

	if err := r.Get(ctx, key, rsrc); err != nil {
		return nil, err
	} else {
		return rsrc, nil
	}
}

func AutoMapperRelationshipDelete(loc tardellicomauv1alpha1.AutoMapperRelLocator, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	// Grab the CR
	cr, err := getRelationship(r, ctx, loc.Name, loc.Namespace)

	if err != nil {
		return errors.New("failed to get relationship when attempting to delete")
	}

	list := cr.Status.Resources

	for _, obj := range list {
		// Form GVK
		obj, err := AutoMapperRelationshipGetUnstructuredResource(obj.Result, r, ctx)

		if err != nil {
			// if not found maybe user has pre-emptively deleted
			if !errorsv1.IsNotFound(err) {
				return err
			}
		} else {
			// delete if can get
			if err := r.Delete(ctx, obj); err != nil {
				return err
			}
		}
	}

	// if no errors and we deleted everything can safely remove relationship
	err := deleteRelationship(r, ctx, loc.Name, loc.Namespace)

	return nil
}

// Gets values from unstructured objects
func AutoMapperRelationshipGetValue(obj *unstructured.Unstructured, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	val, found, err := unstructured.NestedFieldNoCopy(obj.Object, parts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("unable to locate nested field in get value func")
	}
	return val, nil
}

// Sets values in unstructured objects
func AutoMapperRelationshipSetValue(obj *unstructured.Unstructured, variable interface{}, path string) error {
	parts := strings.Split(path, ".")
	if err := unstructured.SetNestedField(obj.Object, variable, parts...); err != nil {
		return err
	} else {
		return nil
	}
}

func AutoMapperRelationshipResourceHandler(operation string, ctx context.Context, src, dst *unstructured.Unstructured, varmaps []tardellicomauv1alpha1.AutoMapperVariableMap, r *AutoMapperDefinitionReconciler) error {
	// if we're deleting, just delete straight off the bat
	if operation == "delete" {
		err := r.Delete(ctx, dst)
		return err
	}

	// otherwise worry about the mappings etc
	for _, mapping := range varmaps {
		if mapping.Type == "static" {
			// if static sourceVar is the value
			val := mapping.SourceVar
			AutoMapperRelationshipSetValue(dst, val, mapping.DestinationVar)
		} else if mapping.Type == "Referenced" {
			// if referenced sourceVar is path
			val, err := AutoMapperRelationshipGetValue(src, mapping.SourceVar)
			if err != nil {
				return err
			} else {
				AutoMapperRelationshipSetValue(dst, val, mapping.DestinationVar)
			}
		}
	}

	if operation == "create" {
		// if we're creating from scratch
		if err := r.Create(ctx, dst); err != nil {
			return err
		}
	} else if operation == "update" {
		// if we're updating a resource
		if err := r.Update(ctx, dst); err != nil {
			return err
		}
	}

	return nil
}

func AutoMapperRelationshipClusterResourceDiscoverer(basis string, label tardellicomauv1alpha1.AutoMapperDefinitionLabel, namespace string, src, dst tardellicomauv1alpha1.AutoMapperDefinitionGVK, r *AutoMapperDefinitionReconciler, ctx context.Context) ([]AutoMapperTmpMap, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   src.Group,
		Version: src.Version,
		Kind:    src.Kind + "List",
	})

	var opts client.ListOptions
	if basis == "namespace" {
		opts = client.ListOptions{
			Namespace: namespace,
		}
	} else if basis == "label" {
		opts = client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				label.Key: label.Value,
			}),
		}
	} else {
		opts = client.ListOptions{
			Namespace: "",
		}
	}

	if err := r.List(ctx, list, &opts); err != nil {
		return nil, err
	}

	var mappings []AutoMapperTmpMap

	for _, rsrc := range list.Items {
		name := rsrc.GetName()
		namespace := rsrc.GetNamespace()
		sublist := &unstructured.UnstructuredList{}
		sublist.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   dst.Group,
			Version: dst.Version,
			Kind:    dst.Kind,
		})
		subopts := client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app.kubernetes.io/managed-by": "lazy",
				"lazy/link":                    namespace + " - " + name,
			}),
		}

		if err := r.List(ctx, sublist, &subopts); err != nil {
			mappings = append(mappings, AutoMapperTmpMap{
				Source: rsrc,
				Result: nil,
				Error:  err,
			})
		} else if len(sublist.Items) > 1 {
			err := errors.New("when getting item by label to establish relationships, found one-to-many relationship")
			mappings = append(mappings, AutoMapperTmpMap{
				Source: rsrc,
				Result: nil,
				Error:  err,
			})
		} else if len(sublist.Items) == 0 {
			err := errorsv1.NewNotFound(schema.GroupResource{
				Group:    dst.Group,
				Resource: "resource",
			}, "resource")
			mappings = append(mappings, AutoMapperTmpMap{
				Source: rsrc,
				Result: nil,
				Error:  err,
			})
		} else {
			mappings = append(mappings, AutoMapperTmpMap{
				Source: rsrc,
				Result: &sublist.Items[0],
			})
		}
	}

	return mappings, nil
}

func AutoMapperRelationshipUpdate(loc tardellicomauv1alpha1.AutoMapperRelLocator, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	// Grab the CR
	cr, err := getRelationship(r, ctx, loc.Name, loc.Namespace)
	if err != nil {
		return errors.New("failed to get relationship when attempting to update")
	}

	clusterCurrentMappings, err := AutoMapperRelationshipClusterResourceDiscoverer(cr.Spec.Basis, cr.Spec.Labels, cr.Spec.Namespaces, cr.Spec.Source, r, ctx)
	if err != nil {
		return err
	}

	filteredList := []tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings{}

	// first need to get the status and iterate over, check if the status src resources are present in
	// the current cluster audit, if they aren't make sure that the resource that was provisioned as a
	// result is also deleted, then move through the current audit and update each definition/create if
	// not already existing

	for _, obj := range clusterCurrentMappings {
		// if we cant grab the src, nothing should exist on the result side
		if obj.Source.Object != nil && errorsv1.IsNotFound(errSrc) {
			if errDst != nil && !errorsv1.IsNotFound(errDst) {
				// if grabbing the result resource failed and its not cause it wasn't found then raise alarm
				return errDst
			} else if errDst == nil {
				// if src is not able to be grabbed and the dst stil exists then we need to remove the dst
				if err := AutoMapperRelationshipResourceHandler("delete", ctx, src, dst, *cr.Spec.VarMap, r); err != nil {
					return err
				}
			}
		} else if errSrc != nil && !errorsv1.IsNotFound(errSrc) {
			// if src has error and its not cos it cant be found then we got a doozy
			return errSrc
		} else if errSrc == nil {
			// now to cases whereby we could find the source object
			if errDst != nil && !errorsv1.IsNotFound(errDst) {
				// if the error isnt that it cant be found then we have an issue
				return errDst
			} else if errDst != nil && errorsv1.IsNotFound(errDst) {
				// if error is that dst cant be found then provision it
				if err := AutoMapperRelationshipResourceHandler("create", ctx, src, dst, *cr.Spec.VarMap, r); err != nil {
					return err
				}
				filteredList = append(filteredList, obj)
			} else if errDst == nil {
				if err := AutoMapperRelationshipResourceHandler("update", ctx, src, dst, *cr.Spec.VarMap, r); err != nil {
					return err
				}
				filteredList = append(filteredList, obj)
			}
		}
	}

	return nil
}

func AutoMapperRelationshipCreate(object tardellicomauv1alpha1.AutoMapperDefinitionObject, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	return nil
}
