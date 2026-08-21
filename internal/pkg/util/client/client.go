/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

package client

import (
	"os"
	"path/filepath"

	"github.com/HYGON-AI/k8s-hcu-device-plugin/internal/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	KubeClient kubernetes.Interface
)

func init() {
	var err error
	KubeClient, err = NewClient()
	if err != nil {
		panic(err)
	}
}

func GetClient() kubernetes.Interface {
	return KubeClient
}

// NewClient connects to an API server.
func NewClient() (kubernetes.Interface, error) {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Infof("BuildConfigFromFlags failed for file %s: %v using inClusterConfig", kubeConfig, err)
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Errorf("InClusterConfig Failed for err:%s", err.Error())
		}
	}
	KubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("new config error %s", err.Error())
	}
	return KubeClient, err
}
