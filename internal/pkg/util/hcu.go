/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package util

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"
	corev1 "k8s.io/api/core/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const dcgmDeviceReconcileInterval = 30 * time.Second

var dcgmReconcileMu sync.Mutex

// CountChengduDevicesByLspci counts Chengdu Display/Co-processor devices via lspci.
func CountChengduDevicesByLspci() (int, error) {
	cmd := exec.Command("sh", "-c", "lspci | grep Display | grep Chengdu || lspci | grep Co-processor | grep Chengdu")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return 0, nil
		}
		return 0, fmt.Errorf("lspci query failed: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return 0, nil
	}
	return len(strings.Split(trimmed, "\n")), nil
}

// hasIncompletePhysicalDevices reports devices visible via RSMI but missing DMI fields.
func hasIncompletePhysicalDevices() bool {
	devices, err := dcgm.DeviceInfos()
	if err != nil {
		log.Errorf("DeviceInfos failed while checking completeness: %v", err)
		return true
	}
	for _, d := range devices {
		if d.MemoryTotal > 0 && (d.ComputeUnit <= 0 || d.DevTypeName == "") {
			log.V(2).Infof("Incomplete HCU detected: id=%s cu=%v type=%q", d.DeviceId, d.ComputeUnit, d.DevTypeName)
			return true
		}
	}
	return false
}

// ReconcileDCGMDevices compares DCGM/DMI device counts with lspci;
// if mismatched or DMI info is incomplete, shuts down and re-initializes DCGM via Init().
func ReconcileDCGMDevices() {
	dcgmReconcileMu.Lock()
	defer dcgmReconcileMu.Unlock()

	rsmiCount, err := dcgm.NumMonitorDevices()
	if err != nil {
		log.Errorf("NumMonitorDevices failed: %v", err)
		return
	}

	lspciCount, err := CountChengduDevicesByLspci()
	if err != nil {
		log.Errorf("CountChengduDevicesByLspci failed: %v", err)
		return
	}

	log.V(3).Infof("DCGM  get  hcu  count ::: rsmi=%d       lspci=%d ", rsmiCount, lspciCount)
	if rsmiCount == lspciCount {
		return
	}

	log.Warningf("DCGM out of sync (rsmi=%d  lspci=%d ), reinitializing", rsmiCount, lspciCount)
	if err := dcgm.ShutDown(); err != nil {
		log.Errorf("DCGM ShutDown failed: %v", err)
	}
	if err := dcgm.Init(); err != nil { // just Initialize
		log.Errorf("DCGM Init failed: %v", err)
		return
	}
	log.V(2).Infof("DCGM reinitialized successfully")
}

// StartDCGMDeviceReconcileLoop starts a background loop that reconciles DCGM devices periodically.
func StartDCGMDeviceReconcileLoop() {
	go func() {
		ticker := time.NewTicker(dcgmDeviceReconcileInterval)
		defer ticker.Stop()
		for range ticker.C {
			ReconcileDCGMDevices()
		}
	}()
	log.V(2).Infof("Started DCGM device reconcile loop, interval: %v", dcgmDeviceReconcileInterval)
}

// ResolveDevTypeName returns DMI type name, falling back to RSMI device name when empty.
func ResolveDevTypeName(val dcgm.DeviceInfo) string {
	if val.DevTypeName != "" {
		return val.DevTypeName
	}
	if name, err := dcgm.DevName(val.DvInd); err == nil && name != "" {
		return dcgm.NormalizeDevTypeName(name)
	}
	return ""
}

const RESOURCE_REGISTER_STRATEGY = "RESOURCE_REGISTER_STRATEGY"

// GetCardAndRender get HCU render and card index from driver module
func GetCardAndRender(pcieAddress string) ([]string, error) {
	pattern := filepath.Join(
		"/sys/module",
		"*",
		"drivers",
		"pci:*",
		pcieAddress,
		"drm",
	)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern %s error: %w", pattern, err)
	}

	for _, dirPath := range matches {
		// directory structure：
		// /sys/module/<module>/drivers/pci:<driver>/<pcieAddress>/drm
		rel, err := filepath.Rel("/sys/module", dirPath)
		if err != nil {
			continue
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 5 {
			continue
		}

		moduleName := parts[0]
		driverName := strings.TrimPrefix(parts[2], "pci:")

		// The module and PCI driver names must be exactly the same.
		if moduleName != driverName {
			continue
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			log.Errorf("read directory %s error: %v", dirPath, err)
			return nil, fmt.Errorf("read directory %s error: %w", dirPath, err)
		}

		result := make([]string, 0, len(entries))
		for _, entry := range entries {
			result = append(result, entry.Name())
		}

		return result, nil
	}

	log.Errorf("no matching DRM device found for PCI address %s", pcieAddress)
	return nil, fmt.Errorf(
		"no matching DRM device found for PCI address %s",
		pcieAddress,
	)
}

// SimpleHealthCheck check HCU healthy by index
func SimpleHealthCheck(statuses []dcgm.HCUHealthStatus, idx uint) bool {
	for _, status := range statuses {
		if status.HCU == idx {
			return strings.EqualFold(status.Status, "Healthy")
		}
	}
	return false
}

func GetNumaNode(index int) (*pluginapi.TopologyInfo, error) {
	numas, err := dcgm.ShowNumaTopology([]int{index})
	log.V(3).Infof("Watching HCU with HCU Index: %d NUMA Node: %+v", index, numas)
	if err != nil || len(numas) == 0 {
		log.Errorf("Get NUMA info error: %v", err)
		return &pluginapi.TopologyInfo{}, err
	}

	numaNodes := make([]*pluginapi.NUMANode, len(numas))
	for j, v := range numas {
		numaNodes[j] = &pluginapi.NUMANode{
			ID: int64(v.NumaNode),
		}
	}

	return &pluginapi.TopologyInfo{
		Nodes: numaNodes,
	}, nil
}

// GetAllVirtualHCUs get vHCUs in map format
func GetAllVirtualHCUs() map[string][]dcgm.VDeviceInfo {
	virtualHCUInfos, err := dcgm.VDeviceInfos()
	if err != nil {
		log.Errorf("Get device infos error: %v ", err)
	}
	log.V(2).Infof("Get Virtual HCU number : %d \n", len(virtualHCUInfos))

	virtualHCUs := make(map[string][]dcgm.VDeviceInfo)
	for _, virtualHCU := range virtualHCUInfos {
		cu_string := strconv.Itoa(virtualHCU.VComputeUnitCount) + "c"
		mem_string := strconv.Itoa(int(math.Round(float64(virtualHCU.VMemoryTotal)/(1024*1024*1024)))) + "g"
		vkey := "share-" + cu_string + "-" + mem_string
		_, exists := virtualHCUs[vkey]
		if exists {
			virtualHCUs[vkey] = append(virtualHCUs[vkey], virtualHCU)
		} else {
			virtualHCUs[vkey] = []dcgm.VDeviceInfo{virtualHCU}
		}
	}
	return virtualHCUs
}

// GetAllMigHCUs get all MIG HCUs in map format
func GetAllMigHCUs() map[string][]dcgm.MigInfo {
	migDeviceInfos, err := dcgm.MigInfos()
	if err != nil {
		log.Errorf("Get device infos error: %v ", err)
	}
	log.V(2).Infof("Get MIG HCUs number: %d \n", len(migDeviceInfos))

	migHCUs := make(map[string][]dcgm.MigInfo)
	for _, migDeviceInfo := range migDeviceInfos {
		migDeviceName := migDeviceInfo.Name
		migDeviceProfile := strings.TrimPrefix(migDeviceName, "MIG ")
		migDeviceProfile = strings.ReplaceAll(migDeviceProfile, ".", "-")
		migKey := "mig-" + migDeviceProfile
		_, exists := migHCUs[migKey]
		if exists {
			migHCUs[migKey] = append(migHCUs[migKey], migDeviceInfo)
		} else {
			migHCUs[migKey] = []dcgm.MigInfo{migDeviceInfo}
		}
	}
	return migHCUs
}

// GetAllPhysicalDevices Get All Physical HCUs
func GetAllPhysicalDevices() map[string]dcgm.DeviceInfo {
	physicalDeviceInfos := make(map[string]dcgm.DeviceInfo)
	hcuDeviceInfos, err := dcgm.DeviceInfos()
	if err != nil {
		log.Error("Error get physical HCU device information")
	}
	for _, hcuDeviceInfo := range hcuDeviceInfos {
		physicalDeviceInfos[hcuDeviceInfo.PciBusNumber] = hcuDeviceInfo
	}
	return physicalDeviceInfos
}

func RequestsHCU(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		for resourceName, quantity := range container.Resources.Limits {
			if (strings.Contains(string(resourceName), "hygon.com")) && quantity.Value() > 0 {
				return true
			}
		}
	}
	return false
}

func RequestsVirtualHCU(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		for resourceName, quantity := range container.Resources.Limits {
			if strings.Contains(string(resourceName), "hygon.com") && strings.Contains(string(resourceName), "hcumem") && quantity.Value() > 0 {
				return true
			}
		}
	}
	return false
}

func GetResourceNamePrefix(policy string) string {
	info, err := dcgm.GetDeviceInfo(0)
	if err != nil {
		log.Errorf("Get device info err: %v", err)
	}

	if policy == "1" {
		return info.Name
	}

	if policy == "2" {
		cu_string := strconv.Itoa(info.ComputeUnitCount) + "CU"
		mem_string := strconv.Itoa(int(math.Round(float64(info.GlobalMemSize)/(1024*1024*1024)))) + "G"
		return strings.ReplaceAll(info.Name, "_", "-") + "_" + mem_string + "_" + cu_string
	}
	return "hcu"
}

func GetHAMiResourceName(policy string) string {
	info, err := dcgm.GetDeviceInfo(0)
	if err != nil {
		log.Errorf("Get device info err: %v", err)
	}

	if policy == "1" {
		return info.Name + "_hcunum"
	}

	if policy == "2" {
		cu_string := strconv.Itoa(info.ComputeUnitCount) + "CU"
		mem_string := strconv.Itoa(int(math.Round(float64(info.GlobalMemSize)/(1024*1024*1024)))) + "G"
		return strings.ReplaceAll(info.Name, "_", "-") + "_" + mem_string + "_" + cu_string + "_hcunum"
	}
	return "hcunum"
}
