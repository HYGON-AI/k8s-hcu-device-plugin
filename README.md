<!--
SPDX-License-Identifier: Apache-2.0
Copyright 2026 Hygon Information Technology Co., Ltd.
-->

# Hygon HCU Device Plugin

Hygon HCU Device Plugin 是一个 Kubernetes 设备插件（Device Plugin），以 DaemonSet 方式部署到每个 HCU 节点，实现 [Kubernetes Device Plugin API](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)，将节点上的海光 HCU 资源注册到集群，供 Pod 申请使用。

当前版本：**v3.0.0**

## 目录

- [功能特性](#功能特性)
- [使用模式对比](#使用模式对比)
- [架构概览](#架构概览)
- [代码结构](#代码结构)
- [前置要求](#前置要求)
- [资源注册策略](#资源注册策略)
- [配置参数](#配置参数)
- [部署](#部署)
- [使用示例](#使用示例)
- [构建](#构建)
- [验证](#验证)
- [License](#license)
- [第三方声明](#第三方声明)

## 功能特性

- **物理 HCU**：发现、注册、健康检查与整卡分配
- **预切分 vHCU**：支持管理员预先划分的虚拟 HCU 实例的注册、健康检查与调度
- **动态切分 vHCU（HAMi 模式）**：Pod 启动时按算力/显存需求动态创建 vHCU，Pod 退出后自动回收
- **预切分 MIG HCU**：支持 MIG 实例的注册、健康检查与调度
- **NUMA 拓扑感知**：向 Kubelet 上报设备的 NUMA 节点信息
- **拓扑信息注册**（HAMi 模式）：将节点 HCU 互联拓扑写入 `kube-system/hcu-topology-info` ConfigMap

## 使用模式对比

| 模式 | 注册策略 | 典型资源 | 是否需要预切分 | 是否需要 Scheduler |
|------|----------|----------|----------------|-------------------|
| 物理整卡 | `hcu` / `mixed` | `hygon.com/hcu` | 否 | 否 |
| 预切分 vHCU | `vhcu` / `mixed` | `hygon.com/hcu-share-4c-16g` | 是（`hy-smi virtual`） | 否 |
| 动态切分 vHCU | `hami` | `hygon.com/hcunum` + `hcucores` + `hcumem` | 否 | 是（[k8s-hcu-scheduler](../k8s-hcu-scheduler)） |
| MIG | `mig` | `hygon.com/hcu-mig-*` | 是（`hy-smi mig`） | 否 |

## 架构概览

### 物理 / 预切分模式

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Node                       │
│  ┌──────────────┐    ┌──────────────────────────────┐   │
│  │   Kubelet    │◄──►│  HCU Device Plugin (DaemonSet)│   │
│  └──────────────┘    │  - ListAndWatch 设备列表      │   │
│                      │  - Allocate 分配设备到容器      │   │
│                      │  - 健康检查 (DCGM)             │   │
│                      └──────────┬───────────────────┘   │
│                                 │ DCGM                   │
│                      ┌──────────▼───────────────────┐   │
│                      │  /dev/dri  /dev/kfd  /dev/mkfd│   │
│                      │  /etc/vdev  /etc/dmi_mig_config│  │
│                      └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 动态切分 vHCU（HAMi 模式）

```
  Pod (hcunum/hcucores/hcumem)
           │
           ▼
  ┌─────────────────────┐
  │  Admission Webhook   │  改写 schedulerName，写入算力/显存注解
  └─────────┬───────────┘
            ▼
  ┌─────────────────────┐
  │  HCU Scheduler       │  选择节点与物理卡，写入分配注解
  └─────────┬───────────┘
            ▼
  ┌─────────────────────┐
  │  HCU Device Plugin   │  Allocate 阶段动态创建 vHCU，挂载配置文件
  │  (strategy=hami)     │  Pod 退出后自动销毁 vHCU
  └─────────────────────┘
```

## 代码结构

```
k8s-hcu-device-plugin/
├── cmd/
│   └── main.go                          # 程序入口：CLI 参数解析、DCGM 初始化、按策略启动插件
├── internal/pkg/
│   ├── plugin/
│   │   ├── plugin.go                    # Device Plugin 核心：ListAndWatch / Allocate / 健康检查
│   │   └── register.go                  # HAMi 模式：节点注解注册、Pod Informer、拓扑 ConfigMap
│   ├── util/
│   │   ├── hcu.go                       # HCU 设备发现、资源命名、NUMA / 健康检查
│   │   ├── util.go                      # HAMi 调度注解编解码、Pod / Node 补丁
│   │   ├── types.go                     # 设备与容器数据结构、常量定义
│   │   └── client/
│   │       └── client.go                # Kubernetes InCluster / kubeconfig 客户端
│   ├── log/
│   │   └── log.go                       # 全工程唯一日志入口（klog v2 封装、等级门控）
│   └── api/
│       └── device_register.go           # 设备信息 API 数据结构
├── deployment/
│   ├── static/                          # kubectl 直部署清单
│   │   ├── k8s-hcu-plugin.yaml          # mixed 模式（物理 HCU + 预切分 vHCU）
│   │   ├── k8s-hcu-plugin-mig.yaml      # MIG 模式
│   │   └── k8s-hcu-plugin-hami.yaml     # HAMi 动态切分模式（含 RBAC）
│   └── helm/                            # Helm Chart
│       ├── k8s-hcu-plugin/              # mixed 模式
│       ├── k8s-hcu-plugin-mig/          # MIG 模式
│       └── k8s-hcu-plugin-hami/         # HAMi 模式
├── demo/                                # Pod 使用示例
│   ├── pytorch-hcu.yaml                 # 物理 HCU
│   ├── pytorch-hcu-share.yaml           # 预切分 vHCU
│   ├── pytorch-hcu-mig.yaml             # MIG HCU
│   └── pytorch-hcu-dynamic-vhcu.yaml    # 动态切分 vHCU
├── build.sh                             # 编译二进制、构建镜像、导出离线包
├── Dockerfile                           # 运行时镜像（依赖节点挂载的 /opt/hyhal）
├── go.mod / go.sum                      # Go 模块依赖
├── LICENSE
├── THIRD_PARTY_NOTICES.md               # 第三方开源组件声明（版本 / 许可证 / 版权 / 修改情况）
└── README.md
```

### 核心模块说明

| 模块 | 职责                                                                                             |
|------|------------------------------------------------------------------------------------------------|
| `cmd/main.go` | 解析 `--strategy` 等 CLI 参数，初始化 DCGM，按策略为每种资源类型启动独立的 Device Plugin 实例                             |
| `plugin/plugin.go` | 实现 Kubelet Device Plugin gRPC 接口：`ListAndWatch` 上报设备列表，`Allocate` 将设备挂载到容器                     |
| `plugin/register.go` | HAMi 模式下监听 Pod 事件、向节点写入设备注册注解、维护 `hcu-topology-info` ConfigMap                                 |
| `util/hcu.go` | 通过 [hcu-dcgm](https://github.com/HYGON-AI/hcu-dcgm) 发现物理卡 / vHCU / MIG 实例，生成 `hygon.com/*` 资源名 |
| `util/util.go` | 与 [k8s-hcu-scheduler](../k8s-hcu-scheduler) 协作的注解编解码、节点锁、Pod 分配状态管理                            |
| `util/client` | 集群内 Kubernetes API 访问（InCluster 优先，回退 kubeconfig）                                              |
| `log/log.go` | 全工程统一日志入口：封装 klog v2，实现 `--log-severity` 等级门控与 glog（dpm 依赖）参数桥接                                |

### 运行时依赖

| 依赖                                                                                  | 说明 |
|-------------------------------------------------------------------------------------|------|
| [hcu-dcgm](https://github.com/HYGON-AI/hcu-dcgm)                                    | Go 模块，提供 HCU 设备发现、vHCU 创建/销毁、健康检查等能力 |
| [Project-HAMi/HAMi](https://github.com/Project-HAMi/HAMi)                           | HAMi 模式下的节点锁与调度协作工具 |
| [kubevirt/device-plugin-manager](https://github.com/kubevirt/device-plugin-manager) | Device Plugin 生命周期管理框架 |
| 节点 `/opt/hyhal`                                                                     | 运行时通过 hostPath 挂载，提供 HCU 底层库（镜像内不打包） |

## 前置要求

### 集群与节点

| 项目 | 说明 |
|------|------|
| Kubernetes 集群 | 已安装 HCU 的 Worker 节点 |
| HCU 驱动 | 节点已正确安装 HCU 驱动，`/sys/class/kfd` 存在 |
| hyhal | 节点存在 `/opt/hyhal` 目录，部署时以 hostPath 挂载到插件容器 |
| vHCU 功能 | 驱动版本 ≥ 6.2.26（6.2.26 之后虚拟化命令为 `hy-smi virtual`） |
| 节点标签 | 建议部署 [k8s-hcu-label-node](../k8s-hcu-label-node) 自动为 HCU 节点打上 `hygon.com/hcu=true` 等标签 |
| 动态 vHCU 额外要求 | DTK ≥ 24.04、`hy-smi` ≥ v1.6.0，并部署 [k8s-hcu-scheduler](../k8s-hcu-scheduler) |

### 本地构建

| 项目 | 说明 |
|------|------|
| Go | ≥ 1.22（见 `go.mod`） |
| CGO | 必须启用（`CGO_ENABLED=1`），用于链接 DCGM 库 |
| Docker | 构建容器镜像时需要 |
| HCU 节点 | 编译可在普通机器完成；功能验证需在已安装驱动的 HCU 节点上进行 |

## 资源注册策略

通过 `--strategy` 参数或环境变量 `RESOURCE_REGISTER_STRATEGY` 控制插件注册的资源类型：

| 策略 | 说明 | 注册的资源示例 |
|------|------|----------------|
| `hcu` | 仅注册物理 HCU | `hygon.com/hcu` |
| `vhcu` | 仅注册预切分 vHCU | `hygon.com/hcu-share-4c-16g` |
| `mig` | 仅注册 MIG 实例 | `hygon.com/hcu-mig-4g-31gb` |
| `mixed`（默认） | 同时注册物理 HCU 与 vHCU | `hygon.com/hcu`、`hygon.com/hcu-share-*` |
| `hami` | 动态共享模式 | `hygon.com/hcunum`（可选 `hcucores`、`hcumem`） |

### 资源命名规则（POLICY=0，默认）

- **物理 HCU**：`hygon.com/hcu`
- **预切分 vHCU**：`hygon.com/hcu-share-{CU数}c-{显存GB}g`，例如 `hygon.com/hcu-share-4c-16g`
- **MIG**：`hygon.com/hcu-mig-{规格}`，例如 `hygon.com/hcu-mig-4g-31gb`
- **动态 vHCU**：`hygon.com/hcunum`、`hygon.com/hcucores`、`hygon.com/hcumem`

`POLICY` 参数还支持按设备型号命名（`1`）或按型号+显存+CU 命名（`2`），详见下方配置参数说明。

## 配置参数

| 参数 / 环境变量 | 默认值 | 说明 |
|----------------|--------|------|
| `--strategy` / `RESOURCE_REGISTER_STRATEGY` | `mixed` | 资源注册策略 |
| `--policy` / `POLICY` | `0` | 资源命名策略：`0` 默认命名；`1` 使用设备型号；`2` 型号+显存+CU |
| `--pulse` / `PULSE` | `30` | 设备健康检查间隔（秒） |
| `--node-name` / `NODE_NAME` | - | 当前节点名称（部署时通过 Downward API 注入） |
| `--topology-register` / `TOPOLOGY_REGISTER` | `true` | HAMi 模式下是否注册 HCU 拓扑到 ConfigMap |
| `--resource-multiple` / `RESOURCE_MULTIPLE` | `false` | HAMi 模式下是否额外向 Kubelet 注册 `hcucores` 和 `hcumem` 资源 |
| `--device-split-count` / `DEVICE_SPLIT_COUNT` | `4` | HAMi 模式下单张物理 HCU 最多可切分的 vHCU 数量（须大于 0） |
| `--log-level` / `LOG_LEVEL` | `2` | 详细日志级别（0-10），控制 `V(n)` 日志的输出粒度 |
| `--log-severity` / `LOG_SEVERITY` | `INFO` | 日志输出的最低等级：`INFO` / `WARNING` / `ERROR`。设为 `WARNING` 时丢弃 Info 与 `V(n)` 日志 |

> 日志统一使用 klog，全部输出到 stderr（不落盘），由容器运行时统一收集。

## 部署

部署清单位于 `deployment/static/`（kubectl 直部署）和 `deployment/helm/`（Helm Chart）。

### 1. 安装 HCU 驱动

在 HCU 节点上安装并加载 HCU 驱动，确认 `hy-smi` 或 `hy-smi virtual` 可正常识别设备。

### 2. 部署节点标签组件（推荐）

```bash
# 部署 k8s-hcu-label-node，自动为 HCU 节点打标签 hygon.com/hcu=true
kubectl apply -f ../k8s-hcu-label-node/deployment/
```

### 3. 物理 HCU + 预切分 vHCU（mixed 模式，默认）

适用于普通 HCU 节点及已预切分 vHCU 的共享节点。DaemonSet 通过节点亲和性调度到 `hygon.com/hcu=true` 且非 MIG/HAMi 专用节点。

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin.yaml
```

**预切分 vHCU 步骤**（在需要共享 HCU 的节点上执行）：

1. 在物理 HCU 上创建 vHCU 实例：

```bash
# 6.2.26 之前
hy-virtual -d ${dev_id} \
  -create-vdevices ${num_vhcu} \
  -vdevice-compute-units $<cu_num, ...> \
  -vdevice-memory-size $<mem_size, ...>

# 6.2.26 及之后
hy-smi virtual -h   # 查看具体用法
```

2. 为节点打标签（可选，用于标识共享模式节点）：

```bash
kubectl label nodes <node-name> hcu-mode=share
```

3. 部署 Device Plugin（mixed 模式会自动发现并注册 vHCU 资源）。

### 4. MIG HCU 模式

1. 在物理 HCU 上创建 MIG 实例：

```bash
hy-smi mig -cgi ${gi_profile_id} -C -i ${dev_id}
# 更多用法：hy-smi mig -h 或参考《Hygon HCU Multi-Instance 使用手册》
```

2. 为节点打标签：

```bash
kubectl label nodes <node-name> hcu-mode=mig
```

3. 部署 MIG Device Plugin：

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin-mig.yaml
```

### 5. 动态切分 vHCU（HAMi 模式）

动态切分模式下，Device Plugin 在 Pod 容器启动的 `Allocate` 阶段调用 DCGM 动态创建 vHCU，Pod 结束后自动销毁，无需管理员提前划分实例。

**部署步骤：**

1. 为节点打标签：

```bash
kubectl label nodes <node-name> hcu=on
```

2. 部署 HAMi Device Plugin（含 RBAC）：

```bash
kubectl apply -f deployment/static/k8s-hcu-plugin-hami.yaml
```

3. 部署 HCU Scheduler 扩展组件（准入 Webhook + 自定义调度器），详见 [k8s-hcu-scheduler 部署文档](../k8s-hcu-scheduler/README.md#部署)：

```bash
# 安装 cert-manager 后
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-admission-webhook-certmanager.yaml
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-admission-webhook.yaml
kubectl apply -f ../k8s-hcu-scheduler/deployment/static/vhcu-scheduler.yaml
```

4. 确认节点已上报资源：

```bash
kubectl describe node <node-name> | grep -E 'hcunum|hcu-register'
```

### Helm 部署

```bash
# 默认 mixed 模式
helm install hcu-dp deployment/helm/k8s-hcu-plugin/

# MIG 模式
helm install hcu-dp-mig deployment/helm/k8s-hcu-plugin-mig/

# HAMi 模式
helm install hcu-dp-hami deployment/helm/k8s-hcu-plugin-hami/
```

> **注意**：Helm Chart 与 static 清单当前均使用镜像 tag `v3.0.0`。若构建了其他版本，需同步修改 `values.yaml` 中的 `image.tag`。

## 使用示例

`demo/` 目录提供了各类 HCU 资源的 Pod 示例，可直接 `kubectl apply -f demo/<文件名>` 使用。

### 物理 HCU（整卡）

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

### 预切分 vHCU

管理员需提前在节点上用 `hy-smi virtual` 划分实例，Device Plugin 以 `mixed` 或 `vhcu` 策略注册后，Pod 按规格名称申请：

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
          hygon.com/hcu-share-4c-16g: 1   # 按节点实际上报的规格名称替换
```

> 每个容器当前仅支持申请 1 个预切分 vHCU 实例。

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
          hygon.com/hcu-mig-4g-31gb: 1   # 按实际 MIG Profile 替换
```

```bash
kubectl apply -f demo/pytorch-hcu-mig.yaml
```

### 动态切分 vHCU（HAMi 模式）

Pod 通过三个扩展资源声明 HCU 需求，由 Scheduler 选择物理卡，Device Plugin 在容器启动时动态创建 vHCU：

| 资源 | 含义 | 取值说明 |
|------|------|----------|
| `hygon.com/hcunum` | HCU 槽位数 | 通常为 `1`；当算力或显存非零时，只能为 `1` |
| `hygon.com/hcucores` | 算力占比 | 1–100，`100` 表示独占整卡算力 |
| `hygon.com/hcumem` | 显存申请量 | 单位为 **MiB**（默认 `RESOURCE_MULTIPLE=false`） |

> `hcunum`、`hcucores` 通过 Device Plugin 的 ListAndWatch 上报；`hcumem` 因为按 MiB 计数，
> 8 卡节点的总量会超出 Device Plugin 单次上报的设备数量上限，改为由插件直接 patch 到
> Node 的 `status.capacity` / `status.allocatable`（需要 `nodes/status` 的 patch 权限）。

**部分共享示例**（申请 30% 算力、8 GB 显存）：

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

**整卡示例**（申请 100% 算力与全部显存，不创建 vHCU，直接使用物理卡）：

```yaml
resources:
  limits:
    hygon.com/hcunum: 1
    hygon.com/hcucores: 100
    hygon.com/hcumem: 32768   # 设为物理卡总显存（MB）
```

```bash
kubectl apply -f demo/pytorch-hcu-dynamic-vhcu.yaml
```

> 准入 Webhook 启用后会自动将 Pod 的 `schedulerName` 改写为 HCU 调度器；如未部署 Webhook，需手动设置 `spec.schedulerName: hcu-scheduler-plugin`。

**验证动态 vHCU 分配结果：**

```bash
# 查看 Pod 调度与绑定状态
kubectl get pod hcu-dynamic-vhcu-demo -o wide
kubectl describe pod hcu-dynamic-vhcu-demo | grep -E 'Annotations|hygon.com'

# 进入容器查看实际分配到的 vHCU
kubectl exec -it hcu-dynamic-vhcu-demo -- bash -c "source /opt/hygondriver/env.sh && hy-virtual -show-device-info"
```

预期输出类似：

```
Device 0:
        Actual Device: 0
        Compute units: 9
        Global memory: 8589934592 bytes
```

**Pod 退出后**，Device Plugin 会自动停止并销毁对应的动态 vHCU 实例，释放物理卡资源。

更多 Deployment / Job 等多容器示例见 [k8s-hcu-scheduler/example](../k8s-hcu-scheduler/example/) 目录。

## 构建

版本号由 `git describe --tags --dirty` 自动推导，无需手工维护；仓库中没有 tag 时 `build.sh` 会直接报错退出。

```bash
git tag v3.0.0          # 首次构建前需先打 tag
./build.sh              # 编译二进制、构建 Docker 镜像、导出离线 tar 包
```

构建产物（以下 `${VERSION}` 即 `git describe` 的输出，例如 `v3.0.0`）：

| 产物 | 路径 / 名称 |
|------|-------------|
| 二进制 | `k8s-device-plugin` |
| 镜像 | `harbor.sourcefind.cn:5443/hcu/admin/base/hcu-device-plugin:${VERSION}` |
| 离线包 | `hcu-device-plugin-${VERSION}.tar` |

手动编译：

```bash
export CGO_ENABLED=1
go mod tidy
go build -ldflags "-X 'main.version=$(git describe --tags --dirty)'" -o k8s-device-plugin cmd/main.go
```

> 工作区有未提交改动时 `git describe --dirty` 会带上 `-dirty` 后缀（如 `v3.0.0-dirty`），该后缀符合 Docker tag 命名规则，可直接用作镜像 tag。

> 镜像运行时通过 `LD_LIBRARY_PATH` 加载节点挂载的 `/opt/hyhal/lib`，无需在镜像内打包 HCU 底层库。

## 验证

```bash
# 查看 Device Plugin Pod 状态（mixed 模式）
kubectl get pods -n kube-system -l name=hcu-dp-ds

# HAMi 模式
kubectl get pods -n kube-system -l name=hcu-dp-ds-hami

# 查看节点 HCU 资源
kubectl describe node <node-name> | grep hygon.com

# 查看 Device Plugin 日志
kubectl logs -n kube-system -l name=hcu-dp-ds
```

## License

本项目部分代码基于 [HAMi](https://github.com/Project-HAMi/HAMi) 改编，Hygon 的修改与原创贡献均采用 [Apache License 2.0](LICENSE)。

## 第三方声明

本项目通过 Go module 引入的全部第三方开源组件，其仓库地址、固定版本、许可证类型、本地路径、版权声明及 HYGON 修改情况，
已逐项记录在 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) 中，依赖清单与 [go.mod](go.mod) 保持一致。

- 所有依赖均为宽松型开源许可证（Apache-2.0 / BSD-3-Clause / BSD-2-Clause / MIT / ISC），不含 GPL、LGPL、AGPL 等 Copyleft 许可证，与本项目的 Apache License 2.0 兼容
- 除 README [License](#license) 一节声明的 HAMi 改编代码外，其余第三方组件均按上游原样引入，未做源码修改
- 依赖完整的许可证正文以各组件发行包内的 `LICENSE` / `NOTICE` 文件为准，执行 `go mod vendor` 后可在 `vendor/` 对应目录下查阅
- 新增或升级依赖（修改 `go.mod`）时，须同步更新 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) 中的对应条目
