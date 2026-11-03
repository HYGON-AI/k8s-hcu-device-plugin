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

	if err := InitResourcePrefix("dcu"); err != nil {
		t.Fatalf("InitResourcePrefix(dcu): %v", err)
	}
	if ResourcePrefix() != "dcu" {
		t.Fatalf("ResourcePrefix=%q", ResourcePrefix())
	}
	if DeviceIDPrefix() != "DCU" {
		t.Fatalf("DeviceIDPrefix=%q", DeviceIDPrefix())
	}
	if ResourceNameNum() != "dcunum" || ResourceNameCores() != "dcucores" || ResourceNameMem() != "dcumem" {
		t.Fatalf("resource names: %s %s %s", ResourceNameNum(), ResourceNameCores(), ResourceNameMem())
	}
	if FormatDeviceUUID("ABC") != "DCU-ABC" {
		t.Fatalf("FormatDeviceUUID=%q", FormatDeviceUUID("ABC"))
	}
	if InRequestDevices[HygonHCUDevice] != "hami.io/dcu-devices-to-allocate" {
		t.Fatalf("InRequestDevices=%q", InRequestDevices[HygonHCUDevice])
	}
	if SupportDevices[HygonHCUDevice] != "hami.io/dcu-devices-allocated" {
		t.Fatalf("SupportDevices=%q", SupportDevices[HygonHCUDevice])
	}
	if HandshakeAnnosString != "hami.io/node-handshake-dcu" {
		t.Fatalf("HandshakeAnnosString=%q", HandshakeAnnosString)
	}
	if RegisterAnnos != "hami.io/node-dcu-register" || HygonRegisterAnnos != "hygon.com/node-dcu-register" {
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
	_ = InitResourcePrefix("dcu")
	// policy 0 does not call DCGM for the name itself when returning ResourceNameNum.
	if got := GetHAMiResourceName("0"); got != "dcunum" {
		t.Fatalf("GetHAMiResourceName(0)=%q, want dcunum", got)
	}
	if got := GetResourceNamePrefix("0"); got != "dcu" {
		t.Fatalf("GetResourceNamePrefix(0)=%q, want dcu", got)
	}
}
