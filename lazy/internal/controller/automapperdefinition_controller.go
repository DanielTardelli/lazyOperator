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
	"fmt"
	"slices"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

const (
	DefinitionFinalizer = "tardelli.com.au/automapperdefinition-cleanup"
)

func (r *AutoMapperDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var logger = log.FromContext(ctx)
	var cr tardellicomauv1alpha1.AutoMapperDefinition

	// Gets the definition of the resource and handles errors on retrieval
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("AutoMapperDefinition resource was not found. Ignoring...")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		}

		logger.Info("AutoMapperDefinition failed to be retrieved. Exiting...")
		return ctrl.Result{RequeueAfter: time.Minute * 20}, err
	}

	// for debugging
	logger.Info("DEBUG: Received AutoMapperDefinition",
		"name", cr.Name,
		"spec", fmt.Sprintf("%+v", cr.Spec),
		"relationships_count", len(cr.Spec.Relationships))

	// Ensures that finalizers are on the object and graceful deletion with finalizer
	// consideration
	if cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&cr, DefinitionFinalizer) {
			controllerutil.AddFinalizer(&cr, DefinitionFinalizer)
			if err := r.Update(ctx, &cr); err != nil {
				logger.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}

			// Return early — the object has been updated; we’ll reconcile again
			return ctrl.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&cr, DefinitionFinalizer) {
			// Perform cleanup (e.g., delete ScaledObjects, remove relationships)
			for _, rel := range cr.Status.Relationships {
				if err := AutoMapperRelationshipDelete(rel, r, ctx); err != nil {
					logger.Error(err, "Failed to clean up relationship")
					return ctrl.Result{}, err // try again later
				}
			}

			// Remove finalizer to allow actual deletion
			controllerutil.RemoveFinalizer(&cr, DefinitionFinalizer)
			if err := r.Update(ctx, &cr); err != nil {
				return ctrl.Result{}, err
			}

			// Stop reconciling, resource is about to be deleted
			return ctrl.Result{}, nil
		}
	}

	var creation = []tardellicomauv1alpha1.AutoMapperDefinitionObject{} // full definition
	var deletion = []tardellicomauv1alpha1.AutoMapperRelLocator{}       // just name, we can find ourselves

	for _, rel := range cr.Spec.Relationships {
		// Check if relationships is present, in the list if it is great, if not provision
		if !slices.ContainsFunc(cr.Status.Relationships, func(loc tardellicomauv1alpha1.AutoMapperRelLocator) bool {
			return rel.ObjectName == loc.ObjectName
		}) {
			creation = append(creation, rel)
		}
	}

	for _, cur := range cr.Status.Relationships {
		// checking current list against spec in case deletion is required
		if !slices.ContainsFunc(cr.Spec.Relationships, func(rel tardellicomauv1alpha1.AutoMapperDefinitionObject) bool {
			return rel.ObjectName == cur.ObjectName
		}) {
			deletion = append(deletion, cur)
		}
	}

	// Delete all the relationships we need to
	// need to consider error handling here
	for _, loc := range deletion {
		if err := AutoMapperRelationshipDelete(loc, r, ctx); err != nil {
			logger.Info("AutoMapperRelationshipDelete FAILED")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		}

		// Remove out of relationships so that the update function updates the right resources
		cr.Status.Relationships = slices.DeleteFunc(cr.Status.Relationships, func(rel tardellicomauv1alpha1.AutoMapperRelLocator) bool {
			return rel.ObjectName == loc.ObjectName
		})
	}

	// Update current
	for _, loc := range cr.Status.Relationships {
		if err := AutoMapperRelationshipUpdate(loc, r, ctx); err != nil {
			logger.Info("AutoMapperRelationshipUpdate FAILED")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		}
	}

	// Provision new
	for _, object := range creation {
		newRelationship, err := AutoMapperRelationshipCreate(object, r, ctx, &cr)
		if err != nil {
			logger.Info("AutoMapperRelationshipCreation FAILED")
			return ctrl.Result{RequeueAfter: time.Minute * 20}, err
		} else {
			cr.Status.Relationships = append(cr.Status.Relationships, tardellicomauv1alpha1.AutoMapperRelLocator{
				ObjectName: object.ObjectName,
				Name:       newRelationship.GetName(),
				Namespace:  newRelationship.GetNamespace(),
			})
		}
	}

	// Update the object with the new status
	if err := r.Status().Update(ctx, &cr); err != nil {
		logger.Error(err, "Failed to update AutoMapperDefinition status")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err
	}

	// Begin requeue timer
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoMapperDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tardellicomauv1alpha1.AutoMapperDefinition{}).
		Complete(r)
}
