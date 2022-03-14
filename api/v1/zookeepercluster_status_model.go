package v1

import (
	v1 "k8s.io/api/core/v1"
	"time"
)

type ClusterConditionType string

const (
	ClusterConditionPodsReady ClusterConditionType = "PodsReady"
	ClusterConditionUpgrading                      = "Upgrading"
	ClusterConditionError                          = "Error"
	UpdatingZookeeperReason = "Updating Zookeeper"
)

type MembersStatus struct {
	//+nullable
	Ready []string `json:"ready,omitempty"`
	//+nullable
	Unready []string `json:"unready,omitempty"`
}

type ClusterCondition struct {

	Type ClusterConditionType `json:"type,omitempty"`

	Status v1.ConditionStatus `json:"status,omitempty"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

func (zs *ZookeeperClusterStatus) Init() {
	conditionTypes := []ClusterConditionType{
		ClusterConditionPodsReady,
		ClusterConditionUpgrading,
		ClusterConditionError,
	}
	for _, conditionType := range conditionTypes {
		if _, condition := zs.GetClusterCondition(conditionType); condition == nil {
			c := newClusterCondition(conditionType, v1.ConditionFalse, "", "")
			zs.setClusterCondition(*c)
		}
	}
}

func newClusterCondition(condType ClusterConditionType, status v1.ConditionStatus, reason, message string) *ClusterCondition {
	return &ClusterCondition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastUpdateTime:     "",
		LastTransitionTime: "",
	}
}

func (zs *ZookeeperClusterStatus) SetPodsReadyConditionTrue() {
	c := newClusterCondition(ClusterConditionPodsReady, v1.ConditionTrue, "", "")
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) SetPodsReadyConditionFalse() {
	c := newClusterCondition(ClusterConditionPodsReady, v1.ConditionFalse, "", "")
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) SetUpgradingConditionTrue(reason, message string) {
	c := newClusterCondition(ClusterConditionUpgrading, v1.ConditionTrue, reason, message)
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) SetUpgradingConditionFalse() {
	c := newClusterCondition(ClusterConditionUpgrading, v1.ConditionFalse, "", "")
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) SetErrorConditionTrue(reason, message string) {
	c := newClusterCondition(ClusterConditionError, v1.ConditionTrue, reason, message)
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) SetErrorConditionFalse() {
	c := newClusterCondition(ClusterConditionError, v1.ConditionFalse, "", "")
	zs.setClusterCondition(*c)
}

func (zs *ZookeeperClusterStatus) GetClusterCondition(t ClusterConditionType) (int, *ClusterCondition) {
	for i, c := range zs.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func (zs *ZookeeperClusterStatus) setClusterCondition(newCondition ClusterCondition) {
	now := time.Now().Format(time.RFC3339)
	position, existingCondition := zs.GetClusterCondition(newCondition.Type)

	if existingCondition == nil {
		zs.Conditions = append(zs.Conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = now
		existingCondition.LastUpdateTime = now
	}

	if existingCondition.Reason != newCondition.Reason || existingCondition.Message != newCondition.Message {
		existingCondition.Reason = newCondition.Reason
		existingCondition.Message = newCondition.Message
		existingCondition.LastUpdateTime = now
	}

	zs.Conditions[position] = *existingCondition
}

func (zs *ZookeeperClusterStatus) IsClusterInUpgradeFailedState() bool {
	_, errorCondition := zs.GetClusterCondition(ClusterConditionError)
	if errorCondition == nil {
		return false
	}
	if errorCondition.Status == v1.ConditionTrue && errorCondition.Reason == "UpgradeFailed" {
		return true
	}
	return false
}

func (zs *ZookeeperClusterStatus) IsClusterInUpgradingState() bool {
	_, upgradeCondition := zs.GetClusterCondition(ClusterConditionUpgrading)
	if upgradeCondition == nil {
		return false
	}
	if upgradeCondition.Status == v1.ConditionTrue {
		return true
	}
	return false
}

func (zs *ZookeeperClusterStatus) IsClusterInReadyState() bool {
	_, readyCondition := zs.GetClusterCondition(ClusterConditionPodsReady)
	if readyCondition != nil && readyCondition.Status == v1.ConditionTrue {
		return true
	}
	return false
}

func (zs *ZookeeperClusterStatus) UpdateProgress(reason, updatedReplicas string) {
	if zs.IsClusterInUpgradingState() {
		zs.SetUpgradingConditionTrue(reason, updatedReplicas)
	}
}

func (zs *ZookeeperClusterStatus) GetLastCondition() (lastCondition *ClusterCondition) {
	if zs.IsClusterInUpgradingState() {
		_, lastCondition := zs.GetClusterCondition(ClusterConditionUpgrading)
		return lastCondition
	}
	return nil
}
