package yulong

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Reasons for YuLong conditions.

	// Ready .
	Ready = "Ready"
	// ClusterUnReady .
	ClusterUnReady = "ClusterUnReady"
)

// NewYuLongCondition creates a new yu long condition.
func NewYuLongCondition(condType v1alpha1.YuLongConditionType, status v1.ConditionStatus, reason, message string) *v1alpha1.YuLongCondition {
	return &v1alpha1.YuLongCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetYuLongCondition returns the condition with the provided type.
func GetYuLongCondition(status v1alpha1.YuLongStatus, condType v1alpha1.YuLongConditionType) *v1alpha1.YuLongCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetYuLongCondition updates the tidb cluster to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetYuLongCondition(status *v1alpha1.YuLongStatus, condition v1alpha1.YuLongCondition) {
	currentCond := GetYuLongCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of yu long conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.YuLongCondition, condType v1alpha1.YuLongConditionType) []v1alpha1.YuLongCondition {
	var newConditions []v1alpha1.YuLongCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetYuLongReadyCondition extracts the yu long ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetYuLongReadyCondition(status v1alpha1.YuLongStatus) *v1alpha1.YuLongCondition {
	return GetYuLongCondition(status, v1alpha1.YuLongReady)
}
