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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	viewv1 "github.com/ystkfujii/playground_operator/api/v1"
)

var _ = Describe("AppDeployer Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		appdeployer := &viewv1.AppDeployer{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind AppDeployer")
			err := k8sClient.Get(ctx, typeNamespacedName, appdeployer)
			if err != nil && errors.IsNotFound(err) {
				resource := &viewv1.AppDeployer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
				}
				resource.Spec.Deployment.ImageName = "nginx:latest"
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &viewv1.AppDeployer{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance AppDeployer")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &AppDeployerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the AppDeployer status conditions are set correctly")
			Eventually(func() error {
				err := k8sClient.Get(ctx, typeNamespacedName, appdeployer)
				if err != nil {
					return err
				}
				return nil
			}).Should(Succeed())

			Expect(appdeployer.Status.Conditions).To(HaveLen(2))

			By("Checking Ready condition")
			readyCondition := findCondition(appdeployer.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.Reason).To(Equal("ReconcileSuccessful"))
			Expect(readyCondition.Message).To(Equal("AppDeployer reconciled successfully"))

			By("Checking DeploymentReady condition")
			deploymentReadyCondition := findCondition(appdeployer.Status.Conditions, "DeploymentReady")
			Expect(deploymentReadyCondition).NotTo(BeNil())
			Expect(deploymentReadyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(deploymentReadyCondition.Reason).To(Equal("DeploymentCreated"))
			Expect(deploymentReadyCondition.Message).To(Equal("Deployment has been created and is ready"))

			By("Verifying ServiceAccountName is set")
			Expect(appdeployer.Status.ServiceAccountName).To(Equal(resourceName))
		})

		It("should maintain conditions across multiple reconciliation cycles", func() {
			By("Reconciling the created resource multiple times")
			controllerReconciler := &AppDeployerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Get the resource after first reconciliation
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, appdeployer)
			}).Should(Succeed())

			firstReconcileConditions := make([]metav1.Condition, len(appdeployer.Status.Conditions))
			copy(firstReconcileConditions, appdeployer.Status.Conditions)

			// Second reconciliation
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Get the resource after second reconciliation
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, appdeployer)
			}).Should(Succeed())

			By("Verifying conditions are still present and consistent")
			Expect(appdeployer.Status.Conditions).To(HaveLen(2))

			readyCondition := findCondition(appdeployer.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))

			deploymentReadyCondition := findCondition(appdeployer.Status.Conditions, "DeploymentReady")
			Expect(deploymentReadyCondition).NotTo(BeNil())
			Expect(deploymentReadyCondition.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})

// Helper function to find a condition by type
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
