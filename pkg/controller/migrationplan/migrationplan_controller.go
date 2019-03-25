/*
Copyright 2019 redhat.

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

package migrationplan

import (
	"context"
	"fmt"
	migrationv1beta1 "migration/pkg/apis/migration/v1beta1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MigrationPlan Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMigrationPlan{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type PlanUpdatedPredicate struct {
	predicate.Funcs
}

func (r PlanUpdatedPredicate) Update(e event.UpdateEvent) bool {
	planOld, cast := e.ObjectOld.(*migrationv1beta1.MigrationPlan)
	if !cast {
		return true
	}
	planNew, cast := e.ObjectNew.(*migrationv1beta1.MigrationPlan)
	if !cast {
		return true
	}
	changed := !reflect.DeepEqual(planOld.Spec, planNew.Spec)
	return changed
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("migrationplan-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MigrationPlan
	err = c.Watch(
		&source.Kind{
			Type: &migrationv1beta1.MigrationPlan{},
		},
		&handler.EnqueueRequestForObject{},
		&PlanUpdatedPredicate{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by MigrationPlan - change this for objects you create
	err = c.Watch(
		&source.Kind{
			Type: &appsv1.Deployment{},
		},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &migrationv1beta1.MigrationPlan{},
		})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMigrationPlan{}

// ReconcileMigrationPlan reconciles a MigrationPlan object
type ReconcileMigrationPlan struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a MigrationPlan object and makes changes based on the state read
// and what is in the MigrationPlan.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=migration.openshit.io,resources=migrationplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=migration.openshit.io,resources=migrationplans/status,verbs=get;update;patch
func (r *ReconcileMigrationPlan) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	plan := &migrationv1beta1.MigrationPlan{}
	err := r.Get(context.TODO(), request.NamespacedName, plan)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			fmt.Printf("Plan: %s, deleted\n", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = r.reconcilePlan(plan)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileDeployment(plan)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileService(plan)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMigrationPlan) reconcilePlan(plan *migrationv1beta1.MigrationPlan) error {
	log.Info("reconcilePlan()")
	fmt.Printf("*** Thing=%s Count=%d\n", plan.Spec.Thing, plan.Status.Reconciled)
	plan.Status.Reconciled++
	err := r.Update(context.TODO(), plan)
	if err != nil {
		fmt.Println("Increment failed.")
	}
	return err
}

func (r *ReconcileMigrationPlan) reconcileDeployment(plan *migrationv1beta1.MigrationPlan) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plan.Name + "-deployment",
			Namespace: plan.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": plan.Name + "-deployment",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": plan.Name + "-deployment",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "bitnami/nginx",
						},
					},
				},
			},
		},
	}

	log.Info("reconcileDeployment()")

	found := &appsv1.Deployment{}
	err := r.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      dep.Name,
			Namespace: dep.Namespace,
		},
		found)
	if err == nil {
		// update as needed.
		dirty := false
		if !reflect.DeepEqual(found.Spec.Template.Spec, dep.Spec.Template.Spec) {
			found.Spec.Template.Spec = dep.Spec.Template.Spec
			dirty = true
		}
		if !reflect.DeepEqual(found.Spec.Selector, dep.Spec.Selector) {
			found.Spec.Selector = dep.Spec.Selector
			dirty = true
		}
		if dirty {
			log.Info("Updating Deployment", "namespace", dep.Namespace, "name", dep.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if errors.IsNotFound(err) {
		// create
		log.Info("Creating Deployment", "namespace", dep.Namespace, "name", dep.Name)
		err = r.Create(context.TODO(), dep)
		if err != nil {
			return err
		}
		err := controllerutil.SetControllerReference(plan, dep, r.scheme)
		if err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func (r *ReconcileMigrationPlan) reconcileService(plan *migrationv1beta1.MigrationPlan) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plan.Name + "-service",
			Namespace: plan.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"deployment": plan.Name + "-deployment",
			},
			Ports: []corev1.ServicePort{
				{Port: 8888},
			},
		},
	}

	log.Info("reconcileService()")

	found := &corev1.Service{}
	err := r.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		found)
	if err == nil {
		// update as needed.
		dirty := false
		if !reflect.DeepEqual(found.Spec.Selector, service.Spec.Selector) {
			found.Spec.Selector = service.Spec.Selector
			dirty = true
		}
		if !reflect.DeepEqual(found.Spec.Ports, service.Spec.Ports) {
			found.Spec.Ports = service.Spec.Ports
			dirty = true
		}
		if dirty {
			log.Info("Updating Service", "namespace", service.Namespace, "name", service.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if errors.IsNotFound(err) {
		// create
		log.Info("Creating Service", "namespace", service.Namespace, "name", service.Name)
		err = r.Create(context.TODO(), service)
		if err != nil {
			return err
		}
		err := controllerutil.SetControllerReference(plan, service, r.scheme)
		if err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}
