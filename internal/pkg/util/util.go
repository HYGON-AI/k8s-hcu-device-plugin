/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2024 The HAMi Authors.
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 *
 * Modifications made by Hygon Information Technology Co., Ltd. in 2026.
 * Changes: Adapted utility functions for Hygon HCU device management and scheduling.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/api"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util/client"
	"github.com/Project-HAMi/HAMi/pkg/util/nodelock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	// OneContainerMultiDeviceSplitSymbol this is when one container use multi device, use : symbol to join device info.
	OneContainerMultiDeviceSplitSymbol = ":"

	// OnePodMultiContainerSplitSymbol this is when one pod having multi container and more than one container use device, use ; symbol to join device info.
	OnePodMultiContainerSplitSymbol = ";"

	HandshakeAnnosString = "hami.io/node-handshake-hcu"
	RegisterAnnos        = "hami.io/node-hcu-register"
	HygonRegisterAnnos   = "hygon.com/node-hcu-register"
	// ResourceNamespace is the vendor prefix for all resources exposed by this plugin.
	ResourceNamespace  = "hygon.com"
	HygonHCUDevice     = "HCU"
	HygonHCUCommonWord = "HCU"
	HCUInUse           = "hygon.com/use-hcutype"
	HCUNoUse           = "hygon.com/nouse-hcutype"
	HCUUseUUID         = "hygon.com/use-gpuuuid"
	HCUNoUseUUID       = "hygon.com/nouse-gpuuuid"

	IndexAnnos = "hygon.com/container-index"

	mode = "hami"
)

var (
	InRequestDevices map[string]string
	SupportDevices   map[string]string
	HandshakeAnnos   map[string]string
)

func init() {
	InRequestDevices = make(map[string]string)
	SupportDevices = make(map[string]string)
	HandshakeAnnos = make(map[string]string)
	InRequestDevices[HygonHCUDevice] = "hami.io/hcu-devices-to-allocate"
	SupportDevices[HygonHCUDevice] = "hami.io/hcu-devices-allocated"
	HandshakeAnnos[HygonHCUDevice] = HandshakeAnnosString
}

func GetNode(nodename string) (*corev1.Node, error) {
	n, err := client.GetClient().CoreV1().Nodes().Get(context.Background(), nodename, metav1.GetOptions{})
	return n, err
}

func GetPendingPod(node string) (*corev1.Pod, error) {
	podlist, err := client.GetClient().CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, p := range podlist.Items {
		if _, ok := p.Annotations[BindTimeAnnotations]; !ok {
			continue
		}
		if phase, ok := p.Annotations[DeviceBindPhase]; !ok {
			continue
		} else {
			if strings.Compare(phase, DeviceBindAllocating) != 0 {
				continue
			}
		}
		if n, ok := p.Annotations[AssignedNodeAnnotations]; !ok {
			continue
		} else {
			if strings.Compare(n, node) == 0 {
				return &p, nil
			}
		}
	}
	return nil, fmt.Errorf("no binding pod found on node %s", node)
}

func DecodeNodeDevices(str string) ([]*api.DeviceInfo, error) {
	if !strings.Contains(str, OneContainerMultiDeviceSplitSymbol) {
		return []*api.DeviceInfo{}, errors.New("node annotations not decode successfully")
	}
	tmp := strings.Split(str, OneContainerMultiDeviceSplitSymbol)
	var retval []*api.DeviceInfo
	for _, val := range tmp {
		if strings.Contains(val, ",") {
			items := strings.Split(val, ",")
			if len(items) == 7 {
				count, _ := strconv.Atoi(items[1])
				devmem, _ := strconv.Atoi(items[2])
				devcore, _ := strconv.Atoi(items[3])
				health, _ := strconv.ParseBool(items[6])
				numa, _ := strconv.Atoi(items[5])
				i := api.DeviceInfo{
					Id:      items[0],
					Count:   int32(count),
					Devmem:  int32(devmem),
					Devcore: int32(devcore),
					Type:    items[4],
					Numa:    numa,
					Health:  health,
				}
				retval = append(retval, &i)
			} else {
				return []*api.DeviceInfo{}, errors.New("node annotations not decode successfully")
			}
		}
	}
	return retval, nil
}

func GetPod(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	pod, err := client.GetClient().CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	return pod, err
}

func DeletePod(ctx context.Context, pod *corev1.Pod) error {

	gracePeriodSeconds := int64(0)
	err := client.GetClient().CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		Preconditions: &metav1.Preconditions{
			UID: &pod.UID,
		},
	})
	return err
}

func EncodeNodeDevices(dlist []*api.DeviceInfo) string {
	tmp := ""
	for _, val := range dlist {
		tmp += val.Id + "," + strconv.FormatInt(int64(val.Count), 10) + "," + strconv.Itoa(int(val.Devmem)) + "," +
			strconv.Itoa(int(val.Devcore)) + "," + val.Type + "," + strconv.Itoa(val.Numa) + "," +
			strconv.FormatBool(val.Health) + "," + strconv.Itoa(val.Index) + "," + mode + OneContainerMultiDeviceSplitSymbol
	}
	log.V(3).Infof("Encoded node Devices: %s", tmp)
	return tmp
}

func EncodeContainerDevices(cd ContainerDevices) string {
	tmp := ""
	for _, val := range cd {
		tmp += val.UUID + "," + val.Type + "," + strconv.Itoa(int(val.Usedmem)) + "," + strconv.Itoa(int(val.Usedcores)) + OneContainerMultiDeviceSplitSymbol
	}
	log.V(3).Infof("Encoded container Devices: %s", tmp)
	return tmp
	//return strings.Join(cd, ",")
}

func EncodeContainerDeviceType(cd ContainerDevices, t string) string {
	tmp := ""
	for _, val := range cd {
		if strings.Compare(val.Type, t) == 0 {
			tmp += val.UUID + "," + val.Type + "," + strconv.Itoa(int(val.Usedmem)) + "," + strconv.Itoa(int(val.Usedcores))
		}
		tmp += OneContainerMultiDeviceSplitSymbol
	}
	log.Infof("Encoded container Certain Device type: %s->%s", t, tmp)
	return tmp
}

func EncodePodSingleDevice(pd PodSingleDevice) string {
	res := ""
	for _, ctrdevs := range pd {
		res = res + EncodeContainerDevices(ctrdevs)
		res = res + OnePodMultiContainerSplitSymbol
	}
	log.Infof("Encoded pod single devices %s", res)
	return res
}

func EncodePodDevices(checklist map[string]string, pd PodDevices) map[string]string {
	res := map[string]string{}
	for devType, cd := range pd {
		log.V(3).Infoln("devtype=", devType)
		res[checklist[devType]] = EncodePodSingleDevice(cd)
	}
	log.V(3).Infof("Encoded pod Devices %s\n", res)
	return res
}

func DecodeContainerDevices(str string) (ContainerDevices, error) {
	if len(str) == 0 {
		return ContainerDevices{}, nil
	}
	cd := strings.Split(str, OneContainerMultiDeviceSplitSymbol)
	contdev := ContainerDevices{}
	tmpdev := ContainerDevice{}
	log.V(3).Infof("Start to decode container device %s", str)
	if len(str) == 0 {
		return ContainerDevices{}, nil
	}
	for _, val := range cd {
		if strings.Contains(val, ",") {
			//fmt.Println("cd is ", val)
			tmpstr := strings.Split(val, ",")
			if len(tmpstr) < 4 {
				return ContainerDevices{}, fmt.Errorf("pod annotation format error; information missing, please do not use nodeName field in task")
			}
			tmpdev.UUID = tmpstr[0]
			tmpdev.Type = tmpstr[1]
			devmem, _ := strconv.ParseInt(tmpstr[2], 10, 32)
			tmpdev.Usedmem = int32(devmem)
			devcores, _ := strconv.ParseInt(tmpstr[3], 10, 32)
			tmpdev.Usedcores = int32(devcores)
			contdev = append(contdev, tmpdev)
		}
	}
	log.V(3).Infof("Finished decoding container devices. Total devices: %d", len(contdev))
	return contdev, nil
}

func DecodePodDevices(checklist map[string]string, annos map[string]string) (PodDevices, error) {
	log.V(3).Infof("checklist is [%+v], annos is [%+v]", checklist, annos)
	if len(annos) == 0 {
		return PodDevices{}, nil
	}
	pd := make(PodDevices)
	for devID, devs := range checklist {
		str, ok := annos[devs]
		if !ok {
			continue
		}
		pd[devID] = make(PodSingleDevice, 0)
		for _, s := range strings.Split(str, OnePodMultiContainerSplitSymbol) {
			cd, err := DecodeContainerDevices(s)
			if err != nil {
				return PodDevices{}, nil
			}
			if len(cd) == 0 {
				continue
			}
			pd[devID] = append(pd[devID], cd)
		}
	}
	log.V(3).Info("Decoded pod annos", "poddevices", pd)
	return pd, nil
}

func GetNextDeviceRequest(dtype string, p corev1.Pod) (corev1.Container, ContainerDevices, error) {
	pdevices, err := DecodePodDevices(InRequestDevices, p.Annotations)
	if err != nil {
		return corev1.Container{}, ContainerDevices{}, err
	}
	log.V(3).Infof("pod annotation decode vaule is %+v", pdevices)
	res := ContainerDevices{}

	pd, ok := pdevices[dtype]
	if !ok {
		return corev1.Container{}, res, errors.New("device request not found")
	}
	for ctridx, ctrDevice := range pd {
		if len(ctrDevice) > 0 {
			return p.Spec.Containers[ctridx], ctrDevice, nil
		}
	}
	return corev1.Container{}, res, errors.New("device request not found")
}

func GetContainerDeviceStrArray(c ContainerDevices) []string {
	tmp := []string{}
	for _, val := range c {
		tmp = append(tmp, val.UUID)
	}
	return tmp
}

func EraseNextDeviceTypeFromAnnotation(dtype string, p corev1.Pod) error {
	pdevices, err := DecodePodDevices(InRequestDevices, p.Annotations)
	if err != nil {
		return err
	}
	res := PodSingleDevice{}
	pd, ok := pdevices[dtype]
	if !ok {
		return errors.New("erase device annotation not found")
	}
	found := false
	for _, val := range pd {
		if found {
			res = append(res, val)
		} else {
			if len(val) > 0 {
				found = true
				res = append(res, ContainerDevices{})
			} else {
				res = append(res, val)
			}
		}
	}
	log.V(3).Infoln("After erase res=", res)
	newannos := make(map[string]string)
	newannos[InRequestDevices[dtype]] = EncodePodSingleDevice(res)
	return PatchPodAnnotations(&p, newannos)
}

func PatchNodeAnnotations(node *corev1.Node, annotations map[string]string) error {
	type patchMetadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	}
	type patchPod struct {
		Metadata patchMetadata `json:"metadata"`
		//Spec     patchSpec     `json:"spec,omitempty"`
	}

	p := patchPod{}
	p.Metadata.Annotations = annotations

	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = client.GetClient().CoreV1().Nodes().
		Patch(context.Background(), node.Name, k8stypes.StrategicMergePatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		log.V(3).Infoln("annotations=", annotations)
		log.V(3).Infof("patch pod %v failed, %v", node.Name, err)
	}
	return err
}

// PatchNodeResources patches extended resources into the node status capacity and
// allocatable maps. Values are plain integer quantities keyed by full resource name
// (e.g. "hygon.com/hcumem").
func PatchNodeResources(nodeName string, resources map[string]int64) error {
	quantities := make(map[string]string, len(resources))
	for name, value := range resources {
		quantities[name] = strconv.FormatInt(value, 10)
	}

	type patchStatus struct {
		Capacity    map[string]string `json:"capacity,omitempty"`
		Allocatable map[string]string `json:"allocatable,omitempty"`
	}
	type patchNode struct {
		Status patchStatus `json:"status"`
	}

	p := patchNode{}
	p.Status.Capacity = quantities
	p.Status.Allocatable = quantities

	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = client.GetClient().CoreV1().Nodes().
		Patch(context.Background(), nodeName, k8stypes.StrategicMergePatchType, bytes, metav1.PatchOptions{}, "status")
	if err != nil {
		log.V(3).Infoln("resources=", quantities)
		log.V(3).Infof("patch node %v status failed, %v", nodeName, err)
	}
	return err
}

func PatchPodAnnotations(pod *corev1.Pod, annotations map[string]string) error {
	type patchMetadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	}
	type patchPod struct {
		Metadata patchMetadata `json:"metadata"`
		//Spec     patchSpec     `json:"spec,omitempty"`
	}

	p := patchPod{}
	p.Metadata.Annotations = annotations

	bytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	log.V(3).Infof("patch pod %s/%s annotation content is %s", pod.Namespace, pod.Name, string(bytes))
	_, err = client.GetClient().CoreV1().Pods(pod.Namespace).
		Patch(context.Background(), pod.Name, k8stypes.StrategicMergePatchType, bytes, metav1.PatchOptions{})
	if err != nil {
		log.Infof("patch pod %v failed, %v", pod.Name, err)
	}
	return err
}

func CheckHealth(devType string, n *corev1.Node) (bool, bool) {
	handshake := n.Annotations[HandshakeAnnos[devType]]
	if strings.Contains(handshake, "Requesting") {
		formertime, _ := time.Parse("2006.01.02 15:04:05", strings.Split(handshake, "_")[1])
		return time.Now().Before(formertime.Add(time.Second * 60)), false
	} else if strings.Contains(handshake, "Deleted") {
		return true, false
	} else {
		return true, true
	}
}

func MarkAnnotationsToDelete(devType string, nn string) error {
	tmppat := make(map[string]string)
	tmppat[devType] = "Deleted_" + time.Now().Format("2006.01.02 15:04:05")
	n, err := GetNode(nn)
	if err != nil {
		log.Errorln("get node failed", err.Error())
		return err
	}
	return PatchNodeAnnotations(n, tmppat)
}

func PodAllocationTrySuccess(nodeName string, devName string, lockName string, pod *corev1.Pod) {
	refreshed, err := client.GetClient().CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("get pods %s/%s error: %+v", pod.Namespace, pod.Name, err)
		return
	}
	annos := refreshed.Annotations[InRequestDevices[devName]]
	log.V(3).Infoln("TrySuccess:", annos)
	if strings.Contains(annos, HygonHCUDevice) {
		return
	}
	log.V(3).Infoln("AllDevicesAllocateSuccess releasing lock")
	PodAllocationSuccess(nodeName, pod, lockName)
}

func PodAllocationSuccess(nodeName string, pod *corev1.Pod, lockname string) {
	newannos := make(map[string]string)
	newannos[DeviceBindPhase] = DeviceBindSuccess
	err := PatchPodAnnotations(pod, newannos)
	if err != nil {
		log.Errorf("patchPodAnnotations failed:%v", err.Error())
	}
	err = nodelock.ReleaseNodeLock(nodeName, lockname, pod, false)
	if err != nil {
		log.Errorf("release lock failed:%v", err.Error())
	}
}

func PodAllocationFailed(nodeName string, pod *corev1.Pod, lockname string) {
	newannos := make(map[string]string)
	newannos[DeviceBindPhase] = DeviceBindFailed
	err := PatchPodAnnotations(pod, newannos)
	if err != nil {
		log.Errorf("patchPodAnnotations failed:%v", err.Error())
	}
	err = nodelock.ReleaseNodeLock(nodeName, lockname, pod, false)
	if err != nil {
		log.Errorf("release lock failed:%v", err.Error())
	}
}

func UpdateContainerIndexAnnotations(pod *corev1.Pod) error {
	containerIndex := ""
	reqDeviceString := pod.Annotations[SupportDevices[HygonHCUDevice]]
	allocteDevicesString := pod.Annotations[InRequestDevices[HygonHCUDevice]]

	if reqDeviceString == allocteDevicesString {
		reqDeviceArray := strings.Split(reqDeviceString, ";")
		if len(reqDeviceArray) != len(pod.Spec.Containers) {
			for i := 0; i < len(pod.Spec.Containers)-len(reqDeviceArray); i++ {
				reqDeviceString = ";" + reqDeviceString
			}
			reqDeviceArray = strings.Split(reqDeviceString, ";")
		}

		for idx, val := range reqDeviceArray {
			if len(val) > 0 {
				containerIndex += strconv.Itoa(idx) + ","
			}
		}
	}

	if len(containerIndex) > 0 {
		ann := make(map[string]string)
		ann[IndexAnnos] = containerIndex
		err := PatchPodAnnotations(pod, ann)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteContainerIndexAnnotations(pod *corev1.Pod) error {
	containerIndex := ""
	indexString := pod.Annotations[IndexAnnos]
	reqDeviceArray := ParseStringToArray(indexString, ",")
	if len(reqDeviceArray) > 0 {
		reqDeviceArray = reqDeviceArray[1:]
	}

	for _, val := range reqDeviceArray {
		containerIndex += val + ","
	}

	if len(containerIndex) > 0 {
		ann := make(map[string]string)
		ann[IndexAnnos] = containerIndex
		err := PatchPodAnnotations(pod, ann)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetCurrentContainerIndex(pod *corev1.Pod) int {
	index := -1
	indexString := pod.Annotations[IndexAnnos]
	reqDeviceArray := ParseStringToArray(indexString, ",")
	if len(reqDeviceArray) > 0 {
		idx, err := strconv.Atoi(reqDeviceArray[0])
		index = idx
		if err != nil {
			return -1
		}
	}
	return index
}

func ParseStringToArray(input string, comma string) []string {
	parts := strings.Split(input, comma)
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
