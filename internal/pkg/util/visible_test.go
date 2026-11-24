/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import (
	"testing"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
)

func sampleDevices() map[string]dcgm.DeviceInfo {
	return map[string]dcgm.DeviceInfo{
		"0000:06:00.0": {DvInd: 0, PciBusNumber: "0000:06:00.0"},
		"0000:09:00.0": {DvInd: 1, PciBusNumber: "0000:09:00.0"},
		"0000:0c:00.0": {DvInd: 2, PciBusNumber: "0000:0c:00.0"},
	}
}

func TestFilterPhysicalDevices(t *testing.T) {
	devs := sampleDevices()

	if got := filterPhysicalDevices(devs, ""); len(got) != 3 {
		t.Fatalf("empty spec: want 3, got %d", len(got))
	}
	if got := filterPhysicalDevices(devs, "ALL"); len(got) != 3 {
		t.Fatalf("all spec: want 3, got %d", len(got))
	}
	if got := filterPhysicalDevices(devs, "none"); len(got) != 0 {
		t.Fatalf("none spec: want 0, got %d", len(got))
	}

	gotIdx := filterPhysicalDevices(devs, "0, 2")
	if len(gotIdx) != 2 {
		t.Fatalf("index spec: want 2, got %d", len(gotIdx))
	}
	if _, ok := gotIdx["0000:06:00.0"]; !ok {
		t.Fatalf("expected 0000:06:00.0 in index filter")
	}
	if _, ok := gotIdx["0000:0c:00.0"]; !ok {
		t.Fatalf("expected 0000:0c:00.0 in index filter")
	}

	gotPCI := filterPhysicalDevices(devs, "0000:09:00.0,0C:00.0")
	if len(gotPCI) != 2 {
		t.Fatalf("pci spec: want 2, got %d", len(gotPCI))
	}
	if _, ok := gotPCI["0000:09:00.0"]; !ok {
		t.Fatalf("expected 0000:09:00.0 in pci filter")
	}
	if _, ok := gotPCI["0000:0c:00.0"]; !ok {
		t.Fatalf("expected 0000:0c:00.0 in pci filter")
	}
}

func TestResolveVisibleSpecPriority(t *testing.T) {
	t.Cleanup(func() {
		SetVisibleDevices("")
	})

	SetVisibleDevices("")
	t.Setenv(envDCUVisibleDevices, "1,2")
	if got := resolveVisibleSpec(); got != "1,2" {
		t.Fatalf("DCU_VISIBLE_DEVICES fallback: want 1,2, got %q", got)
	}

	t.Setenv(envHCUVisibleDevices, "0")
	if got := resolveVisibleSpec(); got != "0" {
		t.Fatalf("HCU_VISIBLE_DEVICES precedence: want 0, got %q", got)
	}

	SetVisibleDevices("0000:06:00.0")
	if got := resolveVisibleSpec(); got != "0000:06:00.0" {
		t.Fatalf("flag precedence: want 0000:06:00.0, got %q", got)
	}
}
