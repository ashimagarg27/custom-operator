/*
Copyright 2021.

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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	replicav1alpha1 "github.com/ashimagarg27/custom-operator/api/v1alpha1"
)

// CustomOperatorReconciler reconciles a CustomOperator object
type CustomOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=replica.example.com,resources=customoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=replica.example.com,resources=customoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=replica.example.com,resources=customoperators/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;get;list;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CustomOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *CustomOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// your logic here
	instance := &replicav1alpha1.CustomOperator{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("CustomOperator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object.
		logger.Error(err, "failed to get CustomOperator resource")
		return ctrl.Result{}, err
	}

	deploymentName := "custom-operator-deployment"
	deploymentNamespace := req.Namespace

	// Check if the deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: deploymentNamespace}, deployment)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Define a new deployment
			deploy := r.createDeployment(instance, deploymentName, deploymentNamespace)
			logger.Info("Creating a new Deployment", "Deployment.Namespace", deploy.Namespace, "Deployment.Name", deploy.Name)
			err = r.Create(ctx, deploy)
			if err != nil {
				logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deploy.Namespace, "Deployment.Name", deploy.Name)
				return ctrl.Result{}, err
			}
			logger.Info("Deployment created successfully!!")
			return ctrl.Result{Requeue: true}, nil
		} else {
			logger.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if replica count is same as in CR spec, if not match it
	replicas := &instance.Spec.Replicas
	if deployment.Spec.Replicas != replicas {
		deployment.Spec.Replicas = replicas
		err = r.Update(ctx, deployment)
		if err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}
		logger.Info("Deployment has created desired number of replicas of pod")
	} else {
		logger.Info("Deployment already has desired number of replicas of pod")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&replicav1alpha1.CustomOperator{}).
		Complete(r)
}

// createDeployment ...
func (r *CustomOperatorReconciler) createDeployment(instance *replicav1alpha1.CustomOperator, name string, namespace string) *appsv1.Deployment {
	replicas := &instance.Spec.Replicas
	lables := labelsForCustomOperator(instance.Name)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: lables,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lables,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "nginx:latest",
							Name:  "nginx-image",
							Ports: []corev1.ContainerPort{
								{
									Name:          "nginx",
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set CustomOperator instance as the owner and controller
	controllerutil.SetControllerReference(instance, deploy, r.Scheme)
	return deploy
}

// labelsForCustomOperator returns the labels for selecting the resources
// belonging to the given customoperator CR name.
func labelsForCustomOperator(name string) map[string]string {
	return map[string]string{"app": "replicas", "customoperator_cr": name}
}
