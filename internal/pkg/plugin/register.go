/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package plugin

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/api"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util/client"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	kubeletdevicepluginv1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const VIRTUAL_HCU_CONF_DIR = "/etc/vdev/"

// DefaultDeviceSplitCount is the fallback max number of vHCUs a single physical
// HCU can be split into in HAMi mode.
const DefaultDeviceSplitCount = 4

var TopologyRegister bool

// DeviceSplitCount is the max number of vHCUs per physical HCU in HAMi mode,
// configurable via the --device-split-count flag / DEVICE_SPLIT_COUNT env.
var DeviceSplitCount = DefaultDeviceSplitCount

type DevListFunc func() []*kubeletdevicepluginv1beta1.Device

func (p *DevicePlugin) apiDevices() (*[]*api.DeviceInfo, error) {
	res := []*api.DeviceInfo{}

	hcus := p.getHCUs()
	log.V(3).Infof("Found physical %d HCU", len(hcus))

	healthList, healthErr := dcgm.HCUHealthCheck()
	if healthErr != nil {
		log.Errorf("HCU health check failed, reporting devices as healthy: %v", healthErr)
	}
	for _, val := range hcus {
		devTypeName := util.ResolveDevTypeName(val)
		if val.MemoryTotal <= 0 || val.ComputeUnit <= 0 || devTypeName == "" {
			log.Warningf("Skip incomplete HCU %s: mem=%v cu=%v type=%q",
				val.DeviceId, val.MemoryTotal, val.ComputeUnit, devTypeName)
			continue
		}

		// A failed health query must not mark every device unhealthy, or the
		// scheduler would drain the whole node on a transient DCGM error.
		health := true
		if healthErr == nil {
			health = util.SimpleHealthCheck(healthList, uint(val.DvInd))
		}
		numas, err := dcgm.ShowNumaTopology([]int{val.DvInd})
		log.V(3).Infof("Watching HCU with Index: %s NUMA Node: %+v", val.DeviceId, numas)
		if err != nil || len(numas) == 0 {
			log.Errorf("Get HCU numa info error: %v", err)
			continue
		}
		res = append(res, &api.DeviceInfo{
			Index:   val.DvInd,
			Id:      "HCU-" + val.DeviceId,
			Count:   int32(DeviceSplitCount),
			Devmem:  int32(val.MemoryTotal / 1024 / 1024),
			Devcore: 100,
			Numa:    numas[0].NumaNode,
			Type:    "HCU-" + devTypeName,
			Health:  health,
		})
	}
	return &res, nil
}

func (p *DevicePlugin) apiDevicesRemain() (*[]*api.DeviceInfo, error) {
	res := []*api.DeviceInfo{}

	hcus := p.getHCUs()
	log.V(3).Infof("Found physical %d HCU", len(hcus))

	healthList, healthErr := dcgm.HCUHealthCheck()
	if healthErr != nil {
		log.Errorf("HCU health check failed, reporting devices as healthy: %v", healthErr)
	}
	for _, val := range hcus {
		devTypeName := util.ResolveDevTypeName(val)
		if val.MemoryTotal <= 0 || val.ComputeUnit <= 0 || devTypeName == "" {
			log.Warningf("Skip incomplete HCU %s: mem=%v cu=%v type=%q",
				val.DeviceId, val.MemoryTotal, val.ComputeUnit, devTypeName)
			continue
		}

		health := true
		if healthErr == nil {
			health = util.SimpleHealthCheck(healthList, uint(val.DvInd))
		}
		numas, err := dcgm.ShowNumaTopology([]int{val.DvInd})
		log.V(3).Infof("Watching HCU with Index: %s NUMA Node: %+v", val.DeviceId, numas)
		if err != nil || len(numas) == 0 {
			log.Errorf("Get HCU numa info error: %v", err)
			continue
		}

		remainCU, remainMem, remainErr := dcgm.DeviceRemainingInfo(val.DvInd)
		vdeviceCount, _, _ := dcgm.VDeviceByDvInd(val.DvInd)

		devmem := int32(val.MemoryTotal / 1024 / 1024)
		devcore := int32(100)
		if remainErr != nil {
			log.Warningf("DeviceRemainingInfo(%d) failed: %v, fallback to full capacity", val.DvInd, remainErr)
		} else {
			devmem = int32(remainMem / 1024 / 1024)
			devcore = int32((float64(remainCU) / val.ComputeUnit) * 100)
		}

		res = append(res, &api.DeviceInfo{
			Index:   val.DvInd,
			Id:      "HCU-" + val.DeviceId,
			Count:   int32(DeviceSplitCount - vdeviceCount),
			Devmem:  devmem,
			Devcore: devcore,
			Numa:    numas[0].NumaNode,
			Type:    "HCU-" + devTypeName,
			Health:  health,
		})
	}
	return &res, nil
}

func (p *DevicePlugin) RegistrInAnnotation() error {
	// Reinit DCGM when hot-plug / DMI incomplete, then refresh device list.
	util.ReconcileDCGMDevices()
	p.setHCUs(util.GetAllPhysicalDevices())

	devices, err := p.apiDevices()
	if err != nil {
		log.Errorln("get api devices error", err.Error())
		return err
	}

	annos := make(map[string]string)
	if len(util.NodeName) == 0 {
		util.NodeName = os.Getenv(util.NodeNameEnvName)
	}
	node, err := util.GetNode(util.NodeName)
	if err != nil {
		log.Errorln("get node error", err.Error())
		return err
	}
	encodeddevices := util.EncodeNodeDevices(*devices)
	annos[util.HandshakeAnnosString] = "Reported " + time.Now().String()
	annos[util.RegisterAnnos] = encodeddevices
	log.V(3).Infof("Reporting devices %s in %v", encodeddevices, time.Now())

	remainDevices, err := p.apiDevicesRemain()
	if err != nil {
		log.Errorln("get remaining api devices error", err.Error())
		return err
	}
	encodedRemainDevices := util.EncodeNodeDevices(*remainDevices)
	annos[util.HygonRegisterAnnos] = encodedRemainDevices

	err = util.PatchNodeAnnotations(node, annos)

	if err != nil {
		log.Errorln("patch node error", err.Error())
	}
	return err
}

func (p *DevicePlugin) WatchAndRegister() {
	log.Info("into WatchAndRegister")
	for {
		err := p.RegistrInAnnotation()
		_ = p.RefreshContainerDevices()
		if TopologyRegister {
			_ = p.UpdateTopologyConfigMap()
		}
		if err != nil {
			log.Errorf("register error, %v", err)
			time.Sleep(time.Second * 5)
		} else {
			time.Sleep(time.Second * 30)
		}
	}
}

func (p *DevicePlugin) RefreshContainerDevices() error {
	files, err := os.ReadDir(VIRTUAL_HCU_CONF_DIR + "dynamic/")
	if err != nil {
		return err
	}

	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", util.NodeName).String()
	options := metav1.ListOptions{}
	options.FieldSelector = fieldSelector
	pods, err := client.GetClient().CoreV1().Pods("").List(context.Background(), options)
	if err != nil {
		return err
	}

	for _, f := range files {
		found := false
		for _, val := range pods.Items {
			if strings.Contains(f.Name(), string(val.UID)) && !(val.Status.Phase == corev1.PodSucceeded || val.Status.Phase == corev1.PodFailed) {
				found = true
			}
		}

		if !found {
			// Mark file name layout is "<podUID>_<container>_<devIdx>_<vdevIdx>".
			// A malformed name must be skipped: Atoi failure would yield index 0
			// and destroy a live vHCU belonging to another container.
			tmpstr := strings.Split(f.Name(), "_")
			if len(tmpstr) < 4 {
				log.Warningf("Skip unexpected mark file %s", f.Name())
				continue
			}
			vdidx, convErr := strconv.Atoi(tmpstr[3])
			if convErr != nil {
				log.Warningf("Skip mark file %s: bad vdev index %q: %v", f.Name(), tmpstr[3], convErr)
				continue
			}

			_ = os.RemoveAll(VIRTUAL_HCU_CONF_DIR + "dynamic/" + f.Name())

			var err error
			if vdidx > -1 {
				for try := 0; try < 5; try++ {
					err = dcgm.StopVDevice(vdidx)
					if err == nil || try == 4 {
						log.V(2).Infof("Stop vHCU %d sucessfully", vdidx)
						for retry := 0; retry < 5; retry++ {
							err = dcgm.DestroySingleVDevice(vdidx)
							if err == nil {
								log.V(2).Infof("Delete vHCU %d sucessfully", vdidx)
								break
							}
						}
						break
					}
					log.Errorf("Stop vHCU %d error: %v. Try Again!", vdidx, err)
				}

				_ = os.Remove(fmt.Sprintf(VIRTUAL_HCU_CONF_DIR+"vdev%d.conf", vdidx))
			}

		}
		log.V(3).Infof("Refresh container file %s.", f.Name())
	}

	for _, val := range pods.Items {
		errorPod := true
		for _, file := range files {
			if strings.Contains(file.Name(), string(val.UID)) {
				errorPod = false
			}
		}
		if util.RequestsVirtualHCU(&val) && errorPod && (val.Status.Phase == corev1.PodRunning || val.Status.Phase == corev1.PodFailed) {
			_ = util.DeletePod(context.Background(), &val)
		}
	}

	return nil
}

func (p *DevicePlugin) informerPodHandler() {
	nodeName := util.NodeName
	stopCh := make(chan struct{})
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName).String()

	podInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return client.GetClient().CoreV1().Pods("").List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return client.GetClient().CoreV1().Pods("").Watch(context.TODO(), options)
			},
		},
		&corev1.Pod{},
		10*time.Minute,
		cache.Indexers{},
	)

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				log.Warningf("[ADD] unexpected object type %T", obj)
				return
			}
			log.V(3).Infof("[ADD] Pod %s/%s\n", pod.Name, pod.Status.Phase)
			if util.RequestsVirtualHCU(pod) {
				log.V(3).Infof("[ADD] Pod %s/%s\n", pod.Namespace, pod.Name)
				if err := p.RegistrInAnnotation(); err != nil {
					log.Errorf("[ADD] register in annotation failed: %v", err)
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod, ok := oldObj.(*corev1.Pod)
			if !ok {
				log.Warningf("[UPDATE] unexpected old object type %T", oldObj)
				return
			}
			newPod, ok := newObj.(*corev1.Pod)
			if !ok {
				log.Warningf("[UPDATE] unexpected new object type %T", newObj)
				return
			}
			if oldPod.Status.Phase == corev1.PodRunning &&
				(newPod.Status.Phase == corev1.PodSucceeded || newPod.Status.Phase == corev1.PodFailed) {

				log.V(3).Infof("[COMPLETE] Pod %s/%s changed from %s -> %s\n",
					newPod.Namespace, newPod.Name, oldPod.Status.Phase, newPod.Status.Phase)
				if util.RequestsVirtualHCU(oldPod) {
					if err := p.RefreshContainerDevices(); err != nil {
						log.Errorf("[COMPLETE] refresh container devices failed: %v", err)
					}
					if err := p.RegistrInAnnotation(); err != nil {
						log.Errorf("[COMPLETE] register in annotation failed: %v", err)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// On watch re-sync the informer may deliver a tombstone instead of a Pod.
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				tombstone, isTombstone := obj.(cache.DeletedFinalStateUnknown)
				if !isTombstone {
					log.Warningf("[DELETE] unexpected object type %T", obj)
					return
				}
				pod, ok = tombstone.Obj.(*corev1.Pod)
				if !ok {
					log.Warningf("[DELETE] tombstone contains unexpected object type %T", tombstone.Obj)
					return
				}
			}
			log.V(3).Infof("[DELETE] Pod %s/%s\n", pod.Namespace, pod.Name)
			if util.RequestsVirtualHCU(pod) {
				if err := p.RefreshContainerDevices(); err != nil {
					log.Errorf("[DELETE] refresh container devices failed: %v", err)
				}
				if err := p.RegistrInAnnotation(); err != nil {
					log.Errorf("[DELETE] register in annotation failed: %v", err)
				}
			}
		},
	})

	log.V(2).Infof("Starting pod watcher on node %s...\n", nodeName)
	go podInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, podInformer.HasSynced)

	<-stopCh
}

func (p *DevicePlugin) CreateMarkFile(current *corev1.Pod, ctr *corev1.Container, devidx int, vdevidx int) (string, error) {
	markFile := string(current.UID) + "_" + ctr.Name + "_" + fmt.Sprint(devidx) + "_" + fmt.Sprint(vdevidx)
	cacheFileHostDirectory := fmt.Sprintf(VIRTUAL_HCU_CONF_DIR+"%s", "dynamic")
	_, err := os.Stat(cacheFileHostDirectory)
	if os.IsNotExist(err) {
		err := os.MkdirAll(cacheFileHostDirectory, 0777)
		if err != nil {
			return "", err
		}
		err = os.Chmod(cacheFileHostDirectory, 0777)
		if err != nil {
			return "", err
		}
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", cacheFileHostDirectory, markFile), []byte(time.Now().Format(time.DateTime)), os.ModePerm)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", cacheFileHostDirectory, markFile), nil
}

func (p *DevicePlugin) UpdateTopologyConfigMap() error {
	topology, err := dcgm.DiscoverInterconnectTopology()
	if err != nil {
		log.Errorf("Get HCU topology error: %v", err)
		return err
	}

	log.V(3).Infof("HCU topology info: %v", topology)
	data, err := json.Marshal(topology)
	if err != nil {
		log.Errorf("Marshal HCU topology json error: %v", err)
		return err
	}

	patch := map[string]interface{}{
		"data": map[string]string{
			util.NodeName: string(data),
		},
	}
	patchBytes, _ := json.Marshal(patch)

	_, err = client.GetClient().CoreV1().
		ConfigMaps("kube-system").
		Patch(context.Background(), "hcu-topology-info", types.MergePatchType, patchBytes, metav1.PatchOptions{})

	if err == nil {
		return nil
	}

	// if not exists, create it
	if apierrors.IsNotFound(err) {

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.DeviceTopologyConfigMapName,
				Namespace: util.DeviceTopologyConfigMapNamespace,
			},
			Data: map[string]string{
				util.NodeName: string(data),
			},
		}

		_, createErr := client.GetClient().CoreV1().
			ConfigMaps(util.DeviceTopologyConfigMapNamespace).
			Create(context.Background(), cm, metav1.CreateOptions{})

		if createErr == nil {
			return nil
		}

		// if it is creating by other node, try patch again
		if apierrors.IsAlreadyExists(createErr) {
			_, retryErr := client.GetClient().CoreV1().
				ConfigMaps(util.DeviceTopologyConfigMapNamespace).
				Patch(context.Background(), util.DeviceTopologyConfigMapName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			return retryErr
		}

		return createErr
	}

	return err
}
