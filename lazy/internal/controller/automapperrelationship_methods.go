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
	"fmt"
	"strings"

	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
	errorsv1 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// for tmp use then to use proper crd structs
type AutoMapperTmpMap struct {
	Source *unstructured.Unstructured
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
			// if not found maybe user has pre-emptively deleted, so we only wanna throw
			// error on case where the error is not to do with not being found
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
	if err := r.Delete(ctx, cr); err != nil {
		return err
	} else {
		return nil
	}
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

// this is for safely updating an object including its annotations, versions etc
func AutoMapperRelationshipSafeUpdate(ctx context.Context, r *AutoMapperDefinitionReconciler, obj *unstructured.Unstructured) error {
	// Get the current resource to obtain the latest resource version
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())

	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	if err := r.Get(ctx, key, existing); err != nil {
		return err
	}

	// Preserve important metadata from existing resource
	obj.SetResourceVersion(existing.GetResourceVersion())
	obj.SetUID(existing.GetUID())
	obj.SetCreationTimestamp(existing.GetCreationTimestamp())

	// Merge annotations (preserve existing, add new)
	existingAnnotations := existing.GetAnnotations()
	newAnnotations := obj.GetAnnotations()
	if existingAnnotations == nil {
		existingAnnotations = make(map[string]string)
	}
	if newAnnotations != nil {
		for k, v := range newAnnotations {
			existingAnnotations[k] = v
		}
	}
	obj.SetAnnotations(existingAnnotations)

	// Merge labels (preserve existing, add new)
	existingLabels := existing.GetLabels()
	newLabels := obj.GetLabels()
	if existingLabels == nil {
		existingLabels = make(map[string]string)
	}
	if newLabels != nil {
		for k, v := range newLabels {
			existingLabels[k] = v
		}
	}
	obj.SetLabels(existingLabels)

	// Perform the update with retry logic
	return r.Update(ctx, obj)
}

func AutoMapperRelationshipResourceHandler(operation string, ctx context.Context, src, dst *unstructured.Unstructured, varmaps []tardellicomauv1alpha1.AutoMapperVariableMap, r *AutoMapperDefinitionReconciler, rel *tardellicomauv1alpha1.AutoMapperRelationship) error {
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
			if err := AutoMapperRelationshipSetValue(dst, val, mapping.DestinationVar); err != nil {
				return err
			}
		} else if mapping.Type == "referenced" {
			// if referenced sourceVar is path
			val, err := AutoMapperRelationshipGetValue(src, mapping.SourceVar)
			if err != nil {
				return err
			} else {
				if err := AutoMapperRelationshipSetValue(dst, val, mapping.DestinationVar); err != nil {
					return err
				}
			}
		}
	}

	// must ensure that the namespace of the dst resource matches the relationships to
	// set ownership or operator throws error
	dst.SetNamespace(rel.GetNamespace())

	if operation == "create" {
		// if we're creating from scratch, ensure that linking labels are present
		labels := make(map[string]string)
		labels["app.kubernetes.io/managed-by"] = "lazy"
		labels["lazy/link"] = src.GetNamespace() + "." + src.GetName()

		dst.SetLabels(safeMergeLabels(labels, dst.GetLabels()))

		if err := controllerutil.SetOwnerReference(rel, dst, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, dst); err != nil {
			return err
		}
	} else if operation == "update" {
		if err := AutoMapperRelationshipSafeUpdate(ctx, r, dst); err != nil {
			return err
		}
	}

	return nil
}

func AutoMapperRelationshipClusterResourceDiscoverer(basis string, label *tardellicomauv1alpha1.AutoMapperDefinitionLabel, namespace *string, src, dst tardellicomauv1alpha1.AutoMapperDefinitionGVK, r *AutoMapperDefinitionReconciler, ctx context.Context) ([]AutoMapperTmpMap, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   src.Group,
		Version: src.Version,
		Kind:    src.Kind + "List",
	})

	var opts client.ListOptions
	if basis == "namespace" {
		if namespace == nil {
			return nil, fmt.Errorf("basis is namespace but no namespace specified")
		}
		opts = client.ListOptions{
			Namespace: *namespace,
		}
	} else if basis == "label" {
		if label == nil {
			return nil, fmt.Errorf("basis is label but no labels specified")
		}
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
			Kind:    dst.Kind + "List",
		})
		subopts := client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app.kubernetes.io/managed-by": "lazy",
				"lazy/link":                    namespace + "." + name,
			}),
		}

		// keep in mind if err is not nil, cos this is list operation it doesnt mean that the rsrc
		// wasn't found, smth else went wrong
		if err := r.List(ctx, sublist, &subopts); err != nil {
			mappings = append(mappings, AutoMapperTmpMap{
				Source: &rsrc,
				Result: nil,
				Error:  err,
			})
		} else if len(sublist.Items) > 1 {
			err := errors.New("when getting item by label to establish relationships, found one-to-many relationship")
			mappings = append(mappings, AutoMapperTmpMap{
				Source: &rsrc,
				Result: nil,
				Error:  err,
			})
		} else if len(sublist.Items) == 0 {
			err := errorsv1.NewNotFound(schema.GroupResource{
				Group:    dst.Group,
				Resource: "resource",
			}, "resource")
			mappings = append(mappings, AutoMapperTmpMap{
				Source: &rsrc,
				Result: nil,
				Error:  err,
			})
		} else {
			mappings = append(mappings, AutoMapperTmpMap{
				Source: &rsrc,
				Result: &sublist.Items[0],
			})
		}
	}

	return mappings, nil
}

const (
	RelationshipFinalizer = "tardelli.com.au/automapperrelationship-cleanup"
)

func AutoMapperRelationshipUpdate(loc tardellicomauv1alpha1.AutoMapperRelLocator, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	// Grab the CR
	cr, err := getRelationship(r, ctx, loc.Name, loc.Namespace)
	if err != nil {
		return errors.New("failed to get relationship when attempting to update")
	}

	// check if the relationship has been deleted as a resource, if it has
	// then deal with accordingly
	var rel tardellicomauv1alpha1.AutoMapperRelationship
	nsName := types.NamespacedName{
		Name:      loc.Name,
		Namespace: loc.Namespace,
	}

	if err := r.Get(ctx, nsName, &rel); err != nil {
		return err
	}

	// Ensures that finalizers are on the object and graceful deletion with finalizer
	// consideration
	if rel.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&rel, RelationshipFinalizer) {
			controllerutil.AddFinalizer(&rel, RelationshipFinalizer)
			if err := r.Update(ctx, &rel); err != nil {
				return err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&rel, RelationshipFinalizer) {
			if err := AutoMapperRelationshipDelete(loc, r, ctx); err != nil {
				return err // try again later
			}

			// Remove finalizer to allow actual deletion
			controllerutil.RemoveFinalizer(&rel, RelationshipFinalizer)
			if err := r.Update(ctx, &rel); err != nil {
				return err
			}

			// return as we have deleted the inners of the relationship
			return nil
		}
	}

	clusterCurrentMappings, err := AutoMapperRelationshipClusterResourceDiscoverer(cr.Spec.Basis, cr.Spec.Label, cr.Spec.Namespace, cr.Spec.Source, cr.Spec.Result, r, ctx)
	if err != nil {
		return err
	}

	// first need to get the status and iterate over, check if the status src resources are present in
	// the current cluster audit, if they aren't make sure that the resource that was provisioned as a
	// result is also deleted, then move through the current audit and update each definition/create if
	// not already existing

	previousList := cr.Status.Resources

	// Handles deletions by checking if the source is not longer present in the cluster
	for _, obj := range previousList {
		found := false
		for _, obj2 := range clusterCurrentMappings {
			autoMapperRepresentation := tardellicomauv1alpha1.AutoMapperRelationshipStatusResource{
				Namespace: obj2.Source.GetNamespace(),
				Name:      obj2.Source.GetName(),
				GVK: tardellicomauv1alpha1.AutoMapperDefinitionGVK{
					Group:   obj2.Source.GetObjectKind().GroupVersionKind().Group,
					Version: obj2.Source.GetObjectKind().GroupVersionKind().Version,
					Kind:    obj2.Source.GetObjectKind().GroupVersionKind().Kind,
				},
			}

			if obj.Source == autoMapperRepresentation {
				found = true
				break
			}
		}

		if !found {
			rsrc, err := AutoMapperRelationshipGetUnstructuredResource(obj.Result, r, ctx)
			if errorsv1.IsNotFound(err) {
				continue
			} else if err != nil {
				return err
			}
			if err := AutoMapperRelationshipResourceHandler("delete", ctx, nil, rsrc, *cr.Spec.VarMap, r, nil); err != nil {
				return err
			}
		}
	}

	filteredList := []tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings{}

	for _, obj := range clusterCurrentMappings {
		if obj.Error != nil && errorsv1.IsNotFound(obj.Error) {
			// Create the object from scratch in line with provided gvk
			newRsrc := &unstructured.Unstructured{}
			newRsrc.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   cr.Spec.Result.Group,
				Version: cr.Spec.Result.Version,
				Kind:    cr.Spec.Result.Kind,
			})

			if err := AutoMapperRelationshipResourceHandler("create", ctx, obj.Source, newRsrc, *cr.Spec.VarMap, r, &rel); err != nil {
				return err
			}

			res, err := toStatusResource(obj.Source, newRsrc)
			if err != nil {
				return err
			}
			filteredList = append(filteredList, *res)
		} else if obj.Error != nil && !errorsv1.IsNotFound(obj.Error) {
			// return error
			return obj.Error
		} else {
			// Update the object
			if err := AutoMapperRelationshipResourceHandler("update", ctx, obj.Source, obj.Result, *cr.Spec.VarMap, r, &rel); err != nil {
				return err
			}

			res, err := toStatusResource(obj.Source, obj.Result)
			if err != nil {
				return err
			}
			filteredList = append(filteredList, *res)
		}
	}

	cr.Status.Resources = filteredList

	if err := r.Status().Update(ctx, cr); err != nil {
		return err
	}

	return nil
}

func AutoMapperRelationshipCreate(obj tardellicomauv1alpha1.AutoMapperDefinitionObject, r *AutoMapperDefinitionReconciler, ctx context.Context, amd *tardellicomauv1alpha1.AutoMapperDefinition) (*tardellicomauv1alpha1.AutoMapperRelationship, error) {

	// we need to build the cr from scratch including the spec and everything, once we've built the cr from scratch
	// then we can continue with normal cluster mappings etc

	cr := &tardellicomauv1alpha1.AutoMapperRelationship{}
	cr.Spec = tardellicomauv1alpha1.AutoMapperRelationshipSpec{
		Source:    obj.Source,
		Result:    obj.Result,
		Basis:     obj.Basis,
		VarMap:    obj.VarMap,
		Label:     obj.Label,
		Namespace: obj.Namespace,
	}

	cr.Status = tardellicomauv1alpha1.AutoMapperRelationshipStatus{
		Resources: []tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings{},
	}

	// set namesapce of the relationship to the same as the definition or it throws error
	// for setting ownership
	cr.SetGenerateName("automapperrelationship-")
	cr.SetNamespace(amd.GetNamespace())

	// First create the resource in the cluster so it gets uid
	if err := r.Create(ctx, cr); err != nil {
		return nil, err
	}

	// then set reference
	if err := controllerutil.SetOwnerReference(amd, cr, r.Scheme); err != nil {
		return nil, err
	}

	// then update
	if err := r.Update(ctx, cr); err != nil {
		return nil, err
	}

	clusterCurrentMappings, err := AutoMapperRelationshipClusterResourceDiscoverer(cr.Spec.Basis, cr.Spec.Label, cr.Spec.Namespace, cr.Spec.Source, cr.Spec.Result, r, ctx)
	if err != nil {
		return nil, err
	}

	filteredList := []tardellicomauv1alpha1.AutoMapperRelationshipStatusMappings{}

	for _, obj := range clusterCurrentMappings {
		if obj.Error != nil && errorsv1.IsNotFound(obj.Error) {
			// Create the object from scratch in line with provided gvk
			newRsrc := &unstructured.Unstructured{}
			newRsrc.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   cr.Spec.Result.Group,
				Version: cr.Spec.Result.Version,
				Kind:    cr.Spec.Result.Kind,
			})

			if err := AutoMapperRelationshipResourceHandler("create", ctx, obj.Source, newRsrc, *cr.Spec.VarMap, r, cr); err != nil {
				return nil, err
			}

			res, err := toStatusResource(obj.Source, newRsrc)
			if err != nil {
				return nil, err
			}
			filteredList = append(filteredList, *res)
		} else if obj.Error != nil && !errorsv1.IsNotFound(obj.Error) {
			// return error
			return nil, obj.Error
		} else {
			// Update the object
			if err := AutoMapperRelationshipResourceHandler("update", ctx, obj.Source, obj.Result, *cr.Spec.VarMap, r, cr); err != nil {
				return nil, err
			}

			res, err := toStatusResource(obj.Source, obj.Result)

			if err != nil {
				return nil, err
			}

			filteredList = append(filteredList, *res)
		}
	}

	cr.Status.Resources = filteredList

	if err := r.Status().Update(ctx, cr); err != nil {
		return nil, err
	}

	return cr, nil
}
