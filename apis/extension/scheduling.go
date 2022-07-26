/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package extension

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/scheduling/v1alpha1"
)

const (
	// AnnotationCustomUsageThresholds represents the user-defined resource utilization threshold.
	// For specific value definitions, see CustomUsageThresholds
	AnnotationCustomUsageThresholds = SchedulingDomainPrefix + "/usage-thresholds"

	// AnnotationReservationAllocated represents the reservation allocated by the pod.
	AnnotationReservationAllocated = SchedulingDomainPrefix + "/reservation-allocated"
)

type Status string

//Gang scheduling consts
const (
	DefaultGangWaitTime = 2 * time.Minute

	StrictMode    = "StrictMode"
	NonStrictMode = "NonStrictMode"

	// Gang's Annotation
	GangNameAnnotation     = "gang.scheduling.koordinator.sh/name"
	GangMinNumAnnotation   = "gang.scheduling.koordinator.sh/min-available"
	GangWaitTimeAnnotation = "gang.scheduling.koordinator.sh/waiting-time"
	GangTotalNumAnnotation = "gang.scheduling.koordinator.sh/total-number"
	GangModeAnnotation     = "gang.scheduling.koordinator.sh/gang-mode"
	GangGroupsAnnotation   = "gang.scheduling.koordinator.sh/groups"
	GangTimeOutAnnotation  = "gang.scheduling.koordinator.sh/timeout"

	//Permit internal status
	GangNotFoundInCache Status = "Gang not found in cache"
	Success             Status = "Success"
	Wait                Status = "Wait"
)

// CustomUsageThresholds supports user-defined node resource utilization thresholds.
type CustomUsageThresholds struct {
	UsageThresholds map[corev1.ResourceName]int64 `json:"usageThresholds,omitempty"`
}

func GetCustomUsageThresholds(node *corev1.Node) (*CustomUsageThresholds, error) {
	usageThresholds := &CustomUsageThresholds{}
	data, ok := node.Annotations[AnnotationCustomUsageThresholds]
	if !ok {
		return usageThresholds, nil
	}
	err := json.Unmarshal([]byte(data), usageThresholds)
	if err != nil {
		return nil, err
	}
	return usageThresholds, nil
}

type ReservationAllocated struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

func GetReservationAllocated(pod *corev1.Pod) (*ReservationAllocated, error) {
	if pod.Annotations == nil {
		return nil, nil
	}
	data, ok := pod.Annotations[AnnotationReservationAllocated]
	if !ok {
		return nil, nil
	}
	reservationAllocated := &ReservationAllocated{}
	err := json.Unmarshal([]byte(data), reservationAllocated)
	if err != nil {
		return nil, err
	}
	return reservationAllocated, nil
}

func SetReservationAllocated(pod *corev1.Pod, r *schedulingv1alpha1.Reservation) {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	reservationAllocated := &ReservationAllocated{
		Namespace: r.Namespace,
		Name:      r.Name,
	}
	data, _ := json.Marshal(reservationAllocated) // assert no error
	pod.Annotations[AnnotationReservationAllocated] = string(data)
}
