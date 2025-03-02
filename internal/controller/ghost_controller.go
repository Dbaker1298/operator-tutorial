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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	blogv1 "example.com/api/v1"
)

// GhostReconciler reconciles a Ghost object
type GhostReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	recoder record.EventRecorder
}

const (
	pvcNamePrefix        = "ghost-data-pvc-"
	deploymentNamePrefix = "ghost-deployment-"
	svcNamePrefix        = "ghost-service-"
)

// +kubebuilder:rbac:groups=blog.my.domain,resources=ghosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=blog.my.domain,resources=ghosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=blog.my.domain,resources=ghosts/finalizers,verbs=update

func (r *GhostReconciler) addPvcIfNotExists(ctx context.Context, ghost *blogv1.Ghost) error {
	log := log.FromContext(ctx)

	pvc := &corev1.PersistentVolumeClaim{}
	team := ghost.ObjectMeta.Namespace
	pvcName := pvcNamePrefix + team

	err := r.Get(ctx, client.ObjectKey{Namespace: ghost.ObjectMeta.Namespace, Name: pvcName}, pvc)

	if err == nil {
		// PVC already exists, nothing to do
		return nil
	}

	// PVC does not exist, create it
	desiredPVC := generateDesiredPVC(ghost, pvcName)
	if err := controllerutil.SetControllerReference(ghost, desiredPVC, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference on PVC")
		return err
	}

	if err := r.Create(ctx, desiredPVC); err != nil {
		log.Error(err, "Failed to create PVC")
		return err
	}

	r.recoder.Event(ghost, corev1.EventTypeNormal, "PVCReady", "PVC created successfully")
	log.Info("PVC created successfully", "pvc", pvcName)
	return nil
}

func generateDesiredPVC(ghost *blogv1.Ghost, pvcName string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: ghost.ObjectMeta.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ghost object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *GhostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	ghost := &blogv1.Ghost{}
	if err := r.Get(ctx, req.NamespacedName, ghost); err != nil {
		log.Error(err, "Faled to get Ghost")
	}

	log.Info("Reconciling Ghost", "imageTag", ghost.Spec.ImageTag, "team", ghost.ObjectMeta.Namespace)
	log.Info("Reconciliation complete")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GhostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recoder = mgr.GetEventRecorderFor("ghost-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&blogv1.Ghost{}).
		Named("ghost").
		Complete(r)
}
