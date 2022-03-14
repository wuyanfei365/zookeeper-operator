package controllers

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"strings"
	"time"
	zkv1 "zookeeper-operator/api/v1"
)

func (r *ZookeeperClusterReconciler) handleStatefulSet(instance *zkv1.ZookeeperCluster) (err error) {

	// we cannot upgrade if cluster is in UpgradeFailed
	if instance.Status.IsClusterInUpgradeFailedState() {
		return nil
	}
	//if instance.Spec.Pod.ServiceAccountName != "default" {
	//	serviceAccount := MakeServiceAccount(instance)
	//	if err = controllerutil.SetControllerReference(instance, serviceAccount, r.Scheme); err != nil {
	//		return err
	//	}
	//	// Check if this ServiceAccount already exists
	//	foundServiceAccount := &v1.ServiceAccount{}
	//	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, foundServiceAccount)
	//	if err != nil && errors.IsNotFound(err) {
	//		r.Log.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
	//		err = r.Client.Create(context.TODO(), serviceAccount)
	//		if err != nil {
	//			return err
	//		}
	//	} else if err != nil {
	//		return err
	//	} else {
	//		foundServiceAccount.ImagePullSecrets = serviceAccount.ImagePullSecrets
	//		r.Log.Info("Updating ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
	//		err = r.Client.Update(context.TODO(), foundServiceAccount)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//}
	sts := MakeStatefulSet(instance)
	if err = controllerutil.SetControllerReference(instance, sts, r.Scheme); err != nil {
		return err
	}
	foundSts := &appsv1.StatefulSet{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      sts.Name,
		Namespace: sts.Namespace,
	}, foundSts)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Zookeeper StatefulSet",
			"StatefulSet.Namespace", sts.Namespace,
			"StatefulSet.Name", sts.Name)
		// label the RV of the zookeeperCluster when creating the sts
		sts.Labels["owner-rv"] = instance.ResourceVersion
		err = r.Client.Create(context.TODO(), sts)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	} else {
		// check whether zookeeperCluster is updated before updating the sts
		cmp := compareResourceVersion(instance, foundSts)
		if cmp < 0 {
			return fmt.Errorf("Staleness: cr.ResourceVersion %s is smaller than labeledRV %s", instance.ResourceVersion, foundSts.Labels["owner-rv"])
		} else if cmp > 0 {
			// Zookeeper StatefulSet version inherits ZookeeperCluster resource version
			foundSts.Labels["owner-rv"] = instance.ResourceVersion
		}
		foundSTSSize := *foundSts.Spec.Replicas
		newSTSSize := *sts.Spec.Replicas
		if newSTSSize != foundSTSSize {
			zkUri := GetZkServiceUri(instance)
			err = r.ZkClient.Connect(zkUri)
			if err != nil {
				return fmt.Errorf("Error storing cluster size %v", err)
			}
			defer r.ZkClient.Close()
			r.Log.Info("Connected to ZK", "ZKURI", zkUri)

			path := GetMetaPath(instance)
			version, _ := r.ZkClient.NodeExists(path)
			//if err != nil {
			//	return fmt.Errorf("Error doing exists check for znode %s: %v", path, err)
			//}

			data := "CLUSTER_SIZE=" + strconv.Itoa(int(newSTSSize))
			r.Log.Info("Updating Cluster Size.", "New Data:", data, "Version", version)
			_ = r.ZkClient.UpdateNode(path, data, version)
		}
		err = r.updateStatefulSet(instance, foundSts, sts)
		if err != nil {
			return err
		}
		return r.upgradeStatefulSet(instance, foundSts)
	}
}

var zkDataVolume = "data"

// MakeStatefulSet return a zookeeper stateful set from the zk spec
func MakeStatefulSet(z *zkv1.ZookeeperCluster) *appsv1.StatefulSet {
	extraVolumes := []v1.Volume{}
	persistence := z.Spec.Persistence
	pvcs := []v1.PersistentVolumeClaim{}
	if strings.EqualFold(z.Spec.StorageType, "ephemeral") {
		extraVolumes = append(extraVolumes, v1.Volume{
			Name: zkDataVolume,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &z.Spec.Ephemeral.EmptyDirVolumeSource,
			},
		})
	} else {
		pvcs = append(pvcs, v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: zkDataVolume,
				Labels: mergeLabels(
					z.Spec.Labels,
					map[string]string{"app": z.GetName(), "uid": string(z.UID)},
				),
				Annotations: z.Spec.Persistence.Annotations,
			},
			Spec: persistence.PersistentVolumeClaimSpec,
		})
	}
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      z.GetName(),
			Namespace: z.Namespace,
			Labels:    z.Spec.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: headlessSvcName(z),
			Replicas:    &z.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": z.GetName(),
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: z.GetName(),
					Labels: mergeLabels(
						z.Spec.Labels,
						map[string]string{
							"app":  z.GetName(),
							"kind": "ZookeeperMember",
						},
					),
					Annotations: z.Spec.Pod.Annotations,
				},
				Spec: makeZkPodSpec(z, extraVolumes),
			},
			VolumeClaimTemplates: pvcs,
		},
	}
}

func makeZkPodSpec(z *zkv1.ZookeeperCluster, volumes []v1.Volume) v1.PodSpec {
	zkContainer := v1.Container{
		Name:  "zookeeper",
		Image: z.Spec.Image.ToString(),
		Ports: z.Spec.Ports,
		Env: []v1.EnvVar{
			{
				Name: "ENVOY_SIDECAR_STATUS",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: `metadata.annotations['sidecar.istio.io/status']`,
					},
				},
			},
		},
		ImagePullPolicy: z.Spec.Image.PullPolicy,
		ReadinessProbe: &v1.Probe{
			InitialDelaySeconds: z.Spec.Probes.ReadinessProbe.InitialDelaySeconds,
			PeriodSeconds:       z.Spec.Probes.ReadinessProbe.PeriodSeconds,
			TimeoutSeconds:      z.Spec.Probes.ReadinessProbe.TimeoutSeconds,
			FailureThreshold:    z.Spec.Probes.ReadinessProbe.FailureThreshold,
			SuccessThreshold:    z.Spec.Probes.ReadinessProbe.SuccessThreshold,

			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"zookeeperReady.sh"}},
			},
		},
		LivenessProbe: &v1.Probe{
			InitialDelaySeconds: z.Spec.Probes.LivenessProbe.InitialDelaySeconds,
			PeriodSeconds:       z.Spec.Probes.LivenessProbe.PeriodSeconds,
			TimeoutSeconds:      z.Spec.Probes.LivenessProbe.TimeoutSeconds,
			FailureThreshold:    z.Spec.Probes.LivenessProbe.FailureThreshold,

			ProbeHandler: v1.ProbeHandler{
				Exec: &v1.ExecAction{Command: []string{"zookeeperLive.sh"}},
			},
		},
		VolumeMounts: append(z.Spec.VolumeMounts, []v1.VolumeMount{
			{Name: "data", MountPath: "/data"},
			{Name: "conf", MountPath: "/conf"},
		}...),
		Lifecycle: &v1.Lifecycle{
			PreStop: &v1.LifecycleHandler{
				Exec: &v1.ExecAction{
					Command: []string{"zookeeperTeardown.sh"},
				},
			},
		},
		Command: []string{"/usr/local/bin/zookeeperStart.sh"},
	}
	if z.Spec.Pod.Resources.Limits != nil || z.Spec.Pod.Resources.Requests != nil {
		zkContainer.Resources = z.Spec.Pod.Resources
	}
	volumes = append(volumes, v1.Volume{
		Name: "conf",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: z.ConfigMapName(),
				},
			},
		},
	})

	zkContainer.Env = append(zkContainer.Env, z.Spec.Pod.Env...)
	podSpec := v1.PodSpec{
		Containers: append(z.Spec.Containers, zkContainer),
		Affinity:   z.Spec.Pod.Affinity,
		Volumes:    append(z.Spec.Volumes, volumes...),
	}
	if !reflect.DeepEqual(v1.PodSecurityContext{}, z.Spec.Pod.SecurityContext) {
		podSpec.SecurityContext = z.Spec.Pod.SecurityContext
	}
	podSpec.NodeSelector = z.Spec.Pod.NodeSelector
	podSpec.Tolerations = z.Spec.Pod.Tolerations
	podSpec.TerminationGracePeriodSeconds = &z.Spec.Pod.TerminationGracePeriodSeconds
	podSpec.ServiceAccountName = z.Spec.Pod.ServiceAccountName
	if z.Spec.InitContainers != nil {
		podSpec.InitContainers = z.Spec.InitContainers
	}

	return podSpec
}

func (r *ZookeeperClusterReconciler) updateStatefulSet(instance *zkv1.ZookeeperCluster, foundSts *appsv1.StatefulSet, sts *appsv1.StatefulSet) (err error) {
	r.Log.Info("Updating StatefulSet",
		"StatefulSet.Namespace", foundSts.Namespace,
		"StatefulSet.Name", foundSts.Name)
	SyncStatefulSet(foundSts, sts)

	err = r.Client.Update(context.TODO(), foundSts)
	if err != nil {
		return err
	}
	instance.Status.Replicas = foundSts.Status.Replicas
	instance.Status.ReadyReplicas = foundSts.Status.ReadyReplicas
	return nil
}

func (r *ZookeeperClusterReconciler) upgradeStatefulSet(instance *zkv1.ZookeeperCluster, foundSts *appsv1.StatefulSet) (err error) {

	//Getting the upgradeCondition from the zk clustercondition
	_, upgradeCondition := instance.Status.GetClusterCondition(zkv1.ClusterConditionUpgrading)

	if upgradeCondition == nil {
		// Initially set upgrading condition to false
		instance.Status.SetUpgradingConditionFalse()
		return nil
	}

	//Setting the upgrade condition to true to trigger the upgrade
	//When the zk cluster is upgrading Statefulset CurrentRevision and UpdateRevision are not equal and zk cluster image tag is not equal to CurrentVersion
	if upgradeCondition.Status == v1.ConditionFalse {
		if instance.Status.IsClusterInReadyState() && foundSts.Status.CurrentRevision != foundSts.Status.UpdateRevision && instance.Spec.Image.Tag != instance.Status.CurrentVersion {
			instance.Status.TargetVersion = instance.Spec.Image.Tag
			instance.Status.SetPodsReadyConditionFalse()
			instance.Status.SetUpgradingConditionTrue("", "")
		}
	}

	//checking if the upgrade is in progress
	if upgradeCondition.Status == v1.ConditionTrue {
		//checking when the targetversion is empty
		if instance.Status.TargetVersion == "" {
			r.Log.Info("upgrading to an unknown version: cancelling upgrade process")
			return r.clearUpgradeStatus(instance)
		}
		//Checking for upgrade completion
		if foundSts.Status.CurrentRevision == foundSts.Status.UpdateRevision {
			instance.Status.CurrentVersion = instance.Status.TargetVersion
			r.Log.Info("upgrade completed")
			return r.clearUpgradeStatus(instance)
		}
		//updating the upgradecondition if upgrade is in progress
		if foundSts.Status.CurrentRevision != foundSts.Status.UpdateRevision {
			r.Log.Info("upgrade in progress")
			if fmt.Sprint(foundSts.Status.UpdatedReplicas) != upgradeCondition.Message {
				instance.Status.UpdateProgress(zkv1.UpdatingZookeeperReason, fmt.Sprint(foundSts.Status.UpdatedReplicas))
			} else {
				err = checkSyncTimeout(instance, zkv1.UpdatingZookeeperReason, foundSts.Status.UpdatedReplicas, 10*time.Minute)
				if err != nil {
					instance.Status.SetErrorConditionTrue("UpgradeFailed", err.Error())
					return r.Client.Status().Update(context.TODO(), instance)
				} else {
					return nil
				}
			}
		}
	}
	return r.Client.Status().Update(context.TODO(), instance)
}

func (r *ZookeeperClusterReconciler) clearUpgradeStatus(z *zkv1.ZookeeperCluster) (err error) {
	z.Status.SetUpgradingConditionFalse()
	z.Status.TargetVersion = ""
	// need to deep copy the status struct, otherwise it will be overwritten
	// when updating the CR below
	status := z.Status.DeepCopy()

	err = r.Client.Update(context.TODO(), z)
	if err != nil {
		return err
	}

	z.Status = *status
	return nil
}

func checkSyncTimeout(z *zkv1.ZookeeperCluster, reason string, updatedReplicas int32, t time.Duration) error {
	lastCondition := z.Status.GetLastCondition()
	if lastCondition == nil {
		return nil
	}
	if lastCondition.Reason == reason && lastCondition.Message == fmt.Sprint(updatedReplicas) {
		// if reason and message are the same as before, which means there is no progress since the last reconciling,
		// then check if it reaches the timeout.
		parsedTime, _ := time.Parse(time.RFC3339, lastCondition.LastUpdateTime)
		if time.Now().After(parsedTime.Add(t)) {
			// timeout
			return fmt.Errorf("progress deadline exceeded")
		}
	}
	return nil
}

func MakeServiceAccount(z *zkv1.ZookeeperCluster) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      z.Spec.Pod.ServiceAccountName,
			Namespace: z.Namespace,
		},
		ImagePullSecrets: z.Spec.Pod.ImagePullSecrets,
	}
}

// MergeLabels merges label maps
func mergeLabels(l ...map[string]string) map[string]string {
	res := make(map[string]string)

	for _, v := range l {
		for lKey, lValue := range v {
			res[lKey] = lValue
		}
	}
	return res
}

func SyncStatefulSet(curr *appsv1.StatefulSet, next *appsv1.StatefulSet) {
	curr.Spec.Replicas = next.Spec.Replicas
	curr.Spec.Template = next.Spec.Template
	curr.Spec.UpdateStrategy = next.Spec.UpdateStrategy
}

const (
	// Root ZNode for storing all zookeeper-operator related metadata.
	//ZKMetaRoot = "/zookeeper-operator"
	ZKMetaRoot = "/zookeeper"
)

func GetZkServiceUri(zoo *zkv1.ZookeeperCluster) (zkUri string) {
	zkClientPort, _ := ContainerPortByName(zoo.Spec.Ports, "client")
	zkUri = zoo.GetClientServiceName() + "." + zoo.GetNamespace() + ".svc." + zoo.GetKubernetesClusterDomain() + ":" + strconv.Itoa(int(zkClientPort))
	return zkUri
}

func GetMetaPath(zoo *zkv1.ZookeeperCluster) (path string) {
	return fmt.Sprintf("%s/%s", ZKMetaRoot, zoo.Name)
}

// ContainerPortByName returns a container port of name provided
func ContainerPortByName(ports []v1.ContainerPort, name string) (cPort int32, err error) {
	for _, port := range ports {
		if port.Name == name {
			return port.ContainerPort, nil
		}
	}
	return cPort, fmt.Errorf("port not found")
}

// compareResourceVersion compare resoure versions for the supplied ZookeeperCluster and StatefulSet
// resources
// Returns:
//  0 if versions are equal
// -1 if ZookeeperCluster version is less than StatefulSet version
//  1 if ZookeeperCluster version is greater than StatefulSet version
func compareResourceVersion(zk *zkv1.ZookeeperCluster, sts *appsv1.StatefulSet) int {

	zkResourceVersion, zkErr := strconv.Atoi(zk.ResourceVersion)
	stsVersion, stsVersionFound := sts.Labels["owner-rv"]

	if !stsVersionFound {
		if zkErr != nil {
			return 0
		}
		return 1
	}
	stsResourceVersion, err := strconv.Atoi(stsVersion)
	if err != nil {
		if zkErr != nil {
			return 0
		}
		return 1
	}
	if zkResourceVersion < stsResourceVersion {
		return -1
	} else if zkResourceVersion > stsResourceVersion {
		return 1
	}
	return 0
}
