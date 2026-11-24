/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import (
	"os"
	"strconv"
	"strings"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
)

const (
	envHCUVisibleDevices = "HCU_VISIBLE_DEVICES"
	envDCUVisibleDevices = "DCU_VISIBLE_DEVICES"
)

var visibleDevicesFlag string

// SetVisibleDevices sets the CLI/ENV-configured visible devices expression.
func SetVisibleDevices(visible string) {
	visibleDevicesFlag = strings.TrimSpace(visible)
}

func resolveVisibleSpec() string {
	if visibleDevicesFlag != "" {
		return visibleDevicesFlag
	}
	if v := strings.TrimSpace(os.Getenv(envHCUVisibleDevices)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(envDCUVisibleDevices)); v != "" {
		return v
	}
	return ""
}

func normalizePCIAddress(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	return strings.TrimPrefix(addr, "0000:")
}

// filterPhysicalDevices filters physical devices by a comma-separated whitelist of
// device indices (DvInd, e.g. "0,1") or PCI bus IDs (e.g. "0000:06:00.0" or "06:00.0").
// Empty spec or "all" keeps all devices; "none" keeps none.
func filterPhysicalDevices(devices map[string]dcgm.DeviceInfo, spec string) map[string]dcgm.DeviceInfo {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.EqualFold(spec, "all") {
		return devices
	}
	if strings.EqualFold(spec, "none") {
		return make(map[string]dcgm.DeviceInfo)
	}

	allowedIndices := make(map[int]struct{})
	allowedPCIs := make(map[string]struct{})
	for _, raw := range strings.Split(spec, ",") {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		if idx, err := strconv.Atoi(token); err == nil {
			allowedIndices[idx] = struct{}{}
			continue
		}
		allowedPCIs[normalizePCIAddress(token)] = struct{}{}
	}

	filtered := make(map[string]dcgm.DeviceInfo, len(devices))
	for busID, dev := range devices {
		if _, ok := allowedIndices[dev.DvInd]; ok {
			filtered[busID] = dev
			continue
		}
		if _, ok := allowedPCIs[normalizePCIAddress(busID)]; ok {
			filtered[busID] = dev
		}
	}
	return filtered
}
