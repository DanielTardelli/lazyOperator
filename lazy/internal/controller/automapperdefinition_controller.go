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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tardellicomauv1alpha1 "github.com/DanielTardelli/lazyOperator/api/v1alpha1"
)

// AutoMapperDefinitionReconciler reconciles a AutoMapperDefinition object
type AutoMapperDefinitionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tardelli.com.au,resources=automapperdefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tardelli.com.au,resources=automapperdefinitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tardelli.com.au,resources=automapperdefinitions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AutoMapperDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AutoMapperDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var logger = log.FromContext(ctx)

	// Logic:
	// 1. Use current state of the AutoMapperDefinition to create list of desired relationships
	logger.Info("1. Use current state of the AutoMapperDefinition to create list of desired relationships")
	var cr tardellicomauv1alpha1.AutoMapperDefinition

	// NOTE: Gets the definition of the resource and handles errors on retrieval
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("AutoMapperDefinition resource was not found. Ignoring...")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		}

		logger.Info("AutoMapperDefinition failed to be retrieved. Exiting...")
		return ctrl.Result{RequeueAfter: time.Minute * 20}, err
	}

	// Prepping for status
	var requestedRelationships = make(map[string]tardellicomauv1alpha1.AutoMapperDefinitionStatusMapped)

	for i, rel := range cr.Spec.Maps {
		logger.Info("Processing relationship", "count", i)

		// need to extract these as they may or may not exist
		var basisVars []string

		// conditionally populate both if pointer != nil
		if rel.Basis == "label" && rel.LabelBasis != nil {
			basisVars = *rel.LabelBasis
		} else if rel.Basis == "namespace" && rel.NamespaceBasis != nil {
			basisVars = *rel.NamespaceBasis
		} else {
			basisVars = nil
		}

		// form object as per status
		hashedRelationship := RelationshipHash(rel.Source, rel.Result, rel.Basis, basisVars)

		// append to map
		requestedRelationships[hashedRelationship] = tardellicomauv1alpha1.AutoMapperDefinitionStatusMapped{
			Source:    rel.Source,
			Result:    rel.Result,
			Basis:     rel.Basis,
			BasisVars: basisVars,
			Resources: make(map[string]tardellicomauv1alpha1.AutoMapperDefinitionStatusResource),
			Created:   false,
		}
	}

	// 2. Compare against status list
	logger.Info("2. Compare against status list")
	var presentRelationships = cr.Status.MappedItems
	var deletionQueue = []tardellicomauv1alpha1.AutoMapperDefinitionStatusMapped{}

	// 	2.1. If status item NOT in AutoMapperDefinition: queue to delete
	for key, value := range presentRelationships {
		_, exists := requestedRelationships[key]
		if !exists {
			deletionQueue = append(deletionQueue, value)
		} else {
			// assign current status to the requestedRelationships map
			// which will become our new status
			requestedRelationships[key] = presentRelationships[key]
		}
	}

	// 3. Remove all resources in deletion list
	logger.Info("3. Remove all resources in deletion list")

	for _, item := range deletionQueue {
		for _, resource := range item.Resources {
			object := DynamicObjectRetrieve(r.Client, resource.Name, resource.Namespace, resource.GVK.Group, resource.GVK.Version, resource.GVK.Kind)

			// if cant find resource throw error
			if object == nil {
				logger.Info("Error in retrieving resource from cluster, please ensure args are correct", "name", resource.Name, "namespace", resource.Namespace,
					"group", resource.GVK.Group, "version", resource.GVK.Version, "kind", resource.GVK.Kind)
				return ctrl.Result{RequeueAfter: time.Minute * 20}, errors.New("could not find resource for deletion")
			}

			// If error in deletion throw error
			if err := r.Delete(ctx, object); err != nil {
				logger.Info("Error in deleting associated resources", "name", resource.Name, "namespace", resource.Namespace)
				return ctrl.Result{RequeueAfter: time.Minute * 20}, err
			}
		}
	}

	// 4. Run reconciliation of past relationships against present state of cluster
	logger.Info("4. Run reconciliation of past relationships against present state of cluster")

	for _, item := range requestedRelationships {
		// We will be dealing with these after, for now just reconciling present relationships
		if !item.Created {
			continue
		}

		gvk := schema.GroupVersionKind{Group: item.Source.Group, Version: item.Source.Version, Kind: item.Source.Kind}
		objList := unstructuredv1.UnstructuredList{}
		objList.SetGroupVersionKind(gvk)

		var listopts []client.ListOption

		// need to change LabelBasis to be a map from string to string,
		// need to change namespace basis to be a SINGLE namespace
		if item.Basis == "label" {
			listopts = []client.ListOption{
				client.MatchingLabels(item.BasisVars),
			}
		} else if item.Basis == "namespace" {
			listopts = []client.ListOption{
				client.InNamespace(item.BasisVars),
			}
		} else {
			listopts = []client.ListOption{}
		}

		// this is cool, its a variadic parameter the ... meaning that the function
		// can accept however many of that type that we wanna pass in
		if err := r.List(ctx, &objList, listopts...); err != nil {
			logger.Info("Error retrieving resources to reconcile status with spec, exiting...")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		}

		// Now need to go through each resource in the requests,
		// FOR EACH, check if requested is in current, if it is, check if var mapping is correct, else provision from scratch
		// make sure each one in the current that we find in the requested is removed from the current list
		// then at the end remove everything in the current list as its no longer required

	}
	// 5. Create resultant objects

	return ctrl.Result{RequeueAfter: time.Minute * 20}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoMapperDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tardellicomauv1alpha1.AutoMapperDefinition{}).
		Complete(r)
}
