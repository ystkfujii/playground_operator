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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	viewv1 "github.com/ystkfujii/playground_operator/api/v1"
)

// MarkdownViewReconciler reconciles a MarkdownView object
type MarkdownViewReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=markdownviews,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=markdownviews/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=markdownviews/finalizers,verbs=update
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=appdeployers,verbs=get;list;watch
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=appdeployers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MarkdownView object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *MarkdownViewReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("start reconcile MarkdownView")

	var markdownView viewv1.MarkdownView
	if err := r.Get(ctx, req.NamespacedName, &markdownView); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch MarkdownView")
		return ctrl.Result{}, err
	}

	// Find related AppDeployers in the same namespace
	if err := r.updateRelatedAppDeployerStatus(ctx, markdownView); err != nil {
		logger.Error(err, "unable to update related AppDeployer status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MarkdownViewReconciler) updateRelatedAppDeployerStatus(ctx context.Context, markdownView viewv1.MarkdownView) error {
	logger := logf.FromContext(ctx).WithValues("markdownview", markdownView.Name, "namespace", markdownView.Namespace)

	// List all AppDeployers in the same namespace
	var appDeployerList viewv1.AppDeployerList
	if err := r.List(ctx, &appDeployerList, client.InNamespace(markdownView.Namespace)); err != nil {
		return err
	}

	for _, appDeployer := range appDeployerList.Items {
		// Update each AppDeployer's status with MarkdownView information
		if err := r.updateAppDeployerStatusWithMarkdownInfo(ctx, appDeployer, markdownView); err != nil {
			logger.Error(err, "failed to update AppDeployer status", "appDeployer", appDeployer.Name)
			continue
		}
		logger.Info("updated AppDeployer status with MarkdownView info", "appDeployer", appDeployer.Name)
	}

	return nil
}

func (r *MarkdownViewReconciler) updateAppDeployerStatusWithMarkdownInfo(ctx context.Context, appDeployer viewv1.AppDeployer, markdownView viewv1.MarkdownView) error {
	logger := logf.FromContext(ctx).WithValues("appDeployer", appDeployer.Name, "markdownview", markdownView.Name)

	// Create new condition based on MarkdownView state
	newCondition := metav1.Condition{
		Type:               "MarkdownViewProcessed",
		Status:             metav1.ConditionTrue,
		Reason:             "MarkdownViewReconciled",
		Message:            "MarkdownView " + markdownView.Name + " has been processed",
		LastTransitionTime: metav1.NewTime(time.Now()),
	}

	// Create a status-only patch object for Server-Side Apply
	gvk := schema.GroupVersionKind{
		Group:   "view.ystkfujii.github.io",
		Version: "v1",
		Kind:    "AppDeployer",
	}

	// Get current conditions
	currentConditions := make([]interface{}, 0)
	if len(appDeployer.Status.Conditions) > 0 {
		// Convert existing conditions but filter out the one we're updating
		for _, condition := range appDeployer.Status.Conditions {
			if condition.Type != "MarkdownViewProcessed" {
				currentConditions = append(currentConditions, map[string]interface{}{
					"type":               condition.Type,
					"status":             string(condition.Status),
					"reason":             condition.Reason,
					"message":            condition.Message,
					"lastTransitionTime": condition.LastTransitionTime.Format(time.RFC3339),
				})
			}
		}
	}

	// Add the new condition
	currentConditions = append(currentConditions, map[string]interface{}{
		"type":               newCondition.Type,
		"status":             string(newCondition.Status),
		"reason":             newCondition.Reason,
		"message":            newCondition.Message,
		"lastTransitionTime": newCondition.LastTransitionTime.Format(time.RFC3339),
	})

	patch := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvk.GroupVersion().String(),
			"kind":       gvk.Kind,
			"metadata": map[string]interface{}{
				"name":      appDeployer.Name,
				"namespace": appDeployer.Namespace,
			},
			"status": map[string]interface{}{
				"serviceAccountName": appDeployer.Status.ServiceAccountName,
				"conditions":         currentConditions,
			},
		},
	}
	patch.SetGroupVersionKind(gvk)

	if err := r.Status().Patch(ctx, patch, client.Apply, &client.SubResourcePatchOptions{
		PatchOptions: client.PatchOptions{
			FieldManager: "markdownview-controller",
			Force:        ptr.To(true),
		},
	}); err != nil {
		logger.Error(err, "unable to apply AppDeployer status")
		return err
	}

	logger.Info("AppDeployer status updated successfully using Server-Side Apply")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MarkdownViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&viewv1.MarkdownView{}).
		Watches(
			&viewv1.AppDeployer{},
			handler.EnqueueRequestsFromMapFunc(r.findMarkdownViewsForAppDeployer),
		).
		Named("markdownview").
		Complete(r)
}

// findMarkdownViewsForAppDeployer maps AppDeployer changes to MarkdownView reconcile requests
func (r *MarkdownViewReconciler) findMarkdownViewsForAppDeployer(ctx context.Context, obj client.Object) []reconcile.Request {
	appDeployer := obj.(*viewv1.AppDeployer)

	// List all MarkdownViews in the same namespace as the AppDeployer
	var markdownViewList viewv1.MarkdownViewList
	if err := r.List(ctx, &markdownViewList, client.InNamespace(appDeployer.Namespace)); err != nil {
		// Log error but return empty slice to avoid blocking
		return []reconcile.Request{}
	}

	// Create reconcile requests for all MarkdownViews in the namespace
	requests := make([]reconcile.Request, len(markdownViewList.Items))
	for i, markdownView := range markdownViewList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      markdownView.Name,
				Namespace: markdownView.Namespace,
			},
		}
	}

	return requests
}
