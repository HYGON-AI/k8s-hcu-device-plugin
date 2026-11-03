/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import "testing"

func TestInitResourcePrefix(t *testing.T) {
	t.Cleanup(func() {
		_ = InitResourcePrefix(DefaultResourcePrefix)
	})

	if err := InitResourcePrefix("*cu"); err != nil {
		t.Fatalf("InitResourcePrefix(*cu): %v", err)
	}
	if ResourcePrefix() != "*cu" {
		t.Fatalf("ResourcePrefix=%q", ResourcePrefix())
	}
	if DeviceIDPrefix() != "*CU" {
		t.Fatalf("DeviceIDPrefix=%q", DeviceIDPrefix())
	}
	if ResourceNameNum() != "*cunum" || ResourceNameCores() != "*cucores" || ResourceNameMem() != "*cumem" {
		t.Fatalf("resource names: %s %s %s", ResourceNameNum(), ResourceNameCores(), ResourceNameMem())
	}
	if FormatDeviceUUID("ABC") != "*CU-ABC" {
		t.Fatalf("FormatDeviceUUID=%q", FormatDeviceUUID("ABC"))
	}
	if InRequestDevices[HygonHCUDevice] != "hami.io/*cu-devices-to-allocate" {
		t.Fatalf("InRequestDevices=%q", InRequestDevices[HygonHCUDevice])
	}
	if SupportDevices[HygonHCUDevice] != "hami.io/*cu-devices-allocated" {
		t.Fatalf("SupportDevices=%q", SupportDevices[HygonHCUDevice])
	}
	if HandshakeAnnosString != "hami.io/node-handshake-*cu" {
		t.Fatalf("HandshakeAnnosString=%q", HandshakeAnnosString)
	}
	if RegisterAnnos != "hami.io/node-*cu-register" || HygonRegisterAnnos != "hygon.com/node-*cu-register" {
		t.Fatalf("register annos: %s %s", RegisterAnnos, HygonRegisterAnnos)
	}

	if err := InitResourcePrefix("gpu"); err == nil {
		t.Fatalf("expected error for invalid prefix")
	}

	if err := InitResourcePrefix("HCU"); err != nil {
		t.Fatalf("InitResourcePrefix(HCU): %v", err)
	}
	if ResourcePrefix() != "hcu" || DeviceIDPrefix() != "HCU" {
		t.Fatalf("case fold failed: %s %s", ResourcePrefix(), DeviceIDPrefix())
	}
}

func TestGetHAMiResourceNameUsesPrefix(t *testing.T) {
	t.Cleanup(func() {
		_ = InitResourcePrefix(DefaultResourcePrefix)
	})
	_ = InitResourcePrefix("*cu")
	// policy 0 does not call DCGM for the name itself when returning ResourceNameNum.
	if got := GetHAMiResourceName("0"); got != "*cunum" {
		t.Fatalf("GetHAMiResourceName(0)=%q, want *cunum", got)
	}
	if got := GetResourceNamePrefix("0"); got != "*cu" {
		t.Fatalf("GetResourceNamePrefix(0)=%q, want *cu", got)
	}
}
