/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package plugin

import (
	"strconv"
	"strings"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// markFileInfo is the parsed form of
// "<podUID>_<containerName>_<devIdx>_<vdevIdx>".
// containerName may itself contain underscores.
type markFileInfo struct {
	podUID        string
	containerName string
	devIdx        int
	vdevIdx       int
}

func parseMarkFileName(name string) (markFileInfo, bool) {
	parts := strings.Split(name, "_")
	if len(parts) < 4 {
		return markFileInfo{}, false
	}
	vdevIdx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return markFileInfo{}, false
	}
	devIdx, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil {
		return markFileInfo{}, false
	}
	return markFileInfo{
		podUID:        parts[0],
		containerName: strings.Join(parts[1:len(parts)-2], "_"),
		devIdx:        devIdx,
		vdevIdx:       vdevIdx,
	}, true
}

func findContainerStatus(statuses []corev1.ContainerStatus, name string) *corev1.ContainerStatus {
	for i := range statuses {
		if statuses[i].Name == name {
			return &statuses[i]
		}
	}
	return nil
}

func initContainerByName(pod *corev1.Pod, name string) *corev1.Container {
	for i := range pod.Spec.InitContainers {
		if pod.Spec.InitContainers[i].Name == name {
			return &pod.Spec.InitContainers[i]
		}
	}
	return nil
}

func isRestartableInitContainer(c *corev1.Container) bool {
	if c == nil || c.RestartPolicy == nil {
		return false
	}
	return *c.RestartPolicy == corev1.ContainerRestartPolicyAlways
}

// markFileStillNeeded reports whether the vHCU / mark file for containerName
// must be kept for pod.
//
// Classic init containers run once: reclaim as soon as they are Terminated,
// even while the Pod is still Running. Restartable sidecar inits and app
// containers keep their allocation until the Pod reaches a terminal phase
// (kubelet reuses AllocateResponse across container restarts).
func markFileStillNeeded(pod *corev1.Pod, containerName string) bool {
	if pod == nil {
		return false
	}
	switch pod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodFailed:
		return false
	}

	if init := initContainerByName(pod, containerName); init != nil {
		if isRestartableInitContainer(init) {
			return true
		}
		st := findContainerStatus(pod.Status.InitContainerStatuses, containerName)
		if st != nil && st.State.Terminated != nil {
			return false
		}
		return true
	}

	// App containers: keep while the pod is not terminal.
	return true
}

func containerRequestsVirtualHCU(c corev1.Container) bool {
	mem := util.ResourceNameMem()
	for resourceName, quantity := range c.Resources.Limits {
		name := string(resourceName)
		if strings.Contains(name, "hygon.com") && strings.Contains(name, mem) && quantity.Value() > 0 {
			return true
		}
	}
	return false
}

// podMissingRequiredMark reports whether any container that still needs a
// virtual HCU has no corresponding mark file. Used to detect Allocate
// failures without false-positive deleting pods whose only vHCU consumers
// were finished init containers.
func podMissingRequiredMark(pod *corev1.Pod, markNames []string) bool {
	if pod == nil {
		return false
	}
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodFailed {
		return false
	}

	for _, c := range append(append([]corev1.Container{}, pod.Spec.InitContainers...), pod.Spec.Containers...) {
		if !containerRequestsVirtualHCU(c) {
			continue
		}
		if !markFileStillNeeded(pod, c.Name) {
			continue
		}
		prefix := string(pod.UID) + "_" + c.Name + "_"
		found := false
		for _, name := range markNames {
			if strings.HasPrefix(name, prefix) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}

func initContainerTerminatedSince(oldPod, newPod *corev1.Pod) bool {
	if oldPod == nil || newPod == nil {
		return false
	}
	for _, st := range newPod.Status.InitContainerStatuses {
		if st.State.Terminated == nil {
			continue
		}
		if init := initContainerByName(newPod, st.Name); isRestartableInitContainer(init) {
			continue
		}
		oldSt := findContainerStatus(oldPod.Status.InitContainerStatuses, st.Name)
		if oldSt == nil || oldSt.State.Terminated == nil {
			return true
		}
	}
	return false
}
