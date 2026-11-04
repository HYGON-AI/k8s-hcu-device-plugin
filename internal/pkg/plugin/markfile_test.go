/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseMarkFileName(t *testing.T) {
	info, ok := parseMarkFileName("uid-1_init-0_2_5")
	if !ok {
		t.Fatal("expected parse ok")
	}
	if info.podUID != "uid-1" || info.containerName != "init-0" || info.devIdx != 2 || info.vdevIdx != 5 {
		t.Fatalf("unexpected parse: %+v", info)
	}

	info, ok = parseMarkFileName("uid-1_my_init_name_0_3")
	if !ok {
		t.Fatal("expected parse ok with underscores in container name")
	}
	if info.containerName != "my_init_name" || info.devIdx != 0 || info.vdevIdx != 3 {
		t.Fatalf("underscore container name: %+v", info)
	}

	if _, ok := parseMarkFileName("bad"); ok {
		t.Fatal("expected parse failure")
	}
	if _, ok := parseMarkFileName("uid_ctr_x_y"); ok {
		t.Fatal("expected parse failure on non-int indices")
	}
}

func TestMarkFileStillNeededInitTerminated(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: "pod-1"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "init0"}},
			Containers:     []corev1.Container{{Name: "app"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name:  "init0",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}},
			}},
		},
	}
	if markFileStillNeeded(pod, "init0") {
		t.Fatal("classic init should be reclaimed after Terminated")
	}
	if !markFileStillNeeded(pod, "app") {
		t.Fatal("app container should keep mark while pod Running")
	}
}

func TestMarkFileStillNeededRestartableInit(t *testing.T) {
	always := corev1.ContainerRestartPolicyAlways
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:          "sidecar",
				RestartPolicy: &always,
			}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name:  "sidecar",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}},
			}},
		},
	}
	if !markFileStillNeeded(pod, "sidecar") {
		t.Fatal("restartable init should keep mark while pod Running")
	}
}

func TestMarkFileStillNeededPodTerminal(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	if markFileStillNeeded(pod, "app") {
		t.Fatal("terminal pod should reclaim")
	}
	if markFileStillNeeded(nil, "app") {
		t.Fatal("missing pod should reclaim")
	}
}

func TestPodMissingRequiredMarkIgnoresFinishedInit(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: "uid-abc"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name: "init0",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"hygon.com/hcumem": resource.MustParse("1024"),
					},
				},
			}},
			Containers: []corev1.Container{{Name: "app"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name:  "init0",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}},
			}},
		},
	}
	// Init finished and mark already reclaimed: must NOT treat as missing.
	if podMissingRequiredMark(pod, nil) {
		t.Fatal("finished init-only vHCU must not force-delete the pod")
	}
}

func TestPodMissingRequiredMarkDetectsRunningApp(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: "uid-abc"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"hygon.com/hcumem": resource.MustParse("1024"),
					},
				},
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if !podMissingRequiredMark(pod, nil) {
		t.Fatal("running app without mark should be detected")
	}
	if podMissingRequiredMark(pod, []string{"uid-abc_app_0_1"}) {
		t.Fatal("present mark should satisfy requirement")
	}
}

func TestInitContainerTerminatedSince(t *testing.T) {
	oldPod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name:  "init0",
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
		},
	}
	newPod := &corev1.Pod{
		Spec: corev1.PodSpec{InitContainers: []corev1.Container{{Name: "init0"}}},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name:  "init0",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}},
			}},
		},
	}
	if !initContainerTerminatedSince(oldPod, newPod) {
		t.Fatal("expected init termination to be detected")
	}
	if initContainerTerminatedSince(newPod, newPod) {
		t.Fatal("already-terminated should not re-trigger")
	}
}
