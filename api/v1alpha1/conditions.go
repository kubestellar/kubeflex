package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	TypeReady  ConditionType = "Ready"
	TypeSynced ConditionType = "Synced"
)

type ConditionReason string

const (
	ReasonAvailable                 ConditionReason = "Available"
	ReasonUnavailable               ConditionReason = "Unavailable"
	ReasonCreating                  ConditionReason = "Creating"
	ReasonDeleting                  ConditionReason = "Deleting"
	ReasonWaitingForPostCreateHooks ConditionReason = "WaitingForPostCreateHooks"
)

const (
	ReasonReconcileSuccess ConditionReason = "ReconcileSuccess"
	ReasonReconcileError   ConditionReason = "ReconcileError"
	ReasonReconcilePaused  ConditionReason = "ReconcilePaused"
)

// ControlPlaneCondition describes the state of a control plane at a certain point.
type ControlPlaneCondition struct {
	Type               ConditionType          `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastUpdateTime     metav1.Time            `json:"lastUpdateTime"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             ConditionReason        `json:"reason"`
	Message            string                 `json:"message"`
}

// areConditionsEqual compares two ControlPlaneCondition structs and
// returns true if they are equal (excluding LastTransitionTime and LastUpdateTime),
// false otherwise.
func AreConditionsEqual(c1, c2 ControlPlaneCondition) bool {
	if c1.Type != c2.Type || c1.Status != c2.Status || c1.Reason != c2.Reason || c1.Message != c2.Message {
		return false
	}
	return true
}

// setCondition sets the supplied ControlPlaneCondition in
// the given slice of conditions, replacing any existing conditions of
// the same type. Returns the updated slice of conditions.
func setCondition(conditions []ControlPlaneCondition, newCondition ControlPlaneCondition) []ControlPlaneCondition {
	for i, condition := range conditions {
		if condition.Type == newCondition.Type {
			conditions[i] = newCondition
			return conditions
		}
	}
	conditions = append(conditions, newCondition)
	return conditions
}

// areConditionSlicesSame compares two slices of ControlPlaneCondition structs and returns true if they are the same (ignoring order and LastTransitionTime and LastUpdateTime), false otherwise.
func AreConditionSlicesSame(c1, c2 []ControlPlaneCondition) bool {
	if len(c1) != len(c2) {
		return false
	}

	// Create maps for the conditions (keyed by Type) in both slices, ignoring LastTransitionTime and LastUpdateTime
	c1Map := make(map[ConditionType]ControlPlaneCondition)
	c2Map := make(map[ConditionType]ControlPlaneCondition)

	for _, condition := range c1 {
		withoutTimes := ControlPlaneCondition{
			Type:    condition.Type,
			Status:  condition.Status,
			Reason:  condition.Reason,
			Message: condition.Message,
		}
		c1Map[condition.Type] = withoutTimes
	}

	for _, condition := range c2 {
		withoutTimes := ControlPlaneCondition{
			Type:    condition.Type,
			Status:  condition.Status,
			Reason:  condition.Reason,
			Message: condition.Message,
		}
		c2Map[condition.Type] = withoutTimes
	}

	// Compare the maps
	for key, value := range c1Map {
		value2, ok := c2Map[key]
		if !ok || !AreConditionsEqual(value, value2) {
			return false
		}
	}
	return true
}

func HasConditionAvailable(conditions []ControlPlaneCondition) bool {
	for _, condition := range conditions {
		if condition.Type == TypeReady &&
			condition.Status == corev1.ConditionTrue &&
			condition.Reason == ReasonAvailable {
			return true
		}
	}
	return false
}

func EnsureCondition(cp *ControlPlane, newCondition ControlPlaneCondition) {
	if cp.Status.Conditions == nil {
		cp.Status.Conditions = []ControlPlaneCondition{}
	}
	cp.Status.Conditions = setCondition(cp.Status.Conditions, newCondition)
}

// Creating returns a condition that indicates the cp is currently
// being created.
func ConditionCreating() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonCreating,
	}
}

// Deleting returns a condition that indicates the cp is currently
// being deleted.
func ConditionDeleting() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonDeleting,
	}
}

// Available returns a condition that indicates the resource is
// currently observed to be available for use.
func ConditionAvailable() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonAvailable,
	}
}

// Unavailable returns a condition that indicates the resource is not
// currently available for use.
func ConditionUnavailable() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonUnavailable,
	}
}

// WaitingForPostCreateHooks returns a condition that indicates the resource is
// waiting for PostCreateHook completion before becoming ready.
func ConditionWaitingForPostCreateHooks() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonWaitingForPostCreateHooks,
		Message:            "Waiting for PostCreateHook completion",
	}
}

// ReconcileSuccess returns a condition indicating that KubeFlex reconciled the resource
func ConditionReconcileSuccess() ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeSynced,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonReconcileSuccess,
	}
}

// ReconcileError returns a condition indicating that KubeFlex encountered an
// error while reconciling the resource.
func ConditionReconcileError(err error) ControlPlaneCondition {
	return ControlPlaneCondition{
		Type:               TypeSynced,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             ReasonReconcileError,
		Message:            err.Error(),
	}
}
