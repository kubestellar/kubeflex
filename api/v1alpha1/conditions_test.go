package v1alpha1

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAreConditionsEqual(t *testing.T) {
	// Create two conditions with only the LastUpdateTime field different
	c1 := ControlPlaneCondition{
		Type:               "ConditionType",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             "Reason",
		Message:            "Message",
	}
	c2 := ControlPlaneCondition{
		Type:               "ConditionType",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.NewTime(time.Now().Add(1 * time.Hour)),
		Reason:             "Reason",
		Message:            "Message",
	}

	if !AreConditionsEqual(c1, c2) {
		t.Errorf("AreConditionsEqual failed: expected true, but got false")
	}

	// Create two conditions with all fields different
	c3 := ControlPlaneCondition{
		Type:               "ConditionTypeA",
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastUpdateTime:     metav1.Now(),
		Reason:             "ReasonA",
		Message:            "MessageA",
	}
	c4 := ControlPlaneCondition{
		Type:               "ConditionTypeB",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
		LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
		Reason:             "ReasonB",
		Message:            "MessageB",
	}

	if AreConditionsEqual(c3, c4) {
		t.Errorf("AreConditionsEqual failed: expected false, but got true")
	}
}

func TestSetCondition(t *testing.T) {
	// Create a slice of conditions and set a new condition
	conditions1 := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeA",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Reason:             "ReasonA",
			Message:            "MessageA",
		},
		{
			Type:               "ConditionTypeB",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
			Reason:             "ReasonB",
			Message:            "MessageB",
		},
	}

	newCondition := ControlPlaneCondition{
		Type:               "ConditionTypeA",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now().Add(4 * time.Hour)),
		LastUpdateTime:     metav1.NewTime(time.Now().Add(5 * time.Hour)),
		Reason:             "ReasonAUpdated",
		Message:            "MessageAUpdated",
	}

	expectedConditions := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeA",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(4 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(5 * time.Hour)),
			Reason:             "ReasonAUpdated",
			Message:            "MessageAUpdated",
		},
		{
			Type:               "ConditionTypeB",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
			Reason:             "ReasonB",
			Message:            "MessageB",
		},
	}

	actualConditions := SetCondition(conditions1, newCondition)

	if !AreConditionSlicesSame(actualConditions, expectedConditions) {
		t.Errorf("SetCondition failed: expected %+v, but got %+v", expectedConditions, actualConditions)
	}
}

func TestAreConditionSlicesSame(t *testing.T) {
	// Create two slices of conditions with the same elements in different orders
	c1 := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeA",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Reason:             "ReasonA",
			Message:            "MessageA",
		},
		{
			Type:               "ConditionTypeB",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
			Reason:             "ReasonB",
			Message:            "MessageB",
		},
	}
	c2 := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeB",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
			Reason:             "ReasonB",
			Message:            "MessageB",
		},
		{
			Type:               "ConditionTypeA",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Reason:             "ReasonA",
			Message:            "MessageA",
		},
	}

	if !AreConditionSlicesSame(c1, c2) {
		t.Errorf("AreConditionSlicesSame failed: expected true, but got false")
	}

	// Create two slices of conditions with different elements
	c3 := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeA",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Reason:             "ReasonA",
			Message:            "MessageA",
		},
		{
			Type:               "ConditionTypeB",
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(2 * time.Hour)),
			LastUpdateTime:     metav1.NewTime(time.Now().Add(3 * time.Hour)),
			Reason:             "ReasonB",
			Message:            "MessageB",
		},
	}
	c4 := []ControlPlaneCondition{
		{
			Type:               "ConditionTypeC",
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Reason:             "ReasonC",
			Message:            "MessageC",
		},
	}

	if AreConditionSlicesSame(c3, c4) {
		t.Errorf("AreConditionSlicesSame failed: expected false, but got true")
	}
}
