/*
Copyright 2023.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	esphev1 "demo/api/v1"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=esphe.esphe,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=esphe.esphe,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=esphe.esphe,resources=jobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Controller get a job to handle")

	// 1. Get target CR triggered controller
	var job esphev1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		log.Error(err, "Unable to fetch Job")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: 2. Execute target command
	log.Info(job.Spec.Command)

	// TODO: 3. Update job status
	job.Status.State = esphev1.Running
	if err := r.Status().Update(ctx, &job); err != nil {
		log.Error(err, "unable to update job state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esphev1.Job{}).
		Complete(r)
}
