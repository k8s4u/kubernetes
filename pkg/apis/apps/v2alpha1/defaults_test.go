/*
Copyright 2017 The Kubernetes Authors.

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

package v2alpha1_test

import (
	"reflect"
	"testing"

	appsv2alpha1 "k8s.io/api/apps/v2alpha1"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	_ "k8s.io/kubernetes/pkg/apis/apps/install"
	. "k8s.io/kubernetes/pkg/apis/apps/v1"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	"k8s.io/kubernetes/pkg/features"
	utilpointer "k8s.io/utils/pointer"
)

func TestSetDefaultDaemonSetSpec(t *testing.T) {
	defaultLabels := map[string]string{"foo": "bar"}
	maxUnavailable := intstr.FromInt(1)
	maxSurge := intstr.FromInt(0)
	period := int64(v1.DefaultTerminationGracePeriodSeconds)
	defaultTemplate := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
			SchedulerName:                 v1.DefaultSchedulerName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: defaultLabels,
		},
	}
	templateNoLabel := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
			SchedulerName:                 v1.DefaultSchedulerName,
		},
	}
	tests := []struct {
		original *appsv2alpha1.DaemonSet
		expected *appsv2alpha1.DaemonSet
	}{
		{ // Labels change/defaulting test.
			original: &appsv2alpha1.DaemonSet{
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &appsv2alpha1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.DaemonSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateDaemonSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDaemonSet{
							MaxUnavailable: &maxUnavailable,
							MaxSurge:       &maxSurge,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{ // Labels change/defaulting test.
			original: &appsv2alpha1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: appsv2alpha1.DaemonSetSpec{
					Template:             defaultTemplate,
					RevisionHistoryLimit: utilpointer.Int32Ptr(1),
				},
			},
			expected: &appsv2alpha1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.DaemonSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateDaemonSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDaemonSet{
							MaxUnavailable: &maxUnavailable,
							MaxSurge:       &maxSurge,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(1),
				},
			},
		},
		{ // OnDeleteDaemonSetStrategyType update strategy.
			original: &appsv2alpha1.DaemonSet{
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: templateNoLabel,
					UpdateStrategy: appsv2alpha1.DaemonSetUpdateStrategy{
						Type: appsv2alpha1.OnDeleteDaemonSetStrategyType,
					},
				},
			},
			expected: &appsv2alpha1.DaemonSet{
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: templateNoLabel,
					UpdateStrategy: appsv2alpha1.DaemonSetUpdateStrategy{
						Type: appsv2alpha1.OnDeleteDaemonSetStrategyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{ // Custom unique label key.
			original: &appsv2alpha1.DaemonSet{
				Spec: appsv2alpha1.DaemonSetSpec{},
			},
			expected: &appsv2alpha1.DaemonSet{
				Spec: appsv2alpha1.DaemonSetSpec{
					Template: templateNoLabel,
					UpdateStrategy: appsv2alpha1.DaemonSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateDaemonSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDaemonSet{
							MaxUnavailable: &maxUnavailable,
							MaxSurge:       &maxSurge,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
	}

	for i, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*appsv2alpha1.DaemonSet)
		if !ok {
			t.Errorf("(%d) unexpected object: %v", i, got)
			t.FailNow()
		}
		if !apiequality.Semantic.DeepEqual(got.Spec, expected.Spec) {
			t.Errorf("(%d) got different than expected\ngot:\n\t%+v\nexpected:\n\t%+v", i, got.Spec, expected.Spec)
		}
	}
}

func getMaxUnavailable(maxUnavailable int) *intstr.IntOrString {
	maxUnavailableIntOrStr := intstr.FromInt(maxUnavailable)
	return &maxUnavailableIntOrStr
}

func getPartition(partition int32) *int32 {
	return &partition
}

func TestSetDefaultStatefulSet(t *testing.T) {
	defaultLabels := map[string]string{"foo": "bar"}
	var defaultPartition int32 = 0
	var defaultReplicas int32 = 1
	var notTheDefaultPartition int32 = 42

	period := int64(v1.DefaultTerminationGracePeriodSeconds)
	defaultTemplate := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
			SchedulerName:                 v1.DefaultSchedulerName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: defaultLabels,
		},
	}

	tests := []struct {
		name                       string
		original                   *appsv2alpha1.StatefulSet
		expected                   *appsv2alpha1.StatefulSet
		enablePVCDeletionPolicy    bool
		enableMaxUnavailablePolicy bool
	}{
		{
			name: "labels and default update strategy",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					MinReadySeconds:     int32(0),
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{
			name: "Alternate update strategy",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.OnDeleteStatefulSetStrategyType,
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					MinReadySeconds:     int32(0),
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.OnDeleteStatefulSetStrategyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{
			name: "Parallel pod management policy",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.ParallelPodManagement,
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					MinReadySeconds:     int32(0),
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.ParallelPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{
			name: "UpdateStrategy.RollingUpdate.Partition is not lost when UpdateStrategy.Type is not set",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &notTheDefaultPartition,
						},
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					MinReadySeconds:     int32(0),
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &notTheDefaultPartition,
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
		},
		{
			name: "PVC delete policy enabled, no policy specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: appsv2alpha1.RetainPersistentVolumeClaimRetentionPolicyType,
						WhenScaled:  appsv2alpha1.RetainPersistentVolumeClaimRetentionPolicyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enablePVCDeletionPolicy: true,
		},
		{
			name: "PVC delete policy enabled, with scaledown policy specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenScaled: appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: appsv2alpha1.RetainPersistentVolumeClaimRetentionPolicyType,
						WhenScaled:  appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enablePVCDeletionPolicy: true,
		},
		{
			name: "PVC delete policy disabled, with set deletion policy specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
						WhenScaled:  appsv2alpha1.RetainPersistentVolumeClaimRetentionPolicyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enablePVCDeletionPolicy: true,
		},
		{
			name: "PVC delete policy disabled, with policy specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenScaled: appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: &defaultPartition,
						},
					},
					PersistentVolumeClaimRetentionPolicy: &appsv2alpha1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenScaled: appsv2alpha1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enablePVCDeletionPolicy: false,
		},
		{
			name: "MaxUnavailable disabled, with maxUnavailable not specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition: getPartition(0),
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enableMaxUnavailablePolicy: false,
		},
		{
			name: "MaxUnavailable disabled, with default maxUnavailable specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      &defaultPartition,
							MaxUnavailable: getMaxUnavailable(1),
						},
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      getPartition(0),
							MaxUnavailable: getMaxUnavailable(1),
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enableMaxUnavailablePolicy: false,
		},
		{
			name: "MaxUnavailable disabled, with non default maxUnavailable specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      &notTheDefaultPartition,
							MaxUnavailable: getMaxUnavailable(3),
						},
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      getPartition(42),
							MaxUnavailable: getMaxUnavailable(3),
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enableMaxUnavailablePolicy: false,
		},
		{
			name: "MaxUnavailable enabled, with no maxUnavailable specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      getPartition(0),
							MaxUnavailable: getMaxUnavailable(1),
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enableMaxUnavailablePolicy: true,
		},
		{
			name: "MaxUnavailable enabled, with non default maxUnavailable specified",
			original: &appsv2alpha1.StatefulSet{
				Spec: appsv2alpha1.StatefulSetSpec{
					Template: defaultTemplate,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      &notTheDefaultPartition,
							MaxUnavailable: getMaxUnavailable(3),
						},
					},
				},
			},
			expected: &appsv2alpha1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: appsv2alpha1.StatefulSetSpec{
					Replicas:            &defaultReplicas,
					Template:            defaultTemplate,
					PodManagementPolicy: appsv2alpha1.OrderedReadyPodManagement,
					UpdateStrategy: appsv2alpha1.StatefulSetUpdateStrategy{
						Type: appsv2alpha1.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateStatefulSetStrategy{
							Partition:      getPartition(42),
							MaxUnavailable: getMaxUnavailable(3),
						},
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(10),
				},
			},
			enableMaxUnavailablePolicy: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.StatefulSetAutoDeletePVC, test.enablePVCDeletionPolicy)()
			defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.MaxUnavailableStatefulSet, test.enableMaxUnavailablePolicy)()

			obj2 := roundTrip(t, runtime.Object(test.original))
			got, ok := obj2.(*appsv2alpha1.StatefulSet)
			if !ok {
				t.Errorf("unexpected object: %v", got)
				t.FailNow()
			}
			if !apiequality.Semantic.DeepEqual(got.Spec, test.expected.Spec) {
				t.Errorf("got different than expected\ngot:\n\t%+v\nexpected:\n\t%+v", got.Spec, test.expected.Spec)
			}
		})
	}
}

func TestSetDefaultDeployment(t *testing.T) {
	defaultIntOrString := intstr.FromString("25%")
	differentIntOrString := intstr.FromInt(5)
	period := int64(v1.DefaultTerminationGracePeriodSeconds)
	defaultTemplate := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
			SchedulerName:                 v1.DefaultSchedulerName,
		},
	}
	tests := []struct {
		original *appsv2alpha1.Deployment
		expected *appsv2alpha1.Deployment
	}{
		{
			original: &appsv2alpha1.Deployment{},
			expected: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(1),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDeployment{
							MaxSurge:       &defaultIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					RevisionHistoryLimit:    utilpointer.Int32Ptr(10),
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(600),
					Template:                defaultTemplate,
				},
			},
		},
		{
			original: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						RollingUpdate: &appsv2alpha1.RollingUpdateDeployment{
							MaxSurge: &differentIntOrString,
						},
					},
				},
			},
			expected: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDeployment{
							MaxSurge:       &differentIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					RevisionHistoryLimit:    utilpointer.Int32Ptr(10),
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(600),
					Template:                defaultTemplate,
				},
			},
		},
		{
			original: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(3),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type:          appsv2alpha1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: nil,
					},
				},
			},
			expected: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(3),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &appsv2alpha1.RollingUpdateDeployment{
							MaxSurge:       &defaultIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					RevisionHistoryLimit:    utilpointer.Int32Ptr(10),
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(600),
					Template:                defaultTemplate,
				},
			},
		},
		{
			original: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RecreateDeploymentStrategyType,
					},
					RevisionHistoryLimit: utilpointer.Int32Ptr(0),
				},
			},
			expected: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RecreateDeploymentStrategyType,
					},
					RevisionHistoryLimit:    utilpointer.Int32Ptr(0),
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(600),
					Template:                defaultTemplate,
				},
			},
		},
		{
			original: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RecreateDeploymentStrategyType,
					},
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(30),
					RevisionHistoryLimit:    utilpointer.Int32Ptr(2),
				},
			},
			expected: &appsv2alpha1.Deployment{
				Spec: appsv2alpha1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(5),
					Strategy: appsv2alpha1.DeploymentStrategy{
						Type: appsv2alpha1.RecreateDeploymentStrategyType,
					},
					ProgressDeadlineSeconds: utilpointer.Int32Ptr(30),
					RevisionHistoryLimit:    utilpointer.Int32Ptr(2),
					Template:                defaultTemplate,
				},
			},
		},
	}

	for _, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*appsv2alpha1.Deployment)
		if !ok {
			t.Errorf("unexpected object: %v", got)
			t.FailNow()
		}
		if !apiequality.Semantic.DeepEqual(got.Spec, expected.Spec) {
			t.Errorf("object mismatch!\nexpected:\n\t%+v\ngot:\n\t%+v", got.Spec, expected.Spec)
		}
	}
}

func TestDefaultDeploymentAvailability(t *testing.T) {
	d := roundTrip(t, runtime.Object(&appsv2alpha1.Deployment{})).(*appsv2alpha1.Deployment)

	maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*(d.Spec.Replicas)), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *(d.Spec.Replicas)-int32(maxUnavailable) <= 0 {
		t.Fatalf("the default value of maxUnavailable can lead to no active replicas during rolling update")
	}
}

func TestSetDefaultReplicaSetReplicas(t *testing.T) {
	tests := []struct {
		rs             appsv2alpha1.ReplicaSet
		expectReplicas int32
	}{
		{
			rs: appsv2alpha1.ReplicaSet{
				Spec: appsv2alpha1.ReplicaSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 1,
		},
		{
			rs: appsv2alpha1.ReplicaSet{
				Spec: appsv2alpha1.ReplicaSetSpec{
					Replicas: utilpointer.Int32Ptr(0),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 0,
		},
		{
			rs: appsv2alpha1.ReplicaSet{
				Spec: appsv2alpha1.ReplicaSetSpec{
					Replicas: utilpointer.Int32Ptr(3),
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 3,
		},
	}

	for _, test := range tests {
		rs := &test.rs
		obj2 := roundTrip(t, runtime.Object(rs))
		rs2, ok := obj2.(*appsv2alpha1.ReplicaSet)
		if !ok {
			t.Errorf("unexpected object: %v", rs2)
			t.FailNow()
		}
		if rs2.Spec.Replicas == nil {
			t.Errorf("unexpected nil Replicas")
		} else if test.expectReplicas != *rs2.Spec.Replicas {
			t.Errorf("expected: %d replicas, got: %d", test.expectReplicas, *rs2.Spec.Replicas)
		}
	}
}

func TestDefaultRequestIsNotSetForReplicaSet(t *testing.T) {
	s := v1.PodSpec{}
	s.Containers = []v1.Container{
		{
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		},
	}
	rs := &appsv2alpha1.ReplicaSet{
		Spec: appsv2alpha1.ReplicaSetSpec{
			Replicas: utilpointer.Int32Ptr(3),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: s,
			},
		},
	}
	output := roundTrip(t, runtime.Object(rs))
	rs2 := output.(*appsv2alpha1.ReplicaSet)
	defaultRequest := rs2.Spec.Template.Spec.Containers[0].Resources.Requests
	requestValue := defaultRequest[v1.ResourceCPU]
	if requestValue.String() != "0" {
		t.Errorf("Expected 0 request value, got: %s", requestValue.String())
	}
}

func roundTrip(t *testing.T, obj runtime.Object) runtime.Object {
	data, err := runtime.Encode(legacyscheme.Codecs.LegacyCodec(SchemeGroupVersion), obj)
	if err != nil {
		t.Errorf("%v\n %#v", err, obj)
		return nil
	}
	obj2, err := runtime.Decode(legacyscheme.Codecs.UniversalDecoder(), data)
	if err != nil {
		t.Errorf("%v\nData: %s\nSource: %#v", err, string(data), obj)
		return nil
	}
	obj3 := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(runtime.Object)
	err = legacyscheme.Scheme.Convert(obj2, obj3, nil)
	if err != nil {
		t.Errorf("%v\nSource: %#v", err, obj2)
		return nil
	}
	return obj3
}
