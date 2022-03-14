package controllers

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	zkv1 "zookeeper-operator/api/v1"
)

func (r *ZookeeperClusterReconciler) handleConfigMap(instance *zkv1.ZookeeperCluster) (err error) {
	cm := MakeConfigMap(instance)
	if err = controllerutil.SetControllerReference(instance, cm, r.Scheme); err != nil {
		return err
	}
	foundCm := &v1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      cm.Name,
		Namespace: cm.Namespace,
	}, foundCm)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Zookeeper Config Map",
			"ConfigMap.Namespace", cm.Namespace,
			"ConfigMap.Name", cm.Name)
		err = r.Client.Create(context.TODO(), cm)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	} else {
		r.Log.Info("Updating existing config-map",
			"ConfigMap.Namespace", foundCm.Namespace,
			"ConfigMap.Name", foundCm.Name)
		SyncConfigMap(foundCm, cm)
		err = r.Client.Update(context.TODO(), foundCm)
		if err != nil {
			return err
		}
	}
	return nil
}

func MakeConfigMap(z *zkv1.ZookeeperCluster) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      z.ConfigMapName(),
			Namespace: z.Namespace,
			Labels:    z.Spec.Labels,
		},
		Data: map[string]string{
			"zoo.cfg":                makeZkConfigString(z),
			"log4j.properties":       makeZkLog4JConfigString(),
			"log4j-quiet.properties": makeZkLog4JQuietConfigString(),
			"env.sh":                 makeZkEnvConfigString(z),
		},
	}
}

func SyncConfigMap(curr *v1.ConfigMap, next *v1.ConfigMap) {
	curr.Data = next.Data
	curr.BinaryData = next.BinaryData
}

func makeZkConfigString(z *zkv1.ZookeeperCluster) string {
	ports := z.ZookeeperPorts()

	var zkConfig = ""
	for key, value := range z.Spec.Conf.AdditionalConfig {
		zkConfig = zkConfig + fmt.Sprintf("%s=%s\n", key, value)
	}
	return zkConfig + "4lw.commands.whitelist=cons, envi, conf, crst, srvr, stat, mntr, ruok\n" +
		"dataDir=/data\n" +
		"standaloneEnabled=false\n" +
		"reconfigEnabled=true\n" +
		"skipACL=yes\n" +
		"metricsProvider.className=org.apache.zookeeper.metrics.prometheus.PrometheusMetricsProvider\n" +
		"metricsProvider.httpPort=7000\n" +
		"metricsProvider.exportJvmInfo=true\n" +
		"initLimit=" + strconv.Itoa(z.Spec.Conf.InitLimit) + "\n" +
		"syncLimit=" + strconv.Itoa(z.Spec.Conf.SyncLimit) + "\n" +
		"tickTime=" + strconv.Itoa(z.Spec.Conf.TickTime) + "\n" +
		"globalOutstandingLimit=" + strconv.Itoa(z.Spec.Conf.GlobalOutstandingLimit) + "\n" +
		"preAllocSize=" + strconv.Itoa(z.Spec.Conf.PreAllocSize) + "\n" +
		"snapCount=" + strconv.Itoa(z.Spec.Conf.SnapCount) + "\n" +
		"commitLogCount=" + strconv.Itoa(z.Spec.Conf.CommitLogCount) + "\n" +
		"snapSizeLimitInKb=" + strconv.Itoa(z.Spec.Conf.SnapSizeLimitInKb) + "\n" +
		"maxCnxns=" + strconv.Itoa(z.Spec.Conf.MaxCnxns) + "\n" +
		"maxClientCnxns=" + strconv.Itoa(z.Spec.Conf.MaxClientCnxns) + "\n" +
		"minSessionTimeout=" + strconv.Itoa(z.Spec.Conf.MinSessionTimeout) + "\n" +
		"maxSessionTimeout=" + strconv.Itoa(z.Spec.Conf.MaxSessionTimeout) + "\n" +
		"autopurge.snapRetainCount=" + strconv.Itoa(z.Spec.Conf.AutoPurgeSnapRetainCount) + "\n" +
		"autopurge.purgeInterval=" + strconv.Itoa(z.Spec.Conf.AutoPurgePurgeInterval) + "\n" +
		"quorumListenOnAllIPs=" + strconv.FormatBool(z.Spec.Conf.QuorumListenOnAllIPs) + "\n" +
		"admin.serverPort=" + strconv.Itoa(int(ports.AdminServer)) + "\n" +
		"dynamicConfigFile=/data/zoo.cfg.dynamic\n"
}

func makeZkLog4JQuietConfigString() string {
	return "log4j.rootLogger=ERROR, CONSOLE\n" +
		"log4j.appender.CONSOLE=org.apache.log4j.ConsoleAppender\n" +
		"log4j.appender.CONSOLE.Threshold=ERROR\n" +
		"log4j.appender.CONSOLE.layout=org.apache.log4j.PatternLayout\n" +
		"log4j.appender.CONSOLE.layout.ConversionPattern=%d{ISO8601} [myid:%X{myid}] - %-5p [%t:%C{1}@%L] - %m%n\n"
}

func makeZkLog4JConfigString() string {
	return "zookeeper.root.logger=CONSOLE\n" +
		"zookeeper.console.threshold=INFO\n" +
		"log4j.rootLogger=${zookeeper.root.logger}\n" +
		"log4j.appender.CONSOLE=org.apache.log4j.ConsoleAppender\n" +
		"log4j.appender.CONSOLE.Threshold=${zookeeper.console.threshold}\n" +
		"log4j.appender.CONSOLE.layout=org.apache.log4j.PatternLayout\n" +
		"log4j.appender.CONSOLE.layout.ConversionPattern=%d{ISO8601} [myid:%X{myid}] - %-5p [%t:%C{1}@%L] - %m%n\n"
}

func makeZkEnvConfigString(z *zkv1.ZookeeperCluster) string {
	ports := z.ZookeeperPorts()
	return "#!/usr/bin/env bash\n\n" +
		"DOMAIN=" + headlessDomain(z) + "\n" +
		"QUORUM_PORT=" + strconv.Itoa(int(ports.Quorum)) + "\n" +
		"LEADER_PORT=" + strconv.Itoa(int(ports.Leader)) + "\n" +
		"CLIENT_HOST=" + z.GetClientServiceName() + "\n" +
		"CLIENT_PORT=" + strconv.Itoa(int(ports.Client)) + "\n" +
		"ADMIN_SERVER_HOST=" + z.GetAdminServerServiceName() + "\n" +
		"ADMIN_SERVER_PORT=" + strconv.Itoa(int(ports.AdminServer)) + "\n" +
		"CLUSTER_NAME=" + z.GetName() + "\n" +
		"CLUSTER_SIZE=" + fmt.Sprint(z.Spec.Replicas) + "\n"
}

func headlessDomain(z *zkv1.ZookeeperCluster) string {
	return fmt.Sprintf("%s.%s.svc.%s", headlessSvcName(z), z.GetNamespace(), z.GetKubernetesClusterDomain())
}

func headlessSvcName(z *zkv1.ZookeeperCluster) string {
	return fmt.Sprintf("%s-headless", z.GetName())
}
