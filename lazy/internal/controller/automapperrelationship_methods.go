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

	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

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

func AutoMapperRelationshipDelete(loc tardellicomauv1alpha1.AutoMapperRelLocator, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	// Grab the CR
	cr, err := getRelationship(r, ctx, loc.Name, loc.Namespace)
	if err != nil {
		return errors.New("Failed to get relationship in AutoMapperRelationshipDelete")
	}
	// go through all resources and delete
	print(cr)
	return nil
}

func AutoMapperRelationshipUpdate(loc tardellicomauv1alpha1.AutoMapperRelLocator, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	// Grab the CR
	cr, err := getRelationship(r, ctx, loc.Name, loc.Namespace)
	if err != nil {
		return errors.New("Failed to get relationship in AutoMapperRelationshipUpdate")
	}
	print(cr)
	return nil
}

func AutoMapperRelationshipCreate(object tardellicomauv1alpha1.AutoMapperDefinitionObject, r *AutoMapperDefinitionReconciler, ctx context.Context) error {
	return nil
}
