package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	zkv1 "zookeeper-operator/api/v1"
)

var DisableFinalizer bool

const (
	ZkFinalizer = "cleanUpZookeeperPVC"
)

func (r *ZookeeperClusterReconciler) handleFinalizers(instance *zkv1.ZookeeperCluster) (err error) {
	if instance.Spec.Persistence != nil && instance.Spec.Persistence.VolumeReclaimPolicy != zkv1.VolumeReclaimPolicyDelete {
		return nil
	}
	if instance.DeletionTimestamp.IsZero() {
		if !ContainsString(instance.ObjectMeta.Finalizers, ZkFinalizer) && !DisableFinalizer {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, ZkFinalizer)
			if err = r.Client.Update(context.TODO(), instance); err != nil {
				return err
			}
		}
		return r.cleanupOrphanPVCs(instance)
	} else {
		if ContainsString(instance.ObjectMeta.Finalizers, ZkFinalizer) {
			if err = r.cleanUpAllPVCs(instance); err != nil {
				return err
			}
			instance.ObjectMeta.Finalizers = RemoveString(instance.ObjectMeta.Finalizers, ZkFinalizer)
			if err = r.Client.Update(context.TODO(), instance); err != nil {
				return err
			}
		}
	}
	return nil
}

func RemoveString(slice []string, str string) (result []string) {
	for _, item := range slice {
		if item == str {
			continue
		}
		result = append(result, item)
	}
	return result
}

func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func (r *ZookeeperClusterReconciler) cleanUpAllPVCs(instance *zkv1.ZookeeperCluster) (err error) {
	pvcList, err := r.getPVCList(instance)
	if err != nil {
		return err
	}
	for _, pvcItem := range pvcList.Items {
		r.deletePVC(pvcItem)
	}
	return nil
}
func (r *ZookeeperClusterReconciler) getPVCCount(instance *zkv1.ZookeeperCluster) (pvcCount int, err error) {
	pvcList, err := r.getPVCList(instance)
	if err != nil {
		return -1, err
	}
	pvcCount = len(pvcList.Items)
	return pvcCount, nil
}

func (r *ZookeeperClusterReconciler) cleanupOrphanPVCs(instance *zkv1.ZookeeperCluster) (err error) {
	// this check should make sure we do not delete the PVCs before the STS has scaled down
	if instance.Status.ReadyReplicas == instance.Spec.Replicas {
		pvcCount, err := r.getPVCCount(instance)
		if err != nil {
			return err
		}
		r.Log.Info("cleanupOrphanPVCs", "PVC Count", pvcCount, "ReadyReplicas Count", instance.Status.ReadyReplicas)
		if pvcCount > int(instance.Spec.Replicas) {
			pvcList, err := r.getPVCList(instance)
			if err != nil {
				return err
			}
			for _, pvcItem := range pvcList.Items {
				// delete only Orphan PVCs
				if IsPVCOrphan(pvcItem.Name, instance.Spec.Replicas) {
					r.deletePVC(pvcItem)
				}
			}
		}
	}
	return nil
}

func (r *ZookeeperClusterReconciler) getPVCList(instance *zkv1.ZookeeperCluster) (pvList v1.PersistentVolumeClaimList, err error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app": instance.GetName(), "uid": string(instance.UID)},
	})
	pvclistOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: selector,
	}
	pvcList := &v1.PersistentVolumeClaimList{}
	err = r.Client.List(context.TODO(), pvcList, pvclistOps)
	return *pvcList, err
}

func (r *ZookeeperClusterReconciler) deletePVC(pvcItem v1.PersistentVolumeClaim) {
	pvcDelete := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcItem.Name,
			Namespace: pvcItem.Namespace,
		},
	}
	r.Log.Info("Deleting PVC", "With Name", pvcItem.Name)
	err := r.Client.Delete(context.TODO(), pvcDelete)
	if err != nil {
		r.Log.Error(err, "Error deleteing PVC.", "Name", pvcDelete.Name)
	}
}

func IsPVCOrphan(zkPvcName string, replicas int32) bool {
	index := strings.LastIndexAny(zkPvcName, "-")
	if index == -1 {
		return false
	}

	ordinal, err := strconv.Atoi(zkPvcName[index+1:])
	if err != nil {
		return false
	}

	return int32(ordinal) >= replicas
}

