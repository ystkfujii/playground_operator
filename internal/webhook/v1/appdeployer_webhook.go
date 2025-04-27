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

package v1

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apiv1 "github.com/ystkfujii/playground_operator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var appdeployerlog = logf.Log.WithName("appdeployer-resource")

// SetupappdeployerWebhookWithManager registers the webhook for appdeployer in the manager.
func SetupAppdeployerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&apiv1.AppDeployer{}).
		WithValidator(&AppDeployerCustomValidator{}).
		WithDefaulter(&AppDeployerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-view-ystkfujii-github-io-v1-appdeployer,mutating=true,failurePolicy=fail,sideEffects=None,groups=view.ystkfujii.github.io,resources=appdeployers,verbs=create;update,versions=v1,name=mappdeployer-v1.kb.io,admissionReviewVersions=v1

type AppDeployerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &AppDeployerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AppDeployer.
func (d *AppDeployerCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	AppDeployer, ok := obj.(*apiv1.AppDeployer)

	if !ok {
		return fmt.Errorf("expected an AppDeployer object but got %T", obj)
	}
	appdeployerlog.Info("Defaulting for AppDeployer", "name", AppDeployer.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// +kubebuilder:webhook:path=/validate-view-ystkfujii-github-io-v1-appdeployer,mutating=false,failurePolicy=fail,sideEffects=None,groups=view.ystkfujii.github.io,resources=appdeployers,verbs=create;update,versions=v1,name=vappdeployer-v1.kb.io,admissionReviewVersions=v1

// AppDeployerCustomValidator struct is responsible for validating the AppDeployer resource
// when it is created, updated, or deleted.
type AppDeployerCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &AppDeployerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AppDeployer.
func (v *AppDeployerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	appDeployer, ok := obj.(*apiv1.AppDeployer)
	if !ok {
		return nil, fmt.Errorf("expected a AppDeployer object but got %T", obj)
	}
	appdeployerlog.Info("Validation for AppDeployer upon creation", "name", appDeployer.GetName())
	appDeployer.Spec.Deployment.Replicas = 3

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AppDeployer.
func (v *AppDeployerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var errs field.ErrorList
	appDeployer, ok := newObj.(*apiv1.AppDeployer)
	if !ok {
		return nil, fmt.Errorf("expected a AppDeployer object for the newObj but got %T", newObj)
	}
	appdeployerlog.Info("Validation for AppDeployer upon update", "name", appDeployer.GetName())

	if strings.Contains(appDeployer.Spec.ServiceAccountName, "deny") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("serviceAccountName"), appDeployer.Spec.ServiceAccountName, "cannot be updated"))
	}

	if len(errs) > 0 {
		err := apierrors.NewInvalid(schema.GroupKind{Group: "view.ystkfujii.github.io", Kind: "MarkdownView"}, appDeployer.Name, errs)
		markdownviewlog.Error(err, "validation error", "name", appDeployer.Name)
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AppDeployer.
func (v *AppDeployerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	AppDeployer, ok := obj.(*apiv1.AppDeployer)
	if !ok {
		return nil, fmt.Errorf("expected a AppDeployer object but got %T", obj)
	}
	appdeployerlog.Info("Validation for AppDeployer upon deletion", "name", AppDeployer.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
