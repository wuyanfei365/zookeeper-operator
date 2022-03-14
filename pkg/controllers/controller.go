/*
Copyright 2022 wuyanfei365.

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
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
	"zookeeper-operator/pkg/clients"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zookeeperv1 "zookeeper-operator/api/v1"
)

// ZookeeperClusterReconciler reconciles a ZookeeperCluster object
type ZookeeperClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
	ZkClient clients.ZookeeperClient
}

//+kubebuilder:rbac:groups=zookeeper.github.com,resources=zookeeperclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zookeeper.github.com,resources=zookeeperclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zookeeper.github.com,resources=zookeeperclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ZookeeperCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ZookeeperClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	zk := new(zookeeperv1.ZookeeperCluster)
	err := r.Client.Get(ctx, req.NamespacedName, zk)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	changed := zk.WithDefaults()
	if zk.GetTriggerRollingRestart() {
		r.Log.Info("Restarting zookeeper cluster")
		annotationkey, annotationvalue := getRollingRestartAnnotation()
		if zk.Spec.Pod.Annotations == nil {
			zk.Spec.Pod.Annotations = make(map[string]string)
		}
		zk.Spec.Pod.Annotations[annotationkey] = annotationvalue
		zk.SetTriggerRollingRestart(false)
		changed = true
	}
	if changed {
		r.Log.Info("Setting default settings for zookeeper-cluster")
		if err = r.Client.Update(context.TODO(), zk); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	/*
		r.reconcileFinalizers,
		r.reconcileConfigMap,
		r.reconcileStatefulSet,
		r.reconcileClientService,
		r.reconcileHeadlessService,
		r.reconcileAdminServerService,
		r.reconcilePodDisruptionBudget,
		r.reconcileClusterStatus,
	*/

	if err = r.handleFinalizers(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleFinalizers", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleConfigMap(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleConfigMap", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleStatefulSet(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleStatefulSet", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleClientService(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleClientService", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleHeadlessService(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleHeadlessService", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleAdminServerService(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleAdminServerService", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handlePodDisruptionBudget(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handlePodDisruptionBudget", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.handleClusterStatus(zk); err != nil {
		r.Recorder.Eventf(zk, v1.EventTypeWarning, "handleClusterStatus", "ErrMsg: %s", err.Error())
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZookeeperClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zookeeperv1.ZookeeperCluster{}).
		Complete(r)
}

func getRollingRestartAnnotation() (string, string) {
	return "restartTime", time.Now().Format(time.RFC850)
}