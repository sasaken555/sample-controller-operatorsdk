package foo

import (
	"context"

	samplecontrollerv1alpha1 "github.com/sasaken555/sample-controller-operatorsdk/pkg/apis/samplecontroller/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_foo")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Foo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFoo{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("foo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Foo
	err = c.Watch(&source.Kind{Type: &samplecontrollerv1alpha1.Foo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner Foo
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &samplecontrollerv1alpha1.Foo{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileFoo implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileFoo{}

// ReconcileFoo reconciles a Foo object
type ReconcileFoo struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Foo object and makes changes based on the state read
// and what is in the Foo.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileFoo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Foo")
	ctx := context.Background()

	// (1) Fetch Foo Object
	foo := &samplecontrollerv1alpha1.Foo{}
	if err := r.client.Get(ctx, request.NamespacedName, foo); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Foo not found. Ignore not Found.")
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "Failed to get Foo.")
		return reconcile.Result{}, err
	}

	// (2) Delete old Deployment owned by Foo
	if err := r.cleanupOwnedResources(ctx, foo); err != nil {
		reqLogger.Error(err, "Failed to cleanup old Deployment resources for this Foo")
		return reconcile.Result{}, err
	}

	// (3) Create Deployment if not exist
	deploymentName := foo.Spec.DeploymentName
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: request.Namespace,
		},
	}

	if err := r.client.Get(ctx, client.ObjectKey{Namespace: foo.Namespace, Name: deploymentName}, deployment); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Cloud not find existing Deployment for Foo, Creating one...")
			deployment = newDeployment(foo)

			if err := r.client.Create(ctx, deployment); err != nil {
				reqLogger.Error(err, "Failed to create Deployment resource.")
				return reconcile.Result{}, err
			}

			reqLogger.Info("Created Deployment resource for Foo.")
			return reconcile.Result{}, nil
		}

		reqLogger.Error(err, "Failed to get Deployment resource.")
		return reconcile.Result{}, err
	}

	// (4) Compare Deployment spec with Foo spec, adjust specs if not match
	if foo.Spec.Replicas != nil && *foo.Spec.Replicas != *deployment.Spec.Replicas {
		reqLogger.Info("Unmatch spec", "foo.spec.replicas", foo.Spec.Replicas, "deployment.spec.replicas", deployment.Spec.Replicas)
		reqLogger.Info("Deployment replicas is not equal to Foo replicas. Reconcile this.")

		if err := r.client.Update(ctx, newDeployment(foo)); err != nil {
			reqLogger.Error(err, "Faild to update Deployment spec for Foo.")
			return reconcile.Result{}, err
		}

		reqLogger.Info("Updated Deployment spec for Foo.")
		return reconcile.Result{}, nil
	}

	// (5) Update Foo's status
	if foo.Status.AvailableReplicas != deployment.Status.AvailableReplicas {
		reqLogger.Info("Updating Foo status.")
		foo.Status.AvailableReplicas = deployment.Status.AvailableReplicas

		if err := r.client.Update(ctx, foo); err != nil {
			reqLogger.Error(err, "Failed to update Foo status.")
			return reconcile.Result{}, err
		}

		reqLogger.Info("Updated Foo status", "foo.status.availableReplicas", foo.Status.AvailableReplicas)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileFoo) cleanupOwnedResources(ctx context.Context, foo *samplecontrollerv1alpha1.Foo) error {
	reqLogger := log.WithValues("Request.Namespace", foo.Namespace, "Request.Name", foo.Name)
	reqLogger.Info("finding existing Deployments for Foo resource")

	deployments := &appsv1.DeploymentList{}
	labelSelector := labels.SelectorFromSet(labelsForFoo(foo.Name))
	listOps := &client.ListOptions{
		Namespace:     foo.Namespace,
		LabelSelector: labelSelector,
	}
	if err := r.client.List(ctx, listOps, deployments); err != nil {
		reqLogger.Error(err, "Failed to get list of Deployments")
		return err
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == foo.Spec.DeploymentName {
			continue
		}

		if err := r.client.Delete(ctx, &deployment); err != nil {
			reqLogger.Error(err, "Failed to delete Deployment resource")
			return err
		}

		reqLogger.Info("Deleted old Deployment resource for Foo", "deploymentName", deployment.Name)
	}

	return nil
}

func labelsForFoo(name string) map[string]string {
	return map[string]string{"app": "nginx", "controller": name}
}

func newDeployment(foo *samplecontrollerv1alpha1.Foo) *appsv1.Deployment {
	labels := labelsForFoo(foo.Name)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Spec.DeploymentName,
			Namespace: foo.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(foo, samplecontrollerv1alpha1.SchemeGroupVersion.WithKind("Foo")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: foo.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
}

// SCAFFOLDED FUNCTION! REMOVE AFTER IMPLEMENTATION!!
// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *samplecontrollerv1alpha1.Foo) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
