/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func hcuContainer(name string, request bool) corev1.Container {
	c := corev1.Container{Name: name}
	if request {
		c.Resources.Limits = corev1.ResourceList{
			"hygon.com/hcunum": resource.MustParse("1"),
		}
	}
	return c
}

func TestDecodePodDevicesPreservesEmptySegments(t *testing.T) {
	annos := map[string]string{
		InRequestDevices[HygonHCUDevice]: ";HCU-A,HCU,100,100:;;HCU-B,HCU,100,100:;",
	}
	pd, err := DecodePodDevices(InRequestDevices, annos)
	if err != nil {
		t.Fatalf("DecodePodDevices: %v", err)
	}
	got := pd[HygonHCUDevice]
	if len(got) != 4 {
		t.Fatalf("len=%d, want 4 (empty slots preserved)", len(got))
	}
	if len(got[0]) != 0 || len(got[2]) != 0 {
		t.Fatalf("empty segments should stay empty: %#v", got)
	}
	if len(got[1]) != 1 || got[1][0].UUID != "HCU-A" {
		t.Fatalf("segment 1: %#v", got[1])
	}
	if len(got[3]) != 1 || got[3][0].UUID != "HCU-B" {
		t.Fatalf("segment 3: %#v", got[3])
	}
}

func TestGetNextDeviceRequestFlatIndex(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				// init0 empty, init1 has device, app0 empty, app1 has device
				InRequestDevices[HygonHCUDevice]: ";HCU-A,HCU,100,100:;;HCU-B,HCU,100,100:;",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				hcuContainer("init0", false),
				hcuContainer("init1", true),
			},
			Containers: []corev1.Container{
				hcuContainer("app0", false),
				hcuContainer("app1", true),
			},
		},
	}

	ctr, devs, err := GetNextDeviceRequest(HygonHCUDevice, pod)
	if err != nil {
		t.Fatalf("GetNextDeviceRequest: %v", err)
	}
	if ctr.Name != "init1" {
		t.Fatalf("got container %q, want init1 (flat index 1)", ctr.Name)
	}
	if len(devs) != 1 || devs[0].UUID != "HCU-A" {
		t.Fatalf("devs=%#v", devs)
	}
}

// HAMi dense encoding: only segments for containers that requested devices.
// Must map onto init-vhcu-check then hcu-matmul, not fail on 2 vs 4 segments.
func TestGetNextDeviceRequestDenseHAMiEncoding(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				InRequestDevices[HygonHCUDevice]: "HCU-A,HCU,1024,10:;HCU-B,HCU,10240,30:;",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				hcuContainer("init-hello", false),
				hcuContainer("init-vhcu-check", true),
			},
			Containers: []corev1.Container{
				hcuContainer("hcu-matmul", true),
				hcuContainer("hello-cn", false),
			},
		},
	}

	ctr, devs, err := GetNextDeviceRequest(HygonHCUDevice, pod)
	if err != nil {
		t.Fatalf("GetNextDeviceRequest: %v", err)
	}
	if ctr.Name != "init-vhcu-check" {
		t.Fatalf("got container %q, want init-vhcu-check", ctr.Name)
	}
	if len(devs) != 1 || devs[0].UUID != "HCU-A" {
		t.Fatalf("devs=%#v", devs)
	}

	pd, err := DecodePodDevices(InRequestDevices, pod.Annotations)
	if err != nil {
		t.Fatalf("DecodePodDevices: %v", err)
	}
	expanded, err := expandPodDeviceSegments(pd[HygonHCUDevice], &pod)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	pod.Annotations[InRequestDevices[HygonHCUDevice]] = EncodePodSingleDevice(eraseFirstDeviceSegment(expanded))

	ctr, devs, err = GetNextDeviceRequest(HygonHCUDevice, pod)
	if err != nil {
		t.Fatalf("second GetNextDeviceRequest: %v", err)
	}
	if ctr.Name != "hcu-matmul" {
		t.Fatalf("got container %q, want hcu-matmul", ctr.Name)
	}
	if len(devs) != 1 || devs[0].UUID != "HCU-B" {
		t.Fatalf("devs=%#v", devs)
	}
}

// Ambiguous dense annotation (device segments != requesting containers) must fail.
func TestGetNextDeviceRequestRejectsUnalignableDenseAnnotation(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				InRequestDevices[HygonHCUDevice]: "HCU-A,HCU,100,100:;HCU-B,HCU,100,100:;",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{hcuContainer("app0", true)},
		},
	}

	if _, _, err := GetNextDeviceRequest(HygonHCUDevice, pod); err == nil {
		t.Fatalf("expected error when 2 device segments vs 1 requesting container")
	}
}

func TestSplitPodDeviceAnnotation(t *testing.T) {
	tests := []struct {
		name string
		anno string
		want []string
	}{
		{name: "empty", anno: "", want: nil},
		{name: "single with trailing", anno: "HCU-A,HCU,100,100:;", want: []string{"HCU-A,HCU,100,100:"}},
		{name: "four containers last gpu", anno: ";;;HCU-A,HCU,100,100:;", want: []string{"", "", "", "HCU-A,HCU,100,100:"}},
		{name: "two gpu containers", anno: "HCU-A,HCU,100,100:;HCU-B,HCU,100,100:;", want: []string{"HCU-A,HCU,100,100:", "HCU-B,HCU,100,100:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPodDeviceAnnotation(tt.anno)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitPodDeviceAnnotation(%q) = %#v, want %#v", tt.anno, got, tt.want)
			}
		})
	}
}

func TestGetPodContainerByIndex(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				hcuContainer("init0", true),
				hcuContainer("init1", false),
			},
			Containers: []corev1.Container{
				hcuContainer("app0", false),
				hcuContainer("app1", true),
			},
		},
	}

	tests := []struct {
		idx     int
		want    string
		wantErr bool
	}{
		{idx: 0, want: "init0"},
		{idx: 1, want: "init1"},
		{idx: 2, want: "app0"},
		{idx: 3, want: "app1"},
		{idx: -1, wantErr: true},
		{idx: 4, wantErr: true},
	}
	for _, tt := range tests {
		ctr, err := GetPodContainerByIndex(pod, tt.idx)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("idx %d: expected error", tt.idx)
			}
			continue
		}
		if err != nil {
			t.Fatalf("idx %d: unexpected error: %v", tt.idx, err)
		}
		if ctr.Name != tt.want {
			t.Fatalf("idx %d: got %q, want %q", tt.idx, ctr.Name, tt.want)
		}
	}
}

func TestEraseFirstDeviceSegment(t *testing.T) {
	segA, err := DecodeContainerDevices("HCU-A,HCU,100,100:")
	if err != nil {
		t.Fatalf("decode A: %v", err)
	}
	segB, err := DecodeContainerDevices("HCU-B,HCU,100,100:")
	if err != nil {
		t.Fatalf("decode B: %v", err)
	}

	// flat: init0 empty, init1 holds HCU-A, app0 empty, app1 holds HCU-B
	pd := PodSingleDevice{ContainerDevices{}, segA, ContainerDevices{}, segB}

	got := eraseFirstDeviceSegment(pd)
	if len(got) != 4 {
		t.Fatalf("len=%d, want 4 (segment positions must survive the erase)", len(got))
	}
	if len(got[1]) != 0 {
		t.Fatalf("segment 1 not erased: %#v", got[1])
	}
	if len(got[3]) != 1 || got[3][0].UUID != "HCU-B" {
		t.Fatalf("segment 3: %#v", got[3])
	}

	got = eraseFirstDeviceSegment(got)
	if len(got) != 4 {
		t.Fatalf("len=%d, want 4", len(got))
	}
	if len(got[3]) != 0 {
		t.Fatalf("segment 3 not erased: %#v", got[3])
	}
	// The annotation must keep one segment per container until the pod is done.
	if encoded := EncodePodSingleDevice(got); encoded != ";;;;" {
		t.Fatalf("encoded=%q, want %q", encoded, ";;;;")
	}
}

func TestRequestsHCUIncludesInitContainers(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{hcuContainer("init0", true)},
		Containers:     []corev1.Container{hcuContainer("app0", false)},
	}}
	if !RequestsHCU(pod) {
		t.Fatalf("RequestsHCU should consider InitContainers")
	}

	none := &corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{hcuContainer("init0", false)},
		Containers:     []corev1.Container{hcuContainer("app0", false)},
	}}
	if RequestsHCU(none) {
		t.Fatalf("RequestsHCU got true, want false")
	}
}

func TestRequestsVirtualHCUIncludesInitContainers(t *testing.T) {
	init := hcuContainer("init0", false)
	init.Resources.Limits = corev1.ResourceList{"hygon.com/hcumem": resource.MustParse("1000")}

	pod := &corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{init},
		Containers:     []corev1.Container{hcuContainer("app0", false)},
	}}
	if !RequestsVirtualHCU(pod) {
		t.Fatalf("RequestsVirtualHCU should consider InitContainers")
	}
}
