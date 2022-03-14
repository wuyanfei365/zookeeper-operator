package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	zkv1 "zookeeper-operator/api/v1"
)

func (r *ZookeeperClusterReconciler) handleHeadlessService(instance *zkv1.ZookeeperCluster) (err error) {
	svc := MakeHeadlessService(instance)
	if err = controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &v1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating new headless service",
			"Service.Namespace", svc.Namespace,
			"Service.Name", svc.Name)
		err = r.Client.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	} else {
		r.Log.Info("Updating existing headless service",
			"Service.Namespace", foundSvc.Namespace,
			"Service.Name", foundSvc.Name)
		SyncService(foundSvc, svc)
		err = r.Client.Update(context.TODO(), foundSvc)
		if err != nil {
			return err
		}
	}
	return nil
}

func MakeHeadlessService(z *zkv1.ZookeeperCluster) *v1.Service {
	ports := z.ZookeeperPorts()
	svcPorts := []v1.ServicePort{
		{Name: "tcp-client", Port: ports.Client},
		{Name: "tcp-quorum", Port: ports.Quorum},
		{Name: "tcp-leader-election", Port: ports.Leader},
		{Name: "tcp-metrics", Port: ports.Metrics},
		{Name: "tcp-admin-server", Port: ports.AdminServer},
	}
	return makeService(headlessSvcName(z), svcPorts, false, false, z.Spec.HeadlessService.Annotations, z)
}
