/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package plugin

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"
	hmutil "github.com/Project-HAMi/HAMi/pkg/util"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// NodeLockHCU should same with hami scheduler hygon device NodeLockHCU
// there is a bug with nodelock package utils, the key is hard coded as "hami.io/mutex.lock"
// so we can only use this value now.
const (
	NodeLockHCU = "hami.io/mutex.lock"
)

type DevicePlugin struct {
	// mu guards the four device maps below, which are written by ListAndWatch /
	// WatchAndRegister goroutines and read concurrently by Allocate RPCs.
	mu          sync.RWMutex
	HCUs        map[string]dcgm.DeviceInfo
	VirtualHCUs map[string]dcgm.VDeviceInfo
	MigHCUs     map[string]dcgm.MigInfo
	HAMiHCUs    map[string]dcgm.DeviceInfo
	Heartbeat   chan bool
	signal      chan os.Signal
	Resource    string
}

func (p *DevicePlugin) setHCUs(devices map[string]dcgm.DeviceInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HCUs = devices
}

func (p *DevicePlugin) getHCUs() map[string]dcgm.DeviceInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.HCUs
}

func (p *DevicePlugin) setVirtualHCUs(devices map[string]dcgm.VDeviceInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.VirtualHCUs = devices
}

func (p *DevicePlugin) getVirtualHCU(id string) (dcgm.VDeviceInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	device, ok := p.VirtualHCUs[id]
	return device, ok
}

func (p *DevicePlugin) setMigHCUs(devices map[string]dcgm.MigInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.MigHCUs = devices
}

func (p *DevicePlugin) getMigHCU(id string) (dcgm.MigInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	device, ok := p.MigHCUs[id]
	return device, ok
}

func (p *DevicePlugin) setHAMiHCUs(devices map[string]dcgm.DeviceInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HAMiHCUs = devices
}

func (p *DevicePlugin) getHAMiHCU(id string) (dcgm.DeviceInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	device, ok := p.HAMiHCUs[id]
	return device, ok
}

type DevicePluginOption func(*DevicePlugin)

func NewDevicePlugin(options ...DevicePluginOption) *DevicePlugin {
	DevicePlugin := &DevicePlugin{}
	for _, option := range options {
		option(DevicePlugin)
	}
	return DevicePlugin
}

func WithHeartbeat(ch chan bool) DevicePluginOption {
	return func(p *DevicePlugin) {
		p.Heartbeat = ch
	}
}
func WithResource(res string) DevicePluginOption {
	return func(p *DevicePlugin) {
		p.Resource = res
	}
}

// Start is an optional interface that could be implemented by plugin.
// If case Start is implemented, it will be executed by Manager after
// plugin instantiation and before its registration to kubelet. This
// method could be used to prepare resources before they are offered
// to Kubernetes.
func (p *DevicePlugin) Start() error {
	p.signal = make(chan os.Signal, 1)
	signal.Notify(p.signal, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// refresh container devices and node annotation
	strategy := os.Getenv(util.RESOURCE_REGISTER_STRATEGY)
	if strategy == "hami" {
		go p.WatchAndRegister()
		go p.informerPodHandler()
	}
	return nil
}

// Stop is an optional interface that could be implemented by plugin.
// If case Stop is implemented, it will be executed by Manager after the
// plugin is unregistered from kubelet. This method could be used to tear
// down resources.
func (p *DevicePlugin) Stop() error {
	return nil
}

// GetDevicePluginOptions returns options to be communicated with Device
// Manager
func (p *DevicePlugin) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		GetPreferredAllocationAvailable: true,
	}, nil
}

// PreStartContainer is expected to be called before each container start if indicated by plugin during registration phase.
// PreStartContainer allows kubelet to pass reinitialized devices to containers.
// PreStartContainer allows Device Plugin to run device specific operations on the Devices requested
func (p *DevicePlugin) PreStartContainer(ctx context.Context, r *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// ListAndWatch returns a stream of List of Devices
// Whenever a Device state change or a Device disappears, ListAndWatch
// returns the new list
func (p *DevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {

	resourceNamePrefix := util.GetResourceNamePrefix(os.Getenv("POLICY"))
	if p.Resource == resourceNamePrefix {
		return p.ListAndWatchHCUs(e, s)
	}

	if strings.Contains(p.Resource, "share") {
		return p.ListAndWatchVirtualHCUs(e, s)
	}

	if strings.Contains(p.Resource, "mig") {
		return p.ListAndWatchMigHCUs(e, s)
	}

	if strings.Contains(p.Resource, "hcunum") {
		return p.ListAndWatchHAMiHCUs(e, s)
	}
	if strings.Contains(p.Resource, "hcucores") {
		return p.ListAndWatchHCUCores(e, s)
	}
	return nil
}

// refreshDeviceHealth updates devs in place from a fresh DCGM health query.
// Device indexes come from a caller-owned snapshot so the callback never reads
// the shared device maps, and a failed query leaves the previous health intact
// rather than marking the whole node unhealthy.
func refreshDeviceHealth(kind string, devs []*pluginapi.Device, dvInd map[string]int) {
	log.V(2).Infof("Starting %s health check", kind)

	healthList, err := dcgm.HCUHealthCheck()
	if err != nil {
		log.Errorf("%s health check failed, keeping previous health state: %v", kind, err)
		return
	}

	for _, dev := range devs {
		idx, ok := dvInd[dev.ID]
		if !ok {
			continue
		}
		if util.SimpleHealthCheck(healthList, uint(idx)) {
			dev.Health = pluginapi.Healthy
		} else {
			dev.Health = pluginapi.Unhealthy
			log.Errorf("%s health check failed for device %s", kind, dev.ID)
		}
	}

	log.V(2).Infof("Finishing %s health check", kind)
}

// ListAndWatchHCUs
func (p *DevicePlugin) ListAndWatchHCUs(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	strategy := os.Getenv(util.RESOURCE_REGISTER_STRATEGY)

	hcus := util.GetAllPhysicalDevices()
	for deviceID, physicalDevice := range hcus {
		if strategy == "mixed" && physicalDevice.VDeviceCount > 0 {
			delete(hcus, deviceID)
		}
	}
	p.setHCUs(hcus)

	log.V(2).Infof("Found %d HCUs", len(hcus))

	devs := buildDevicesWithNUMAFromHCUInfo(hcus)
	dvInd := make(map[string]int, len(hcus))
	for id, device := range hcus {
		dvInd[id] = device.DvInd
	}

	return p.watchDevices(s, devs, func(devs []*pluginapi.Device) {
		refreshDeviceHealth("HCU", devs, dvInd)
	})
}

func (p *DevicePlugin) ListAndWatchHCUCores(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	deviceCount, _ := dcgm.DeviceCount()
	devs := make([]*pluginapi.Device, deviceCount*100)
	func() {
		i := 0
		for id := 0; id < deviceCount*100; id++ {
			dev := &pluginapi.Device{
				ID:     strconv.Itoa(id),
				Health: pluginapi.Healthy,
			}

			devs[i] = dev
			i++
		}
	}()

	return p.watchDevices(s, devs, nil)
}

func (p *DevicePlugin) ListAndWatchVirtualHCUs(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {

	allVirtualHCUs := util.GetAllVirtualHCUs()
	resourceName := "share" + strings.Split(p.Resource, "share")[1]

	virtualHCUs := make(map[string]dcgm.VDeviceInfo)
	if _, exists := allVirtualHCUs[resourceName]; exists {
		virtualHCUs = make(map[string]dcgm.VDeviceInfo, len(allVirtualHCUs[resourceName]))
		for _, virtualHCU := range allVirtualHCUs[resourceName] {
			virtualHCUs["vdev"+strconv.Itoa(virtualHCU.VdvInd)] = virtualHCU
		}
	}
	p.setVirtualHCUs(virtualHCUs)

	log.V(2).Infof("Found %d Virtual HCUs", len(virtualHCUs))

	devs := buildDevicesWithNUMAFromVDeviceInfo(virtualHCUs)
	dvInd := make(map[string]int, len(virtualHCUs))
	for id, device := range virtualHCUs {
		dvInd[id] = device.DvInd
	}

	return p.watchDevices(s, devs, func(devs []*pluginapi.Device) {
		refreshDeviceHealth("Virtual HCU", devs, dvInd)
	})
}

func (p *DevicePlugin) ListAndWatchMigHCUs(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {

	allMigHCUs := util.GetAllMigHCUs()
	resourceName := "mig" + strings.Split(p.Resource, "mig")[1]

	migHCUs := make(map[string]dcgm.MigInfo)
	if _, exists := allMigHCUs[resourceName]; exists {
		migHCUs = make(map[string]dcgm.MigInfo, len(allMigHCUs[resourceName]))
		for _, migHCU := range allMigHCUs[resourceName] {
			migHCUs[migHCU.UUID] = migHCU
		}
	}
	p.setMigHCUs(migHCUs)

	log.V(2).Infof("Found %d MIG HCUs", len(migHCUs))

	devs := buildDevicesWithNUMAFromMigInfo(migHCUs)
	dvInd := make(map[string]int, len(migHCUs))
	for id, device := range migHCUs {
		dvInd[id] = device.DvInd
	}

	return p.watchDevices(s, devs, func(devs []*pluginapi.Device) {
		refreshDeviceHealth("MIG HCU", devs, dvInd)
	})
}

func (p *DevicePlugin) ListAndWatchHAMiHCUs(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {

	allPhysicalDevices := util.GetAllPhysicalDevices()
	allHAMiHCUs := make(map[string]dcgm.DeviceInfo, len(allPhysicalDevices)*DeviceSplitCount)
	for _, physicalDevice := range allPhysicalDevices {
		for idx := 0; idx < DeviceSplitCount; idx++ {
			allHAMiHCUs["HCU-"+physicalDevice.DeviceId+"-fake-"+strconv.Itoa(idx)] = physicalDevice
		}
	}
	p.setHAMiHCUs(allHAMiHCUs)

	log.V(2).Infof("Found %d HAMi HCUs", len(allHAMiHCUs))

	devs := buildDevicesWithNUMAFromHCUInfo(allHAMiHCUs)
	dvInd := make(map[string]int, len(allHAMiHCUs))
	for id, device := range allHAMiHCUs {
		dvInd[id] = device.DvInd
	}

	return p.watchDevices(s, devs, func(devs []*pluginapi.Device) {
		refreshDeviceHealth("HAMi HCU", devs, dvInd)
	})
}

// watchDevices sends initial device list and then handles heartbeat and signal events.
// If updateHealth is not nil, it will be called on every heartbeat to refresh device health.
func (p *DevicePlugin) watchDevices(
	s pluginapi.DevicePlugin_ListAndWatchServer,
	devs []*pluginapi.Device,
	updateHealth func([]*pluginapi.Device),
) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: devs})

loop:
	for {
		select {
		case <-p.Heartbeat:
			if updateHealth != nil {
				updateHealth(devs)
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: devs})
		case <-p.signal:
			log.V(2).Infof("Received signal, exiting")
			break loop
		}
	}

	// returning a value with this function will unregister the plugin from k8s
	return nil
}

// GetPreferredAllocation returns a preferred set of devices to allocate
// from a list of available ones. The resulting preferred allocation is not
// guaranteed to be the allocation ultimately performed by the
// devicemanager. It is only designed to help the devicemanager make a more
// informed allocation decision when possible.
func (p *DevicePlugin) GetPreferredAllocation(ctx context.Context, req *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// Allocate is called during container creation so that the Device
// Plugin can run device specific operations and instruct Kubelet
// of the steps to make the Device available in the container
func (p *DevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resourceNamePrefix := util.GetResourceNamePrefix(os.Getenv("POLICY"))
	if p.Resource == resourceNamePrefix {
		return p.AllocateHCUs(ctx, r)
	}

	if strings.Contains(p.Resource, "share") {
		return p.AllocateVirtualHCUs(ctx, r)
	}

	if strings.Contains(p.Resource, "mig") {
		return p.AllocateMigHCUs(ctx, r)
	}

	if strings.Contains(p.Resource, "hcunum") {
		return p.AllocateHAMiHCUs(ctx, r)
	}

	if strings.Contains(p.Resource, "hcucores") {
		return p.AllocateCores(ctx, r)
	}

	return &pluginapi.AllocateResponse{}, nil
}

func (p *DevicePlugin) AllocateHCUs(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	for _, req := range r.ContainerRequests {
		car := pluginapi.ContainerAllocateResponse{}

		addCommonDevicesAndMounts(&car)

		for _, id := range req.DevicesIDs {
			log.V(2).Infof("Allocating device Bus ID: %s", id)
			cardAndRenderNames, err := util.GetCardAndRender(id)
			if err != nil {
				log.Errorf("Device Card and Render Found Error by BUS id %s, Error:%v", id, err)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("device Card and Render Found Error by BUS id %s, Error:%v", id, err)
			}
			for _, devPath := range cardAndRenderNames {
				devCardPath := "/dev/dri/" + devPath
				devCard := new(pluginapi.DeviceSpec)
				devCard.HostPath = devCardPath
				devCard.ContainerPath = devCardPath
				devCard.Permissions = "rw"
				car.Devices = append(car.Devices, devCard)
			}
		}

		response.ContainerResponses = append(response.ContainerResponses, &car)
	}

	return &response, nil
}

func (p *DevicePlugin) AllocateVirtualHCUs(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	for _, req := range r.ContainerRequests {
		car := pluginapi.ContainerAllocateResponse{}

		addCommonDevicesAndMounts(&car)

		// Mount requested devices
		if len(req.DevicesIDs) > 1 {
			log.Warningf("In the beta version, each container is allowed to use only one share device. ")
			log.Warningf("The first configuration files will take effect, while the others will not take effect.")
		}

		for _, id := range req.DevicesIDs {
			log.V(2).Infof("Allocating device ID: %s", id)
			virtualHCU, ok := p.getVirtualHCU(id)
			if !ok {
				log.Errorf("Virtual HCU %s not found in mapper", id)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("virtual HCU %s not found in mapper", id)
			}
			cardAndRenderNames, err := util.GetCardAndRender(virtualHCU.PciBusNumber)
			if err != nil {
				log.Errorf("Device Card and Render Found Error by BUS id %s, Error:%v", virtualHCU.PciBusNumber, err)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("device Card and Render Found Error by BUS id %s, Error:%v", virtualHCU.PciBusNumber, err)
			}
			for _, devPath := range cardAndRenderNames {
				devCardPath := "/dev/dri/" + devPath
				devCard := new(pluginapi.DeviceSpec)
				devCard.HostPath = devCardPath
				devCard.ContainerPath = devCardPath
				devCard.Permissions = "rw"
				car.Devices = append(car.Devices, devCard)
			}

			mount := new(pluginapi.Mount)
			hostpath := fmt.Sprintf("/etc/vdev/%s.conf", id)
			containerpath := fmt.Sprintf("/etc/vdev/docker/%s.conf", id)
			mount.HostPath = hostpath
			mount.ContainerPath = containerpath
			mount.ReadOnly = true
			car.Mounts = append(car.Mounts, mount)
		}

		response.ContainerResponses = append(response.ContainerResponses, &car)
	}

	return &response, nil
}

func (p *DevicePlugin) AllocateMigHCUs(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	for _, req := range r.ContainerRequests {
		car := pluginapi.ContainerAllocateResponse{}

		addCommonDevicesAndMounts(&car)

		for _, id := range req.DevicesIDs {
			migHCU, ok := p.getMigHCU(id)
			if !ok {
				log.Errorf("MIG device %s not found in mapper", id)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("MIG device %s not found in mapper", id)
			}
			log.V(2).Infof("Allocating MIG device %s with physical device %s", id, migHCU.PciBusNumber)

			cardAndRenderNames, err := util.GetCardAndRender(migHCU.PciBusNumber)
			if err != nil {
				log.Errorf("Device Card and Render Found Error by BUS id %s, Error:%v", migHCU.PciBusNumber, err)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("device Card and Render Found Error by BUS id %s, Error:%v", migHCU.PciBusNumber, err)
			}
			for _, devPath := range cardAndRenderNames {
				devCardPath := "/dev/dri/" + devPath
				devCard := new(pluginapi.DeviceSpec)
				devCard.HostPath = devCardPath
				devCard.ContainerPath = devCardPath
				devCard.Permissions = "rw"
				car.Devices = append(car.Devices, devCard)
			}

			migInstance, err := dcgm.MigInfoByUUID(id)
			if err != nil {
				log.Errorf("Get MIG instance %s error: %v", id, err)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("get MIG instance %s error: %v", id, err)
			}
			hcuID := migInstance.DvInd
			giID := migInstance.GpuInstanceId
			ciID := migInstance.ComputeInstanceId
			confFileName := "dev" + strconv.Itoa(hcuID) + "gi" + strconv.FormatUint(uint64(giID), 10) + "ci" + strconv.FormatUint(uint64(ciID), 10)
			confPath := fmt.Sprintf("/etc/dmi_mig_config/ci/%s.conf", confFileName)
			car.Mounts = append(car.Mounts, &pluginapi.Mount{
				HostPath:      confPath,
				ContainerPath: confPath,
				ReadOnly:      true,
			})
		}

		response.ContainerResponses = append(response.ContainerResponses, &car)
	}

	return &response, nil
}

// addDeviceIfExists appends a device spec to the container response if the given path exists.
func addDeviceIfExists(path string, car *pluginapi.ContainerAllocateResponse) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Warningf("can not find %s.", path)
		return
	}
	if err != nil {
		log.Errorf("Error occurred when checking %s.", path)
		return
	}

	if path == "/dev/mkfd" {
		log.Infof("preparing mkfd ")
	}

	dev := &pluginapi.DeviceSpec{
		HostPath:      path,
		ContainerPath: path,
		Permissions:   "rw",
	}
	car.Devices = append(car.Devices, dev)
}

// addHyhalMount adds the read-only /opt/hyhal runtime library mount.
func addHyhalMount(car *pluginapi.ContainerAllocateResponse) {
	car.Mounts = append(car.Mounts, &pluginapi.Mount{
		ContainerPath: "/opt/hyhal",
		HostPath:      "/opt/hyhal",
		ReadOnly:      true,
	})
}

// addCommonDevicesAndMounts adds common device nodes and mounts shared by several Allocate methods.
func addCommonDevicesAndMounts(car *pluginapi.ContainerAllocateResponse) {
	addDeviceIfExists("/dev/kfd", car)
	addDeviceIfExists("/dev/mkfd", car)

	addHyhalMount(car)
}

// addCommonRWMDevices adds /dev/kfd and /dev/mkfd with rwm permissions (used by HAMi and cores allocation).
func addCommonRWMDevices(car *pluginapi.ContainerAllocateResponse) {
	paths := []string{"/dev/kfd", "/dev/mkfd"}
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Warningf("can not find %s.", path)
			continue
		} else if err != nil {
			log.Errorf("Error occurred when checking %s.", path)
			continue
		}

		if path == "/dev/mkfd" {
			log.Infof("preparing mkfd ")
		}

		dev := &pluginapi.DeviceSpec{
			HostPath:      path,
			ContainerPath: path,
			Permissions:   "rwm",
		}
		car.Devices = append(car.Devices, dev)
	}
}

// buildDevicesWithNUMAFromHCUInfo builds pluginapi.Device slice from a map keyed by ID with dcgm.DeviceInfo values.
func buildDevicesWithNUMAFromHCUInfo(devices map[string]dcgm.DeviceInfo) []*pluginapi.Device {
	devs := make([]*pluginapi.Device, len(devices))
	i := 0
	for id, device := range devices {
		dev := &pluginapi.Device{
			ID:     id,
			Health: pluginapi.Healthy,
		}

		numaInfo, err := util.GetNumaNode(device.DvInd)
		if err == nil {
			dev.Topology = numaInfo
		}

		devs[i] = dev
		i++
	}
	return devs
}

// buildDevicesWithNUMAFromVDeviceInfo builds pluginapi.Device slice from a map keyed by ID with dcgm.VDeviceInfo values.
func buildDevicesWithNUMAFromVDeviceInfo(devices map[string]dcgm.VDeviceInfo) []*pluginapi.Device {
	devs := make([]*pluginapi.Device, len(devices))
	i := 0
	for id, device := range devices {
		dev := &pluginapi.Device{
			ID:     id,
			Health: pluginapi.Healthy,
		}

		numaInfo, err := util.GetNumaNode(device.DvInd)
		if err == nil {
			dev.Topology = numaInfo
		}

		devs[i] = dev
		i++
	}
	return devs
}

// buildDevicesWithNUMAFromMigInfo builds pluginapi.Device slice from a map keyed by ID with dcgm.MigInfo values.
func buildDevicesWithNUMAFromMigInfo(devices map[string]dcgm.MigInfo) []*pluginapi.Device {
	devs := make([]*pluginapi.Device, len(devices))
	i := 0
	for id, device := range devices {
		dev := &pluginapi.Device{
			ID:     id,
			Health: pluginapi.Healthy,
		}

		numaInfo, err := util.GetNumaNode(device.DvInd)
		if err == nil {
			dev.Topology = numaInfo
		}

		devs[i] = dev
		i++
	}
	return devs
}

func (p *DevicePlugin) AllocateHAMiHCUs(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}
	nodename := util.NodeName
	current, err := hmutil.GetPendingPod(ctx, nodename)
	if err != nil {
		car := pluginapi.ContainerAllocateResponse{}
		addCommonRWMDevices(&car)
		addHyhalMount(&car)

		responses.ContainerResponses = append(responses.ContainerResponses, &car)
		return &responses, nil
	}
	log.V(2).Infof("Allocate for pod %s/%s uid [%s] \n", current.Namespace, current.Name, current.UID)

	err = util.UpdateContainerIndexAnnotations(current)
	if err != nil {
		return &pluginapi.AllocateResponse{}, err
	}

	current, err = util.GetPod(ctx, current.Namespace, current.Name)
	if err != nil {
		return &pluginapi.AllocateResponse{}, err
	}
	ctrIndex := util.GetCurrentContainerIndex(current)
	if ctrIndex < 0 || ctrIndex >= len(current.Spec.Containers) {
		log.Errorf("Invalid container index %d for pod %s/%s with %d containers",
			ctrIndex, current.Namespace, current.Name, len(current.Spec.Containers))
		util.PodAllocationFailed(nodename, current, NodeLockHCU)
		return &pluginapi.AllocateResponse{}, fmt.Errorf("invalid container index %d for pod %s/%s",
			ctrIndex, current.Namespace, current.Name)
	}

	for idx := range reqs.ContainerRequests {
		_, devreq, err := util.GetNextDeviceRequest(util.HygonHCUDevice, *current)
		log.V(2).Infoln("deviceAllocateFromAnnotation=", devreq)
		if err != nil {
			util.PodAllocationFailed(nodename, current, NodeLockHCU)
			return &pluginapi.AllocateResponse{}, err
		}
		if len(devreq) != len(reqs.ContainerRequests[idx].DevicesIDs) {
			util.PodAllocationFailed(nodename, current, NodeLockHCU)
			return &pluginapi.AllocateResponse{}, errors.New("device number not matched")
		}

		err = util.EraseNextDeviceTypeFromAnnotation(util.HygonHCUDevice, *current)
		if err != nil {
			util.PodAllocationFailed(nodename, current, NodeLockHCU)
			return &pluginapi.AllocateResponse{}, err
		}

		car := pluginapi.ContainerAllocateResponse{}
		addCommonRWMDevices(&car)
		addHyhalMount(&car)

		for _, val := range devreq {
			log.Infof("Allocating device Serial Number: %s", val.UUID)
			var devSerialNumber string
			succeedCount, err := fmt.Sscanf(val.UUID, "HCU-%s", &devSerialNumber)
			if err != nil || succeedCount == 0 || devSerialNumber == "" {
				log.Errorf("Invalid request device uuid: %s", val.UUID)
				util.PodAllocationFailed(nodename, current, NodeLockHCU)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("invalid request device uuid %s", val.UUID)
			}

			deviceInfo, ok := p.getHAMiHCU(val.UUID + "-fake-0")
			if !ok {
				log.Errorf("Device serial number %s not found in mapper", devSerialNumber)
				util.PodAllocationFailed(nodename, current, NodeLockHCU)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("device serial number %s not found in mapper", devSerialNumber)
			}

			cardAndRenderNames, err := util.GetCardAndRender(deviceInfo.PciBusNumber)
			if err != nil {
				log.Errorf("Device Card and Render Found Error by BUS id %s, Error:%v", deviceInfo.PciBusNumber, err)
				util.PodAllocationFailed(nodename, current, NodeLockHCU)
				return &pluginapi.AllocateResponse{}, fmt.Errorf("device Card and Render Found Error by BUS id %s, Error:%v", deviceInfo.PciBusNumber, err)
			}
			for _, devPath := range cardAndRenderNames {
				devCardPath := "/dev/dri/" + devPath
				devCard := new(pluginapi.DeviceSpec)
				devCard.HostPath = devCardPath
				devCard.ContainerPath = devCardPath
				devCard.Permissions = "rw"
				car.Devices = append(car.Devices, devCard)
			}

			physicalDeviceInfo := deviceInfo

			if val.Usedcores == 100 && val.Usedmem == int32(physicalDeviceInfo.MemoryTotal/1024/1024) {
				_, _ = p.CreateMarkFile(current, &current.Spec.Containers[ctrIndex], physicalDeviceInfo.DvInd, -1)
			}
		}

		//Create virtual HCU and Make Resource Mark file
		physicalDeviceInfo, ok := p.getHAMiHCU(devreq[0].UUID + "-fake-0")
		if !ok {
			log.Errorf("Device %s not found in mapper", devreq[0].UUID)
			util.PodAllocationFailed(nodename, current, NodeLockHCU)
			return &pluginapi.AllocateResponse{}, fmt.Errorf("device %s not found in mapper", devreq[0].UUID)
		}
		log.V(3).Infoln("devreqs=", len(devreq), "usedmem=", devreq[0].Usedmem, ":", physicalDeviceInfo.MemoryTotal/1024/1024)
		if len(devreq) < 2 && devreq[0].Usedmem < int32(physicalDeviceInfo.MemoryTotal/1024/1024) {
			actualCores := int(math.Ceil(float64(devreq[0].Usedcores) * float64(physicalDeviceInfo.ComputeUnit) / 100.0))
			if actualCores < 1 {
				actualCores = 1
			}
			vIdx, err := dcgm.CreateVDevices(physicalDeviceInfo.DvInd, 1, []int{actualCores}, []int{int(devreq[0].Usedmem)})
			if err != nil {
				util.PodAllocationFailed(nodename, current, NodeLockHCU)
				return &responses, err
			}
			markFile, err := p.CreateMarkFile(current, &current.Spec.Containers[ctrIndex], physicalDeviceInfo.DvInd, vIdx[0])
			if err != nil {
				log.Errorf("Create mark file for vHCU %d failed: %v", vIdx[0], err)
			}
			if len(markFile) > 0 {
				car.Mounts = append(car.Mounts, &pluginapi.Mount{
					ContainerPath: VIRTUAL_HCU_CONF_DIR + fmt.Sprintf("docker/vdev%d.conf", vIdx[0]),
					HostPath:      VIRTUAL_HCU_CONF_DIR + fmt.Sprintf("vdev%d.conf", vIdx[0]),
					ReadOnly:      true,
				})
			}
		}
		responses.ContainerResponses = append(responses.ContainerResponses, &car)
	}
	log.V(3).Infoln("response=", responses)
	_ = util.DeleteContainerIndexAnnotations(current)
	util.PodAllocationTrySuccess(nodename, util.HygonHCUDevice, NodeLockHCU, current)
	return &responses, nil
}

func (p *DevicePlugin) AllocateCores(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}

	car := pluginapi.ContainerAllocateResponse{}
	addCommonRWMDevices(&car)

	responses.ContainerResponses = append(responses.ContainerResponses, &car)

	log.V(3).Infoln("response=", responses)
	return &responses, nil
}

// HCULister serves as an interface between implementation and Manager machinery.
type HCULister struct {
	ResUpdateChan            chan dpm.PluginNameList
	Heartbeat                chan bool
	Signal                   chan os.Signal
	ResourceRegisterStrategy chan string
}

// GetResourceNamespace must return namespace (vendor ID) of implemented Lister.
func (l *HCULister) GetResourceNamespace() string {
	return util.ResourceNamespace
}

// Discover notifies manager with a list of currently available resources in its namespace.
func (l *HCULister) Discover(pluginListCh chan dpm.PluginNameList) {
	for {
		select {
		case newResourcesList := <-l.ResUpdateChan: // New resources found
			pluginListCh <- newResourcesList
		case <-pluginListCh: // Stop message received
			return
		}
	}
}

// NewPlugin instantiates a plugin implementation.
func (l *HCULister) NewPlugin(resourceLastName string) dpm.PluginInterface {
	options := []DevicePluginOption{
		WithHeartbeat(l.Heartbeat),
		WithResource(resourceLastName),
	}
	return NewDevicePlugin(options...)
}
