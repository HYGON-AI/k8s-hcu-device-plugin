/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

// Kubernetes (k8s) device plugin to enable registration of HCU to a container cluster
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/HYGON-AI/hcu-dcgm/v3/pkg/dcgm"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/plugin"
	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/util"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/urfave/cli/v2"
)

var pulse int
var resourceRegisterStrategy, policy string
var version = ""
var resource_mutiple bool
var logLevel int
var logSeverity string

func startDevicePlugin(resources []string) {
	l := plugin.HCULister{
		ResUpdateChan: make(chan dpm.PluginNameList),
		Heartbeat:     make(chan bool),
	}
	manager := dpm.NewManager(&l)

	if pulse > 0 {
		go func() {
			log.V(2).Infof("Heart beating every %d seconds", pulse)
			for {
				time.Sleep(time.Second * time.Duration(pulse))
				l.Heartbeat <- true
			}
		}()
	}

	go func() {
		// /sys/class/kfd only exists if HCU kernel/driver is installed
		var path = "/sys/class/kfd"
		if _, err := os.Stat(path); err == nil {
			if err != nil {
				log.Errorf("Error occured: %v", err)
				os.Exit(1)
			}
			if len(resources) > 0 {
				l.ResUpdateChan <- resources
			}
		}
	}()
	manager.Run()
}

// startHCUMemReporter reports total HCU memory as a node extended resource.
// hcumem is counted in MiB, so an 8-card node would need hundreds of thousands of
// discrete devices to express via ListAndWatch, which exceeds what the device plugin
// API can carry. Patching node status carries the same number as a single quantity.
func startHCUMemReporter(resourceName string) {
	for {
		var totalMiB int64
		for _, device := range util.GetAllPhysicalDevices() {
			totalMiB += int64(device.MemoryTotal) / 1024 / 1024
		}

		err := util.PatchNodeResources(util.NodeName, map[string]int64{resourceName: totalMiB})
		if err != nil {
			log.Errorf("Report %s=%d failed: %v", resourceName, totalMiB, err)
			time.Sleep(time.Second * 5)
			continue
		}

		log.V(3).Infof("Reported %s=%d MiB", resourceName, totalMiB)
		time.Sleep(time.Second * 30)
	}
}

func start() {

	log.V(2).Infof("Hygon HCU Device Plugin start ...")

	log.V(2).Infof("Init HCU DCGM: %v \n", dcgm.Init())
	defer func() {
		err := dcgm.ShutDown()
		if err != nil {
			log.Errorf("Hygon HCU Device Plugin Shutdown Error: %v ", err)
			return
		}
	}()

	util.StartDCGMDeviceReconcileLoop()

	resourceNamePrefix := util.GetResourceNamePrefix(policy)

	if resourceRegisterStrategy == "hcu" {
		// Run hygon.com/hcu Device Plugin Only
		go startDevicePlugin([]string{resourceNamePrefix})

	} else if resourceRegisterStrategy == "mig" {
		// Run hygon.com/hcu-mig-* Device Plugin Only
		allMigHCUs := util.GetAllMigHCUs()
		for resourceName := range allMigHCUs {
			go startDevicePlugin([]string{resourceNamePrefix + "-" + resourceName})
		}

	} else if resourceRegisterStrategy == "vhcu" {
		// Run hygon.com/hcu-share-* Device Plugin Only
		allVirtualHCUs := util.GetAllVirtualHCUs()
		for resourceName := range allVirtualHCUs {
			go startDevicePlugin([]string{resourceNamePrefix + "-" + resourceName})
		}

	} else if resourceRegisterStrategy == "mixed" {
		// Run hygon.com/hcu Device Plugin
		go startDevicePlugin([]string{resourceNamePrefix})

		// Run hygon.com/hcu-share-* Device Plugin
		allVirtualHCUs := util.GetAllVirtualHCUs()
		for resourceName := range allVirtualHCUs {
			go startDevicePlugin([]string{resourceNamePrefix + "-" + resourceName})
		}

	} else if resourceRegisterStrategy == "hami" {
		// Run hygon.com/hcunum Device Plugin
		go startDevicePlugin([]string{util.GetHAMiResourceName(policy)})

		if resource_mutiple {
			// Run hygon.com/hcucores Device Plugin
			go startDevicePlugin([]string{strings.ReplaceAll(util.GetHAMiResourceName(policy), "hcunum", "hcucores")})
			// Report hygon.com/hcumem via node status patch instead of a Device Plugin,
			// because the MiB-denominated total exceeds the device plugin list limit.
			go startHCUMemReporter(util.ResourceNamespace + "/" + strings.ReplaceAll(util.GetHAMiResourceName(policy), "hcunum", "hcumem"))
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		select {
		case <-sig:
			log.Info("Received signal, exiting")
			_ = dcgm.ShutDown()
			log.Flush()
			os.Exit(1)
		}
	}

}

func main() {
	c := cli.NewApp()
	c.Name = "Hygon HCU Device Plugin"
	c.Usage = "Hygon HCU device plugin for Kubernetes"
	c.Version = version

	c.Before = func(c *cli.Context) error {
		if err := log.Init(logLevel, logSeverity); err != nil {
			return err
		}

		if plugin.DeviceSplitCount < 1 {
			return fmt.Errorf("invalid device-split-count %d, must be greater than 0", plugin.DeviceSplitCount)
		}

		_ = os.Setenv(util.RESOURCE_REGISTER_STRATEGY, resourceRegisterStrategy)
		_ = os.Setenv("POLICY", policy)
		return nil
	}

	c.Action = func(ctx *cli.Context) error {
		start()
		return nil
	}

	c.Flags = []cli.Flag{
		&cli.IntFlag{
			Name:        "pulse",
			Value:       30,
			Usage:       "the device health check intervals",
			Destination: &pulse,
			EnvVars:     []string{"PULSE"},
		},
		&cli.StringFlag{
			Name:        "node-name",
			Usage:       "nodeName in k8s cluster",
			Destination: &util.NodeName,
			EnvVars:     []string{"NODE_NAME"},
		},
		&cli.StringFlag{
			Name:        "strategy",
			Value:       "mixed",
			Destination: &resourceRegisterStrategy,
			Usage:       "the desired strategy for exposing HCU/vHCU/MIG devices on HCUs that support it:\n\t\t[hcu | vhcu | mig | mixed | hami], default value is mixed",
			EnvVars:     []string{"RESOURCE_REGISTER_STRATEGY"},
		},
		&cli.StringFlag{
			Name:        "policy",
			Usage:       "resource name registration policy :\n\t\t[0 | 1 | 2]",
			Value:       "0",
			Destination: &policy,
			EnvVars:     []string{"POLICY"},
		},
		&cli.BoolFlag{
			Name:        "topology-register",
			Usage:       "HCU topology detail register or not:\n\t\t[false | true]",
			Value:       true,
			Destination: &plugin.TopologyRegister,
			EnvVars:     []string{"TOPOLOGY_REGISTER"},
		},
		&cli.IntFlag{
			Name:        "device-split-count",
			Usage:       "the max number of vHCUs a single physical HCU can be split into in hami strategy",
			Value:       plugin.DefaultDeviceSplitCount,
			Destination: &plugin.DeviceSplitCount,
			EnvVars:     []string{"DEVICE_SPLIT_COUNT"},
		},
		&cli.IntFlag{
			Name:        "log-level",
			Usage:       "verbosity level for detailed logs:\n\t\t(0-10)",
			Value:       2,
			Destination: &logLevel,
			EnvVars:     []string{"LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:        "log-severity",
			Usage:       "minimum severity written to stderr:\n\t\t[INFO | WARNING | ERROR]",
			Value:       "INFO",
			Destination: &logSeverity,
			EnvVars:     []string{"LOG_SEVERITY"},
		},
		&cli.BoolFlag{
			Name:        "resource-multiple",
			Usage:       "hygon.com/hcucores and hygon.com/hcumem resources register or not:\n\t\t[false | true]",
			Value:       false,
			Destination: &resource_mutiple,
			EnvVars:     []string{"RESOURCE_MULTIPLE"},
		},
	}

	err := c.Run(os.Args)
	if err != nil {
		log.Error(err)
		log.Flush()
		os.Exit(1)
	}
	log.Flush()
}
