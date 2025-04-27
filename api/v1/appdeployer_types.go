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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDeployerSpec defines the desired state of AppDeployer.
type AppDeployerSpec struct {
	// +optional
	ServiceAccountName string         `json:"serviceAccountName"`
	Deployment         DeploymentSpec `json:"deployment"`
}

type DeploymentSpec struct {
	ImageName string `json:"imageName"`

	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
}

// AppDeployerStatus defines the observed state of AppDeployer.
type AppDeployerStatus struct {
	ServiceAccountName string `json:"serviceAccountName"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ServiceAccountName",type="string",JSONPath=".spec.serviceAccountName"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deployment.replicas"

// AppDeployer is the Schema for the appdeployers API.
type AppDeployer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppDeployerSpec   `json:"spec,omitempty"`
	Status AppDeployerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppDeployerList contains a list of AppDeployer.
type AppDeployerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppDeployer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppDeployer{}, &AppDeployerList{})
}
