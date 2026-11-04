/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import (
	"fmt"
	"strings"
)

const (
	// DefaultResourcePrefix is the product-family prefix used in K8s resource
	// names, HAMi annotations, and device UUID ids (HCU-<serial>).
	DefaultResourcePrefix = "hcu"
)

// resourcePrefix is the lowercase product-family prefix (hcu|*cu).
// Mutated only by InitResourcePrefix at process start.
var resourcePrefix = DefaultResourcePrefix

// HandshakeAnnosString / RegisterAnnos / HygonRegisterAnnos are node annotation
// keys. They are vars so InitResourcePrefix can rebuild them for *cu compat.
var (
	HandshakeAnnosString = "hami.io/node-handshake-hcu"
	RegisterAnnos        = "hami.io/node-hcu-register"
	HygonRegisterAnnos   = "hygon.com/node-hcu-register"
)

// InitResourcePrefix validates and applies RESOURCE_PREFIX (hcu|*cu).
// Must be called once at startup before device plugins register resources.
func InitResourcePrefix(prefix string) error {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		prefix = DefaultResourcePrefix
	}
	if !strings.Contains(prefix, "cu") {
		return fmt.Errorf("invalid resource-prefix %q, want hcu or *cu", prefix)
	}
	resourcePrefix = prefix
	applyResourcePrefix(prefix)
	return nil
}

func applyResourcePrefix(prefix string) {
	HandshakeAnnosString = fmt.Sprintf("hami.io/node-handshake-%s", prefix)
	RegisterAnnos = fmt.Sprintf("hami.io/node-%s-register", prefix)
	HygonRegisterAnnos = fmt.Sprintf("hygon.com/node-%s-register", prefix)

	if InRequestDevices == nil {
		InRequestDevices = make(map[string]string)
	}
	if SupportDevices == nil {
		SupportDevices = make(map[string]string)
	}
	if HandshakeAnnos == nil {
		HandshakeAnnos = make(map[string]string)
	}
	InRequestDevices[HygonHCUDevice] = fmt.Sprintf("hami.io/%s-devices-to-allocate", prefix)
	SupportDevices[HygonHCUDevice] = fmt.Sprintf("hami.io/%s-devices-allocated", prefix)
	HandshakeAnnos[HygonHCUDevice] = HandshakeAnnosString
}

// ResourcePrefix returns the lowercase product-family prefix (hcu|*cu).
func ResourcePrefix() string {
	return resourcePrefix
}

// DeviceIDPrefix returns the uppercase UUID prefix (HCU|*CU).
func DeviceIDPrefix() string {
	return strings.ToUpper(resourcePrefix)
}

// ResourceNameNum returns hcunum / *cunum.
func ResourceNameNum() string {
	return resourcePrefix + "num"
}

// ResourceNameCores returns hcucores / *cucores.
func ResourceNameCores() string {
	return resourcePrefix + "cores"
}

// ResourceNameMem returns hcumem / *cumem.
func ResourceNameMem() string {
	return resourcePrefix + "mem"
}

// FormatDeviceUUID builds PREFIX-<serial> (e.g. HCU-xxx / *CU-xxx).
func FormatDeviceUUID(serial string) string {
	return DeviceIDPrefix() + "-" + serial
}

// DeviceUUIDScanFormat is the Sscanf pattern for FormatDeviceUUID values.
func DeviceUUIDScanFormat() string {
	return DeviceIDPrefix() + "-%s"
}
