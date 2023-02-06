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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

	// Get target CR triggered controller
	var job esphev1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		log.Error(err, "Unable to fetch Job")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	updateState := func(job *esphev1.Job, newState esphev1.JobState) error {
		job.Status.State = newState
		if err := r.Status().Update(ctx, job); err != nil {
			log.Error(err, "unable to update job state")
			return err
		}
		return nil
	}

	// Update stauts based current state
	switch job.Status.State {
	case esphev1.Running:
		// Turn into Finished or Error
		log.Info("current state Running")
		break
	case esphev1.Finished:
		log.Info("current state Finished")
		// Check the pod state related to this job
		var podlist corev1.PodList
		var pod *corev1.Pod = nil
		if err := r.List(ctx, &podlist); err != nil {
			log.Error(err, "Unable to list pods")
		} else {
			for _, item := range podlist.Items {
				if item.GetName() == job.Name {
					*pod = item
					break
				}
			}
		}

		// Pod not found
		if pod == nil {
			log.Info("No related pod found")
			updateState(&job, esphev1.Error)
			return reconcile.Result{}, client.IgnoreNotFound(nil)
		}

		// Check target pod status
		// Todo: assuming only one container
		if len(pod.Status.ContainerStatuses) != 1 {
			log.Info("pod has more than one container")
			return reconcile.Result{}, client.IgnoreNotFound(nil)
		}

		status := pod.Status.ContainerStatuses[0]
		if status.State.Terminated == nil {
			// not finished yet
			log.Info("container is still running")
			break
		}

		if status.State.Terminated.ExitCode == 0 {
			// Finished
			if err := updateState(&job, esphev1.Finished); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			log.Info(status.State.Terminated.Reason)
			if err := updateState(&job, esphev1.Error); err != nil {
				return ctrl.Result{}, err
			}
		}
		break
	case esphev1.Error:
		log.Info("current state Error")
		break
	default: // NotStarted Actually
		log.Info("current state NotStarted")
		// Execute target command with a new pod
		pod := NewPod(&job)
		_, errCreate := ctrl.CreateOrUpdate(ctx, r.Client, pod, func() error {
			return ctrl.SetControllerReference(&job, pod, r.Scheme)
		})
		if errCreate != nil {
			log.Error(errCreate, "Error creating pod")
			return ctrl.Result{}, nil
		}

		if err := updateState(&job, esphev1.Running); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esphev1.Job{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// NewPod creates a new pod for executing command of the job
func NewPod(job *esphev1.Job) *corev1.Pod {
	labels := map[string]string{
		"app": job.Name,
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: job.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "alpine",
					Image:   "alpine",
					Command: strings.Split(job.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
}
