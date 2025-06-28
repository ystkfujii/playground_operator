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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	poappv1 "github.com/ystkfujii/playground_operator/api/v1"
)

// AppDeployerReconciler reconciles a AppDeployer object
type AppDeployerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=appdeployers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=appdeployers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=view.ystkfujii.github.io,resources=appdeployers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppDeployer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *AppDeployerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("start reconcile AppDeployer")

	var appDeployer poappv1.AppDeployer
	if err := r.Get(ctx, req.NamespacedName, &appDeployer); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch AppDeployer", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if !appDeployer.DeletionTimestamp.IsZero() {
		logger.Info("AppDeployer is being deleted", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	if err := r.reconcileServiceAccount(ctx, appDeployer); err != nil {
		logger.Error(err, "unable to reconcile ServiceAccount")
		return ctrl.Result{}, err
	}

	if err := r.reconcileConfigMap(ctx, appDeployer); err != nil {
		logger.Error(err, "unable to reconcile ConfigMap")
		return ctrl.Result{}, err
	}
	if err := r.reconcileDeployment(ctx, appDeployer); err != nil {
		logger.Error(err, "unable to reconcile Deployment")
		return ctrl.Result{}, err
	}

	return r.updateStatus(ctx, appDeployer)
}

func (r *AppDeployerReconciler) reconcileServiceAccount(ctx context.Context, ad poappv1.AppDeployer) error {
	logger := logf.FromContext(ctx).WithValues("name", ad.Name, "namespace", ad.Namespace)

	newSAName := getServiceAccountName(ctx, ad)

	if newSAName == ad.Status.ServiceAccountName {
		return nil
	}

	if ad.Status.ServiceAccountName != "" {
		oldSA := &corev1.ServiceAccount{}
		oldSA.SetName(ad.Status.ServiceAccountName)
		oldSA.SetNamespace(ad.Namespace)

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, oldSA, func() error {
			_ = controllerutil.RemoveOwnerReference(&ad, oldSA, r.Scheme)
			return nil
		})
		if err != nil {
			logger.Error(err, "unable to create or update old ServiceAccount")
			return err
		}
		if op != controllerutil.OperationResultNone {
			logger.Info("reconcile ServiceAccount successfully", "op", op)
		}
	}
	// if metadata length is 0,  need to Delete old ServiceAccount ?

	newSA := &corev1.ServiceAccount{}
	newSA.SetNamespace(ad.Namespace)
	newSA.SetName(newSAName)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, newSA, func() error {
		return controllerutil.SetOwnerReference(&ad, newSA, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "unable to create or update new ServiceAccount")
		return err
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ServiceAccount successfully", "op", op)
	}

	return nil
}

func (r *AppDeployerReconciler) reconcileConfigMap(ctx context.Context, ad poappv1.AppDeployer) error {
	logger := logf.FromContext(ctx).WithValues("name", ad.Name, "namespace", ad.Namespace)

	cm := &corev1.ConfigMap{}
	cm.SetNamespace(ad.Namespace)
	cm.SetName(ad.Name)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["index.html"] = "<html><body><h1>Hello from ConfigMap!</h1></body></html>"
		return ctrl.SetControllerReference(&ad, cm, r.Scheme)
	})

	if err != nil {
		logger.Error(err, "unable to create or update ConfigMap")
		return err
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ConfigMap successfully", "op", op)
	}

	return nil
}

func (r *AppDeployerReconciler) reconcileDeployment(ctx context.Context, ad poappv1.AppDeployer) error {
	logger := logf.FromContext(ctx).WithValues("name", ad.Name, "namespace", ad.Namespace)

	var depName string
	if ad.Spec.ServiceAccountName == "" {
		depName = "deployer-shared"
	} else {
		depName = "deployer-" + ad.Spec.ServiceAccountName
	}

	owner, err := controllerReference(ad, r.Scheme)
	if err != nil {
		return err
	}

	dep := appsv1apply.Deployment(depName, ad.Namespace).
		WithLabels(map[string]string{
			"app.kubernetes.io/name":       "playground-operator",
			"app.kubernetes.io/created-by": "app-deployer-controller",
		}).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(ad.Spec.Deployment.Replicas).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(map[string]string{
				"app.kubernetes.io/name":       "playground-operator",
				"app.kubernetes.io/created-by": "app-deployer-controller",
			})).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(map[string]string{
					"app.kubernetes.io/name":       "playground-operator",
					"app.kubernetes.io/created-by": "app-deployer-controller",
				}).
				WithSpec(corev1apply.PodSpec().
					WithContainers(corev1apply.Container().
						WithName("app").
						WithImage(ad.Spec.Deployment.ImageName).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithVolumeMounts(corev1apply.VolumeMount().
							WithName("config-volume").
							WithMountPath("/usr/share/nginx/html"),
						).
						WithPorts(corev1apply.ContainerPort().
							WithName("http").
							WithProtocol(corev1.ProtocolTCP).
							WithContainerPort(80),
						).
						WithLivenessProbe(corev1apply.Probe().
							WithHTTPGet(corev1apply.HTTPGetAction().
								WithPort(intstr.FromString("http")).
								WithPath("/").
								WithScheme(corev1.URISchemeHTTP),
							),
						).
						WithReadinessProbe(corev1apply.Probe().
							WithHTTPGet(corev1apply.HTTPGetAction().
								WithPort(intstr.FromString("http")).
								WithPath("/").
								WithScheme(corev1.URISchemeHTTP),
							),
						),
					).
					WithServiceAccountName(getServiceAccountName(ctx, ad)).
					WithVolumes(corev1apply.Volume().
						WithName("config-volume").
						WithConfigMap(corev1apply.ConfigMapVolumeSource().
							WithName(ad.Name),
						),
					),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: ad.Namespace, Name: depName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := appsv1apply.ExtractDeployment(&current, "app-deployer-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "app-deployer-controller",
		Force:        ptr.To(true),
	})

	if err != nil {
		logger.Error(err, "unable to create or update Deployment")
		return err
	}

	logger.Info("reconcile Deployment successfully", "name", ad.Name)

	return nil
}

func (r *AppDeployerReconciler) updateStatus(ctx context.Context, ad poappv1.AppDeployer) (ctrl.Result, error) {
	logger := logf.FromContext(ctx).WithValues("name", ad.Name, "namespace", ad.Namespace)

	newServiceAccountName := getServiceAccountName(ctx, ad)

	newConditions := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "ReconcileSuccessful",
			Message:            "AppDeployer reconciled successfully",
			LastTransitionTime: metav1.NewTime(time.Now()),
		},
		{
			Type:               "DeploymentReady",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentCreated",
			Message:            "Deployment has been created and is ready",
			LastTransitionTime: metav1.NewTime(time.Now()),
		},
	}

	if ad.Status.ServiceAccountName == newServiceAccountName && conditionsEqual(ad.Status.Conditions, newConditions) {
		return ctrl.Result{}, nil
	}

	// Create a status-only patch object for Server-Side Apply
	gvk := ad.GroupVersionKind()
	if gvk.Empty() {
		// Set default GVK if not available
		gvk = schema.GroupVersionKind{
			Group:   "view.ystkfujii.github.io",
			Version: "v1",
			Kind:    "AppDeployer",
		}
	}

	patch := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvk.GroupVersion().String(),
			"kind":       gvk.Kind,
			"metadata": map[string]interface{}{
				"name":      ad.Name,
				"namespace": ad.Namespace,
			},
			"status": map[string]interface{}{
				"serviceAccountName": newServiceAccountName,
				"conditions":         convertConditionsToUnstructured(newConditions),
			},
		},
	}
	patch.SetGroupVersionKind(gvk)

	if err := r.Status().Patch(ctx, patch, client.Apply, &client.SubResourcePatchOptions{
		PatchOptions: client.PatchOptions{
			FieldManager: "app-deployer-controller",
			Force:        ptr.To(true),
		},
	}); err != nil {
		logger.Error(err, "unable to apply AppDeployer status")
		return ctrl.Result{}, err
	}

	logger.Info("AppDeployer status updated successfully using Server-Side Apply")
	return ctrl.Result{}, nil
}

func conditionsEqual(a, b []metav1.Condition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].Status != b[i].Status ||
			a[i].Reason != b[i].Reason || a[i].Message != b[i].Message {
			return false
		}
	}
	return true
}

func convertConditionsToUnstructured(conditions []metav1.Condition) []interface{} {
	result := make([]interface{}, len(conditions))
	for i, condition := range conditions {
		result[i] = map[string]interface{}{
			"type":               condition.Type,
			"status":             string(condition.Status),
			"reason":             condition.Reason,
			"message":            condition.Message,
			"lastTransitionTime": condition.LastTransitionTime.Format(time.RFC3339),
		}
	}
	return result
}

func getServiceAccountName(_ context.Context, ad poappv1.AppDeployer) string {
	if ad.Spec.ServiceAccountName == "" {
		return ad.Name
	}
	return ad.Spec.ServiceAccountName
}

func controllerReference(ad poappv1.AppDeployer, scheme *runtime.Scheme) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&ad, scheme)
	if err != nil {
		return nil, err
	}
	ref := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(ad.Name).
		WithUID(ad.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)

	return ref, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppDeployerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&poappv1.AppDeployer{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Named("appdeployer").
		Complete(r)
}
