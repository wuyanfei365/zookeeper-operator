package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	zkv1 "zookeeper-operator/api/v1"
)

func (r *ZookeeperClusterReconciler) handleAdminServerService(instance *zkv1.ZookeeperCluster) (err error) {
	svc := MakeAdminServerService(instance)
	if err = controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &v1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating admin server service",
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
		r.Log.Info("Updating existing admin server service",
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

func MakeAdminServerService(z *zkv1.ZookeeperCluster) *v1.Service {
	ports := z.ZookeeperPorts()
	svcPorts := []v1.ServicePort{
		{Name: "tcp-admin-server", Port: ports.AdminServer},
	}
	external := z.Spec.AdminServerService.External
	annotations := z.Spec.AdminServerService.Annotations
	return makeService(z.GetAdminServerServiceName(), svcPorts, true, external, annotations, z)
}
