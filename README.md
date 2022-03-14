# zookeeper-operator demo

## Zookeeper cluster template source
 > https://github.com/helm/charts/tree/master/incubator/zookeeper

## quick start

### (1) download operator and install from
```shell
# url: https://github.com/wuyanfei365/zookeeper-operator/tree/master/charts/operator
helm install zk-operator ./operator
```
### (2) create zookeeper cluster
```shell
kubectl apply -f https://github.com/wuyanfei365/zookeeper-operator/tree/master/charts/zk-cluster/sample.yaml
```

### test record
```shell
kubectl delete ZookeeperCluster zk -nzk
helm uninstall zk-operator  -nzk
helm install zk-operator ./operator -n zk
kubectl get pods -n zk
kubectl apply -f  .\zk-cluster\sample.yaml
kubectl get pods -n zk
kubectl edit ZookeeperCluster zk -nzk
kubectl get pods -n zk
```