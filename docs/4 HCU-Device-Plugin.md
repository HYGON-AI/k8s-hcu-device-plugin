HCU-Device-Plugin负责管理 Kubernetes 集群中的 HCU/vHCU 设备资源分配。它通过 Kubernetes 的设备插件框架（Device Plugin Framework）将 HCU/vHCU 设备暴露给k8s，使应用程序能够直接使用 HCU/vHCU 加速计算任务。支持HCU物理卡、预切分vHCU与动态切分vHCU方案，兼容HAMi项目，从而更灵活地分配计算资源，满足不同应用场景的需求。

HCU-Device-Plugin提供三种工作模式：

(1)    标准模式

•支持物理HCU卡与预切分vHCU的管控。

•通过资源名称区分物理 HCU 和不同规格的 vHCU，例如：

`hygon.com/hcu` → 物理卡整卡调度

`hygon.com/hcu-share-40c-30g` → 40个计算单元30G显存规格的 vHCU

`hygon.com/hcu-share-20c-40g` → 20个计算单元40G显存规格的 vHCU

•规则：存在预切分 vHCU的HCU卡不会参与整卡调度;存在 vHCU 的卡必须通过 vHCU 规格资源名称调度，避免混合调度带来的 OOM 问题。

(2)    MIG 模式

用于管控 HCU 上已划分的 MIG 实例，负责在 Kubernetes 集群中进行资源注册、调度和分配。每个 MIG 实例作为独立的可调度资源对外暴露。

(3)    vHCU动态切分模式

作为HAMi调度器与自研vHCU-Scheduler调度器的HCU设备插件。提供 vHCU 动态切分能力，根据工作负载需求动态划分 HCU 实例，提高资源利用率和弹性。

HCU-Device-Plugin的三种工作模式，通过不同 YAML 文件部署，V2.3.0版本起HCU-Device-Plugin将之前多种部署方式的device-plugin镜像进行整合，使用唯一镜像进行部署（如hcu-device-plugin:v3.0.0），采用环境变量RESOURCE_REGISTER_STRATEGY参数的方式区分不同应用场景：

表 2.1 HCU-Device-Plugin 资源注册策略

| RESOURCE_REGISTER_STRATEGY | 含义                                                         | 资源类型                            |
| -------------------------- | ------------------------------------------------------------ | ----------------------------------- |
| hcu                        | 仅管控HCU物理设备                                            | `hygon.com/hcu `                      |
| vhcu                       | 仅管控预切分vHCU                                             | `hygon.com/hcu-share-*`               |
| mixed（默认）              | 标准模式：物理设备与预切分vHCU协同管控；存在预切分 vHCU的HCU卡不会参与整卡调度。 | `hygon.com/hcu` `hygon.com/hcu-share-*` |
| hami                       | 动态切分vHCU设备                                             | `hygon.com/hcunum`                    |
| mig                        | mig虚拟化设备管控                                            | `hygon.com/hcu-mig-*`                 |

## 2.1 标准模式

本文档涉及的yaml文件可以从[社区资源中心](https://download.sourcefind.cn:65024/5/main/Kubernetes%E6%8F%92%E4%BB%B6)进行下载，镜像可以从[光源镜像仓库](https://developer.sourcefind.cn/servicelist)直接拉取。

### 2.1.1 安装部署

k8s集群以DaemonSet形式部署标准模式HCU-Device-Plugin，DaemonSet部署命令如下：

```
#标准模式部署
kubectl apply -f  k8s-hcu-plugin.yaml
```

### 2.1.2 部署验证

用户可以通过查看HCU-Device-Plugin的Pod运行情况验证HCU-Device-Plugin是否部署成功：

```
#标准模式
kubectl get pod -n kube-system -owide | grep hcu-device-plugin
```

如下输出表示HCU-Device-Plugin已经成功部署：

```
[root@localhost ~]# kubectl get pod -n kube-system -owide | grep hcu-device-plugin
hcu-device-plugin-97bwn               1/1     Running   1 (7d20h ago)   23d     10.233.114.43     worker-200   <none>           <none>
hcu-device-plugin-pdps4               1/1     Running   0               23d     10.233.122.21     worker-183   <none>           <none>
```

HCU-Device-Plugin向k8s集群注册的HCU资源名为`hygon.com/hcu`，用户可以通过查看HCU节点k8s资源详情确认HCU资源是否注册成功：

```
kubectl describe node <node-name>
```

如下为一个HCU节点的资源注册实例，Capacity 字段中“`hygon.com/hcu`: 1”表示该节点有1块HCU整卡可用，“`hygon.com/hcu-share-*c-*g`”表示该节点有对应*c、*g的vHCU可用。

```
Capacity:
    cpu:                            16
    ephemeral-storage:              1526641Mi
    hugepages-1Gi:                  0
    hugepages-2Mi:                  0
    hygon.com/hcu:                  1
    hygon.com/hcu-share-30c-16g:    1
    hygon.com/hcu-share-36c-16g:    1
    hygon.com/hcu-share-36c-5g:     1
    hygon.com/hcunum:               0
    memory:                         38912856Ki
    pods:                           110

Allocatable:
    cpu:                            15600m
    ephemeral-storage:              1526641Mi
    hugepages-1Gi:                  0
    hugepages-2Mi:                  0
    hygon.com/hcu:                  1
    hygon.com/hcu-share-30c-16g:    1
    hygon.com/hcu-share-36c-16g:    1
    hygon.com/hcu-share-36c-5g:     1
    hygon.com/hcunum:               0
    memory:                         37330138288
    pods:                           110
```

### 2.1.3 应用实例

与其他k8s资源一样，用户可以根据所需要的HCU数量，在k8s工作负载yaml配置文件内的resource limit字段的设置进行HCU资源申请（resource limit字段取值只能为整数）。

以下是一个k8s工作负载调用HCU资源的示例：

```
apiVersion: v1
kind: Pod
metadata:
  name: hcu-pytorch-demo
spec:
  containers:
    - name: hcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu: 1 # requesting a HCU
```

k8s工作负载成功运行后，可以进入对应容器运行“hy-smi”命令查看HCU分配情况。

```
[root@localhost test]# kubectl exec -ti hcu-pytorch-demo -- /bin/bash
root@hcu-pytorch-demo:/# hy-smi

==================== System Management Interface ===================
====================================================================
HCU  Temp   AvgPwr Perf  PwrCap  VRAM% HCU% Mode
0    50.0C 77.0W  auto 400.0W 0%    0%   Normal
====================================================================
========================= End of SMI Log =========================
```

以下是一个k8s工作负载调用一个拥有30个计算单元、16g显存的vHCU资源的示例：

```
apiVersion: v1
kind: Pod
metadata:
  name: hcu-share-pytorch-demo
spec:
  containers:
    - name: hcu-share-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu-share-30c-16g: 1 # requesting a vHCU
```

k8s工作负载成功运行后，可以进入对应容器运行 hy-smi virtual --show-device-info 命令查看vHCU分配情况。

```
root@hcu-share-pytorch-demo://# source /opt/hyhal/env.sh
root@hcu-share-pytorch-demo://# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 30
    Global memory: 17179869184 bytes
    Usage memory: 1900896256 bytes (11.06%)
    Dev busy percent : 100 %
```

【vHCU使用限制】：每个container最多可以申请一张 vHCU。当申请多于 1 张 vHCU 时，K8s 资源池会削减申请数量的 vHCU，容器内只有一张vHCU 会生效。

## 2.2 MIG模式

本文档涉及的yaml文件可以从[社区资源中心](https://download.sourcefind.cn:65024/5/main/Kubernetes%E6%8F%92%E4%BB%B6)下载，镜像可以从[光源镜像仓库](https://developer.sourcefind.cn/servicelist)直接拉取。

使用MIG模式需要对应的宿主机开启MIG功能，并创建所需要的MIG设备。具体操作可参考《[HCU Multi-Instance使用手册](https://developer.sourcefind.cn/document/9169ef18-c10d-11f0-b077-0242ac150003?id=4a82aeed-e242-11f0-b9e4-0242ac150003&title=3+HCU+Multi-Instance%E4%BD%BF%E7%94%A8%E6%89%8B%E5%86%8C&version=9169ef18-c10d-11f0-b077-0242ac150003)》。

【注意】开启MIG功能之后，vHCU虚拟化功能不可用。

### 2.2.1 安装部署

使用如下命令部署MIG模式的device-plugin

```
# MIG模式部署
kubectl label nodes <node-name> hcu-mode=mig
kubectl  apply  -f  k8s-hcu-plugin-mig.yaml
```

### 2.2.2 部署验证

用户可以通过查看HCU-Device-Plugin的Pod运行情况验证MIG模式的HCU-Device-Plugin是否部署成功：

#MIG模式

```
kubectl get pod -n kube-system -owide | grep hcu-device-plugin-mig
```

如下输出表示MIG模式的HCU-Device-Plugin已经成功部署：

```
[root@localhost ~]# kubectl -n kube-system get po -o wide |grep mig
hcu-device-plugin-mig-6kfqg         1/1        Running    0    32s    10.233.64.8    master     <none>     <none>
hcu-device-plugin-mig-md6qz         1/1        Running    0    32s    16.233.65.77   worker1    <none>     <none>
```

HCU-Device-Plugin向k8s集群注册的MIG模式HCU资源名为`hygon.com/hcu-mig-*g-*gb`用户可以通过查看HCU节点k8s资源详情确认HCU资源是否注册成功：

```
kubectl describe node  <node-name>
```

如下为一个HCU节点的资源注册实例，Capacity 字段中“`hygon.com/hcu-mig-2g-15gb`: 1”表示该节点有一块对应规格的MIG设备。

```
Capacity:
  cpu:                           60
  ephemeral-storage:             297841Mi
  hugepages-1Gi:                 0
  hugepages-2Mi:                 0
  hygon.com/hcu-mig-2g-15gb:     1
  hygon.com/hcu-mig-4g-31gb:     1
  memory:                        131734840Ki
  pods:                          110
Allocatable:
  cpu:                           60
  ephemeral-storage:             297841Mi
  hugepages-1Gi:                 0
  hugepages-2Mi:                 0
  hygon.com/hcu-mig-2g-15gb:     1
  hygon.com/hcu-mig-4g-31gb:     1
  memory:                        131734840Ki
  pods:                          110
```

### 2.2.3 应用实例

以下是一个k8s工作负载调用MIG实例的示例：

```
apiVersion: v1
kind: Pod
metadata:
  name: hcu-mig-pytorch-demo
spec:
  containers:
    - name: hcu-mig-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcu-mig-4g-31gb: 1 # requesting a mig instance
```

k8s工作负载成功运行后，可以进入对应容器运行“hy-smi”命令查看MIG模式的HCU分配情况。

```
root@localhost ~]# kubectl exec -ti hcu-mig-pytorch-demo -- bash
root@hcu-mig-pytorch-demo:/tmp# hy-smi

========= System Management Interface =========
HCU Temp  AvgPwr Perf PwrCap  VRAM% HCU%   Mode
0   50.0C 66.0W  auto 400.00W N/A   N/A  Normal
===============================================
============================= MIG =============================
HCU GI CI MIG DEV BUS          CU VRAM USED/TOTAL (MiB) HCU (%)
0   1  2  0       0000:06:00.0 60 0/32760 (0%)                0
===============================================================

======================= End of SMI Log ========================
```

