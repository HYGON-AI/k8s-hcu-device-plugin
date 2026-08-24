<!--
SPDX-License-Identifier: Apache-2.0
Copyright 2026 Hygon Information Technology Co., Ltd.
-->

# Hygon HCU Device Plugin

*[中文](README.md) | English*

The Hygon HCU Device Plugin is a Kubernetes device plugin deployed as a DaemonSet on every HCU node. It implements the [Kubernetes Device Plugin API](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/) to register Hygon HCU resources with the cluster so that Pods can request them.

Current version: **v3.0.0**

## Table of Contents

- [Features](#features)
- [Mode Comparison](#mode-comparison)
- [Architecture Overview](#architecture-overview)
- [Code Layout](#code-layout)
- [Prerequisites](#prerequisites)
- [Resource Registration Strategies](#resource-registration-strategies)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Usage Examples](#usage-examples)
- [Building](#building)
- [Verification](#verification)
- [License](#license)
- [Third-Party Notices](#third-party-notices)

## Features

- **Physical HCU**: discovery, registration, health checking, and whole-card allocation
- **Pre-partitioned vHCU**: registration, health checking, and scheduling of virtual HCU instances carved out in advance by an administrator
- **Dynamically partitioned vHCU (HAMi mode)**: vHCUs are created on demand at Pod startup according to the requested compute/memory, and reclaimed automatically when the Pod exits
- **Pre-partitioned MIG HCU**: registration, health checking, and scheduling of MIG instances
- **NUMA topology awareness**: reports each device's NUMA node to the kubelet
- **Topology registration** (HAMi mode): writes the node's HCU interconnect topology into the `kube-system/hcu-topology-info` ConfigMap

## Mode Comparison

| Mode | Strategy | Typical Resource | Pre-partitioning Required | Scheduler Required |
|------|----------|------------------|---------------------------|--------------------|
| Whole physical card | `hcu` / `mixed` | `hygon.com/hcu` | No | No |
| Pre-partitioned vHCU | `vhcu` / `mixed` | `hygon.com/hcu-share-4c-16g` | Yes (`hy-smi virtual`) | No |
| Dynamically partitioned vHCU | `hami` | `hygon.com/hcunum` + `hcucores` + `hcumem` | No | Yes ([k8s-hcu-scheduler](../k8s-hcu-scheduler)) |
| MIG | `mig` | `hygon.com/hcu-mig-*` | Yes (`hy-smi mig`) | No |

## Architecture Overview

### Physical / Pre-partitioned Modes

```
┌───────────────────────────────────────────────────────────────┐
│                       Kubernetes Node                         │
│  ┌──────────────┐    ┌─────────────────────────────────────┐  │
│  │   Kubelet    │◄──►│ HCU Device Plugin (DaemonSet)       │  │
│  └──────────────┘    │  - ListAndWatch: report device list │  │
│                      │  - Allocate: attach devices to ctnr │  │
│                      │  - Health check (DCGM)              │  │
│                      └──────────────┬──────────────────────┘  │
│                                     │ DCGM                    │
│                      ┌──────────────▼──────────────────────┐  │
│                      │  /dev/dri  /dev/kfd  /dev/mkfd      │  │
│                      │  /etc/vdev  /etc/dmi_mig_config     │  │
│                      └─────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### Dynamically Partitioned vHCU (HAMi Mode)

```
  Pod (hcunum / hcucores / hcumem)
             │
             ▼
  ┌──────────────────────┐
  │  Admission Webhook   │  Rewrites schedulerName, injects
  └──────────┬───────────┘  compute/memory annotations
             ▼
  ┌──────────────────────┐
  │  HCU Scheduler       │  Selects the node and physical card,
  └──────────┬───────────┘  writes allocation annotations
             ▼
  ┌──────────────────────┐
  │  HCU Device Plugin   │  Creates the vHCU during Allocate and
  │  (strategy=hami)     │  mounts its config; destroys it on exit
  └──────────────────────┘
```

## Code Layout

```
k8s-hcu-device-plugin/
├── cmd/
│   └── main.go                          # Entry point: CLI parsing, DCGM init, per-strategy plugin startup
├── internal/pkg/
│   ├── plugin/
│   │   ├── plugin.go                    # Device plugin core: ListAndWatch / Allocate / health checks
│   │   └── register.go                  # HAMi mode: node annotation registration, Pod informer, topology ConfigMap
│   ├── util/
│   │   ├── hcu.go                       # HCU discovery, resource naming, NUMA / health checks
│   │   ├── util.go                      # HAMi scheduling annotation codec, Pod / Node patching
│   │   ├── types.go                     # Device and container data structures, constants
│   │   └── client/
│   │       └── client.go                # Kubernetes in-cluster / kubeconfig client
│   ├── log/
│   │   └── log.go                       # Single logging entry point (klog v2 wrapper, severity gating)
│   └── api/
│       └── device_register.go           # Device information API structures
├── deployment/
│   ├── static/                          # Manifests for direct kubectl apply
│   │   ├── k8s-hcu-plugin.yaml          # mixed mode (physical HCU + pre-partitioned vHCU)
│   │   ├── k8s-hcu-plugin-mig.yaml      # MIG mode
│   │   └── k8s-hcu-plugin-hami.yaml     # HAMi dynamic partitioning mode (includes RBAC)
│   └── helm/                            # Helm charts
│       ├── k8s-hcu-plugin/              # mixed mode
│       ├── k8s-hcu-plugin-mig/          # MIG mode
│       └── k8s-hcu-plugin-hami/         # HAMi mode
├── demo/                                # Example Pods
│   ├── pytorch-hcu.yaml                 # Physical HCU
│   ├── pytorch-hcu-share.yaml           # Pre-partitioned vHCU
│   ├── pytorch-hcu-mig.yaml             # MIG HCU
│   └── pytorch-hcu-dynamic-vhcu.yaml    # Dynamically partitioned vHCU
├── build.sh                             # Compile binary, build image, export offline bundle
├── Dockerfile                           # Runtime image (relies on /opt/hyhal mounted from the node)
├── go.mod / go.sum                      # Go module dependencies
├── LICENSE
├── THIRD_PARTY_NOTICES.md               # Third-party notices (version / license / copyright / modifications)
└── README.md
```

### Core Modules

| Module | Responsibility                                                                                                                                        |
|--------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| `cmd/main.go` | Parses CLI flags such as `--strategy`, initializes DCGM, and starts an independent device plugin instance per resource type according to the strategy |
| `plugin/plugin.go` | Implements the kubelet device plugin gRPC interface: `ListAndWatch` reports the device list, `Allocate` attaches devices to containers                |
| `plugin/register.go` | In HAMi mode, watches Pod events, writes device registration annotations onto the node, and maintains the `hcu-topology-info` ConfigMap               |
| `util/hcu.go` | Discovers physical cards / vHCUs / MIG instances via [hcu-dcgm](https://github.com/HYGON-AI/hcu-dcgm) and generates `hygon.com/*` resource names      |
| `util/util.go` | Annotation encoding/decoding, node locking, and Pod allocation state management shared with [k8s-hcu-scheduler](../k8s-hcu-scheduler)                 |
| `util/client` | In-cluster Kubernetes API access (in-cluster config preferred, falls back to kubeconfig)                                                              |
| `log/log.go` | Project-wide logging entry point: wraps klog v2, implements `--log-severity` gating and bridges the settings to glog (pulled in by dpm)               |

### Runtime Dependencies

| Dependency                                                                          | Description |
|-------------------------------------------------------------------------------------|-------------|
| [hcu-dcgm](https://github.com/HYGON-AI/hcu-dcgm)                                    | Go module providing HCU device discovery, vHCU creation/destruction, and health checking |
| [Project-HAMi/HAMi](https://github.com/Project-HAMi/HAMi)                           | Node locking and scheduling coordination utilities used in HAMi mode |
| [kubevirt/device-plugin-manager](https://github.com/kubevirt/device-plugin-manager) | Device plugin lifecycle management framework |
| Node `/opt/hyhal`                                                                   | Mounted via hostPath at runtime to provide the HCU low-level libraries (not baked into the image) |

## Prerequisites

### Cluster and Nodes

| Item | Requirement |
|------|-------------|
| Kubernetes cluster | Worker nodes with HCUs installed |
| HCU driver | Driver correctly installed on the node; `/sys/class/kfd` must exist |
| hyhal | The `/opt/hyhal` directory must exist on the node and is mounted into the plugin container via hostPath |
| vHCU support | Driver version ≥ 6.2.26 (from 6.2.26 onward the virtualization command is `hy-smi virtual`) |
| Node labels | Deploying [k8s-hcu-label-node](../k8s-hcu-label-node) is recommended to label HCU nodes automatically with `hygon.com/hcu=true` and friends |
| Extra requirements for dynamic vHCU | DTK ≥ 24.04, `hy-smi` ≥ v1.6.0, plus [k8s-hcu-scheduler](../k8s-hcu-scheduler) |

### Local Build

| Item | Requirement |
|------|-------------|
| Go | ≥ 1.22 (see `go.mod`) |
| CGO | Must be enabled (`CGO_ENABLED=1`) to link against the DCGM library |
| Docker | Required to build the container image |
| HCU node | Compilation works on any machine; functional verification requires a node with the driver installed |

## Resource Registration Strategies

The `--strategy` flag (or the `RESOURCE_REGISTER_STRATEGY` environment variable) controls which resource types the plugin registers:

| Strategy | Description | Example Resources |
|----------|-------------|-------------------|
| `hcu` | Register physical HCUs only | `hygon.com/hcu` |
| `vhcu` | Register pre-partitioned vHCUs only | `hygon.com/hcu-share-4c-16g` |
| `mig` | Register MIG instances only | `hygon.com/hcu-mig-4g-31gb` |
| `mixed` (default) | Register both physical HCUs and vHCUs | `hygon.com/hcu`, `hygon.com/hcu-share-*` |
| `hami` | Dynamic sharing mode | `hygon.com/hcunum` (optionally `hcucores`, `hcumem`) |

### Resource Naming (POLICY=0, default)

- **Physical HCU**: `hygon.com/hcu`
- **Pre-partitioned vHCU**: `hygon.com/hcu-share-{CUs}c-{memoryGB}g`, e.g. `hygon.com/hcu-share-4c-16g`
- **MIG**: `hygon.com/hcu-mig-{profile}`, e.g. `hygon.com/hcu-mig-4g-31gb`
- **Dynamic vHCU**: `hygon.com/hcunum`, `hygon.com/hcucores`, `hygon.com/hcumem`

`POLICY` also supports naming by device model (`1`) or by model + memory + CU count (`2`); see the configuration section below.

## Configuration

| Flag / Environment Variable | Default | Description |
|-----------------------------|---------|-------------|
| `--strategy` / `RESOURCE_REGISTER_STRATEGY` | `mixed` | Resource registration strategy |
| `--policy` / `POLICY` | `0` | Resource naming policy: `0` default naming, `1` device model, `2` model + memory + CU |
| `--pulse` / `PULSE` | `30` | Device health check interval, in seconds |
| `--node-name` / `NODE_NAME` | - | Name of the current node (injected via the Downward API at deploy time) |
| `--topology-register` / `TOPOLOGY_REGISTER` | `true` | Whether to register the HCU topology into a ConfigMap in HAMi mode |
| `--resource-multiple` / `RESOURCE_MULTIPLE` | `false` | Whether to additionally register the `hcucores` and `hcumem` resources with the kubelet in HAMi mode |
| `--device-split-count` / `DEVICE_SPLIT_COUNT` | `4` | Maximum number of vHCUs a single physical HCU can be split into in HAMi mode (must be greater than 0) |
| `--log-level` / `LOG_LEVEL` | `2` | Verbosity level (0-10) controlling the granularity of `V(n)` logs |
| `--log-severity` / `LOG_SEVERITY` | `INFO` | Minimum severity written to stderr: `INFO` / `WARNING` / `ERROR`. Setting `WARNING` drops Info and `V(n)` logs |

> Logging is unified on klog. All output goes to stderr (never to disk) so the container runtime can collect it.

## Deployment

Manifests live under `deployment/static/` (direct `kubectl apply`) and `deployment/helm/` (Helm charts).

### 1. Install the HCU Driver

Install and load the HCU driver on the node, then confirm that `hy-smi` or `hy-smi virtual` detects the devices.

### 2. Deploy the Node Labeling Component (Recommended)

```bash
# Deploy k8s-hcu-label-node to label HCU nodes with hygon.com/hcu=true automatically
kubectl apply -f ../k8s-hcu-label-node/deployment/
```

### 3. Physical HCU + Pre-partitioned vHCU (mixed mode, default)

Suitable for ordinary HCU nodes as well as shared nodes with pre-partitioned vHCUs. Node affinity schedules the DaemonSet onto nodes labeled `hygon.com/hcu=true` that are not dedicated to MIG or HAMi.

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin.yaml
```

**Pre-partitioning vHCUs** (run on each node that should share HCUs):

1. Create vHCU instances on a physical HCU:

```bash
# Before 6.2.26
hy-virtual -d ${dev_id} \
  -create-vdevices ${num_vhcu} \
  -vdevice-compute-units $<cu_num, ...> \
  -vdevice-memory-size $<mem_size, ...>

# 6.2.26 and later
hy-smi virtual -h   # see the usage details
```

2. Label the node (optional, marks it as a shared-mode node):

```bash
kubectl label nodes <node-name> hcu-mode=share
```

3. Deploy the device plugin — mixed mode discovers and registers the vHCU resources automatically.

### 4. MIG HCU Mode

1. Create MIG instances on a physical HCU:

```bash
hy-smi mig -cgi ${gi_profile_id} -C -i ${dev_id}
# More options: hy-smi mig -h, or see the Hygon HCU Multi-Instance User Guide
```

2. Label the node:

```bash
kubectl label nodes <node-name> hcu-mode=mig
```

3. Deploy the MIG device plugin:

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin-mig.yaml
```

### 5. Dynamically Partitioned vHCU (HAMi Mode)

In this mode the device plugin calls DCGM during the `Allocate` phase of container startup to create a vHCU on the fly, and destroys it once the Pod finishes — no administrator pre-partitioning required.

**Steps:**

1. Label the node:

```bash
kubectl label nodes <node-name> hcu=on
```

2. Deploy the HAMi device plugin (includes RBAC):

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin-hami.yaml
```

3. Deploy the HCU scheduler extensions (admission webhook + custom scheduler). See the [k8s-hcu-scheduler deployment guide](../k8s-hcu-scheduler/README_EN.md#deployment):

```bash
# After installing cert-manager
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-admission-webhook-certmanager.yaml
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-admission-webhook.yaml
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-scheduler.yaml
```

4. Confirm that the node reports the resources:

```bash
kubectl describe node <node-name> | grep -E 'hcunum|hcu-register'
```

### Helm Deployment

```bash
# mixed mode (default)
helm install hcu-dp deployment/helm/k8s-hcu-plugin/

# MIG mode
helm install hcu-dp-mig deployment/helm/k8s-hcu-plugin-mig/

# HAMi mode
helm install hcu-dp-hami deployment/helm/k8s-hcu-plugin-hami/
```

> **Note**: the Helm charts and the static manifests both use image tag `v3.0.0`. If you build a different version, update `image.tag` in `values.yaml` to match.

## Usage Examples

The `demo/` directory contains example Pods for each HCU resource type; apply them directly with `kubectl apply -f demo/<file>`.

### Physical HCU (Whole Card)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hcu-pytorch-demo
spec:
  containers:
    - name: hcu-pytorch-demo
      image: harbor.sourcefind.cn:5443/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu: 1
```

```bash
kubectl apply -f demo/pytorch-hcu.yaml
```

### Pre-partitioned vHCU

An administrator must carve out the instances on the node with `hy-smi virtual` beforehand. Once the device plugin registers them under the `mixed` or `vhcu` strategy, Pods request them by profile name:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hcu-share-pytorch-demo
spec:
  containers:
    - name: hcu-share-pytorch-demo
      image: harbor.sourcefind.cn:5443/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      securityContext:
        privileged: true
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu-share-4c-16g: 1   # replace with the profile name the node actually reports
```

> Each container currently supports requesting only one pre-partitioned vHCU instance.

```bash
kubectl apply -f demo/pytorch-hcu-share.yaml
```

### MIG HCU

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hcu-mig-pytorch-demo
spec:
  containers:
    - name: hcu-mig-pytorch-demo
      image: harbor.sourcefind.cn:5443/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu-mig-4g-31gb: 1   # replace with the actual MIG profile
```

```bash
kubectl apply -f demo/pytorch-hcu-mig.yaml
```

### Dynamically Partitioned vHCU (HAMi Mode)

A Pod declares its HCU needs through three extended resources. The scheduler picks the physical card, and the device plugin creates the vHCU when the container starts:

| Resource | Meaning | Accepted Values |
|----------|---------|-----------------|
| `hygon.com/hcunum` | Number of HCU slots | Usually `1`; must be `1` whenever compute or memory is non-zero |
| `hygon.com/hcucores` | Share of compute | 1–100, where `100` means exclusive use of the whole card's compute |
| `hygon.com/hcumem` | Requested device memory | Measured in **MiB** (with the default `RESOURCE_MULTIPLE=false`) |

> `hcunum` and `hcucores` are reported through the device plugin's ListAndWatch. Because `hcumem` is
> counted in MiB, the total on an 8-card node exceeds the device count a single ListAndWatch response
> can carry, so the plugin patches it directly into the node's `status.capacity` /
> `status.allocatable` instead (which requires patch permission on `nodes/status`).

**Partial sharing example** (30% of compute, 8 GiB of memory):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hcu-dynamic-vhcu-demo
spec:
  containers:
    - name: hcu-dynamic-vhcu-demo
      image: harbor.sourcefind.cn:5443/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1
          hygon.com/hcucores: 30
          hygon.com/hcumem: 8192
```

**Whole-card example** (100% compute and all memory — no vHCU is created, the physical card is used directly):

```yaml
resources:
  limits:
    hygon.com/hcunum: 1
    hygon.com/hcucores: 100
    hygon.com/hcumem: 32768   # set to the physical card's total memory (MB)
```

```bash
kubectl apply -f demo/pytorch-hcu-dynamic-vhcu.yaml
```

> Once the admission webhook is enabled it rewrites the Pod's `schedulerName` to the HCU scheduler automatically. Without the webhook you must set `spec.schedulerName: hcu-scheduler-plugin` yourself.

**Verifying the dynamic vHCU allocation:**

```bash
# Check scheduling and binding status
kubectl get pod hcu-dynamic-vhcu-demo -o wide
kubectl describe pod hcu-dynamic-vhcu-demo | grep -E 'Annotations|hygon.com'

# Inspect the vHCU actually allocated inside the container
kubectl exec -it hcu-dynamic-vhcu-demo -- bash -c "source /opt/hygondriver/env.sh && hy-virtual -show-device-info"
```

Expected output looks roughly like:

```
Device 0:
        Actual Device: 0
        Compute units: 9
        Global memory: 8589934592 bytes
```

**After the Pod exits**, the device plugin stops and destroys the corresponding dynamic vHCU instance, releasing the physical card's resources.

More multi-container examples (Deployments, Jobs, and so on) are available under [k8s-hcu-scheduler/example](../k8s-hcu-scheduler/example/).

## Building

The version is derived automatically from `git describe --tags --dirty`, so it never has to be maintained by hand. `build.sh` fails fast if the repository has no tag.

```bash
git tag v3.0.0          # create a tag before the first build
./build.sh              # compile the binary, build the Docker image, export an offline tarball
```

Build artifacts (`${VERSION}` below is the output of `git describe`, e.g. `v3.0.0`):

| Artifact | Path / Name |
|----------|-------------|
| Binary | `k8s-device-plugin` |
| Image | `harbor.sourcefind.cn:5443/hcu/admin/base/hcu-device-plugin:${VERSION}` |
| Offline bundle | `hcu-device-plugin-${VERSION}.tar` |

Manual build:

```bash
export CGO_ENABLED=1
go mod tidy
go build -ldflags "-X 'main.version=$(git describe --tags --dirty)'" -o k8s-device-plugin cmd/main.go
```

> With uncommitted changes in the working tree, `git describe --dirty` appends a `-dirty` suffix (e.g. `v3.0.0-dirty`). That suffix is a valid Docker tag, so it can be used as the image tag directly.

> At runtime the image loads `/opt/hyhal/lib` from the node through `LD_LIBRARY_PATH`, so the HCU low-level libraries do not need to be baked into the image.

## Verification

```bash
# Device plugin Pod status (mixed mode)
kubectl get pods -n kube-system -l name=hcu-dp-ds

# HAMi mode
kubectl get pods -n kube-system -l name=hcu-dp-ds-hami

# HCU resources on the node
kubectl describe node <node-name> | grep hygon.com

# Device plugin logs
kubectl logs -n kube-system -l name=hcu-dp-ds
```

## License

Parts of this project are adapted from [HAMi](https://github.com/Project-HAMi/HAMi). Hygon's modifications and original contributions are released under the [Apache License 2.0](LICENSE).

## Third-Party Notices

Every third-party open source component pulled in via Go modules — repository, pinned version, license, local path, copyright notice, and HYGON modification status — is documented item by item in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md), kept in sync with [go.mod](go.mod).

- All dependencies use permissive licenses (Apache-2.0 / BSD-3-Clause / BSD-2-Clause / MIT / ISC). No copyleft licenses (GPL, LGPL, AGPL) are involved, so everything is compatible with this project's Apache License 2.0.
- Apart from the HAMi-derived code noted under [License](#license), every third-party component is vendored as-is from upstream with no source modifications.
- The authoritative license text for each dependency is the `LICENSE` / `NOTICE` file shipped in its distribution; run `go mod vendor` to inspect them under `vendor/`.
- When adding or upgrading a dependency (changing `go.mod`), update the corresponding entry in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
