## 2.1 组件安装部署

Kubernetes集群通过设备插件HCU-Device-Plugin与vHCU-Scheduler（或[HAMi](https://github.com/Project-HAMi/HAMi)调度器）实现节点上vHCU的动态切分和自动回收。k8s集群内通过yaml编排文件中resources资源中的hygon.com/hcunum、hygon.com/hcucores和hygon.com/hcumem进行vhcu的动态切分请求，无需在使用vHCU前手动进行vHCU实例划分。

### 2.1.1 HAMi模式HCU-Device-Plugin安装部署

建议给节点打上相应资源标签后使用HAMi模式HCU-Device-Plugin插件管理vHCU。

```
# 此项为固定标签,部署设备插件时需要查询此标签 
kubectl label nodes <node-name> hcu=on
```

详细安装部署方法参考《HCU-Device-Plugin组件vHCU动态切分模式安装部署》

HCU-Device-Plugin向k8s集群注册的vHCU资源名为hygon.com/hcunum，用户可以通过查看HCU节点k8s资源详情确认vHCU资源是否注册成功：

```
kubectl describe node <node-name>
```

如下为一个vHCU节点的资源注册实例，Capacity 字段中“hygon.com/hcunum: 4”表示该节点最多可分配4个可用vHCU资源。

```
Capacity:
  cpu:                 60
  ephemeral-storage:   297841Mi
  hugepages-1Gi:       0
  hugepages-2Mi:       0
  hygon.com/hcunum:    4
  memory:              131734840Ki
  pods:                110
Allocatable:
  cpu:                 59600m
  ephemeral-storage:   297841Mi
  hugepages-1Gi:       0
  hugepages-2Mi:       0
  hygon.com/hcunum:    4
  memory:              127627364252
  pods:                110
```

### 2.1.2 vHCU-Scheduler安装部署

用于动态vHCU调度切分的vHCU-Scheduler组件包含vhcu-admission-webhook和vhcu-scheduler两个插件，部署命令如下：

```
kubectl apply -f vhcu-admission-webhook.yaml
kubectl apply -f vhcu-scheduler.yaml
```

用户可以通过查看动态vHCU切分相关的Pod运行情况验证插件是否部署成功：

```
kubectl get pod -n kube-system -owide | grep vhcu
```

如下输出表示插件已经成功部署：

```
[root@localhost ~]# kubectl get pod -n kube-system -owide | grep vhcu
vhcu-admission-webhook-cd66b84c9-b6tq9     1/1     Running   0               23d    10.233.105.36     master-171   <none>           <none>
vhcu-scheduler-plugin-85c77b87d4-2vlxm     1/1     Running   0               12d    10.233.105.46     master-171   <none>           <none>
```

## 2.2 使用示例

用户可以根据所需要的vHCU数量、HCU计算核心CU百分比和HCU显存容量，在k8s工作负载yaml配置文件内的resource limit字段的设置进行HCU资源申请（resource limit字段取值只能为整数）。

使用限制：每个container最多可以申请一张 vHCU，且不支持initContainers中申请vHCU。

### 2.2.1 指定算力与显存申请vHCU

以下是一个k8s工作负载使用动态切分方式调用vHCU资源的示例：

```
apiVersion: v1
kind: Pod
metadata:
  name: vhcu-pytorch-demo
spec:
  containers:
    - name: vhcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1   #申请1个vHCU
          hygon.com/hcucores: 25  #指定申请vHCU算力为25%
          hygon.com/hcumem: 16384  #指定申请vHCU的显存为16384MB(16GB)
```

k8s工作负载成功运行后，可以进入对应容器运行 hy-smi virtual --show-device-info 命令查看vHCU分配情况。

```
root@vhcu-pytorch-demo:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 30
    Global memory: 17179869184 bytes
    Usage memory: 1900896256 bytes (11.06%)
    Dev busy percent : 76 %
```

### 2.2.2 物理HCU卡申请

vHCU动态切分模式配合vHCU-Scheduler（或HAMi调度器）也可实现物理HCU卡的申请分配，若物理卡分配成功，对应物理卡将不会分配创建vHCU。

以下是一个k8s工作负载使用动态切分方式配合调度器申请分配物理HCU卡的示例：

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
          hygon.com/hcunum: 2   #申请2个物理HCU（即整卡资源，该资源请求数量可以大于0）
```

若系统采用Balance双路架构，通过PCIe Gen5与HSL总线实现CPU、PCIe Switch与HCU之间的高校链接，根据互联方式不同，会又两卡桥接和四卡桥接等形态，此时使用动态切分方式配合调度器申请分配物理HCU卡，调度器会优先选择卡间通信性能最高的HCU设备进行分配。

### 2.2.3 将任务分配到特定 kubernetes 节点

要将任务分配给特定的 kubernetes节点，只需在注释字段中添加 hygon.com/use-nodename 注解。

```
apiVersion: v1
kind: Pod
metadata:
  name: vhcu-pytorch-demo
  annotations:
    hygon.com/use-nodename: "worker01" #指定pod调度到名为 worker01 的节点
spec:
  containers:
    - name: vhcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1   #申请1个vHCU
          hygon.com/hcucores: 25  #指定申请vHCU算力为25%
          hygon.com/hcumem: 16384  #指定申请vHCU的显存为16384MB(16GB)
```

相似地，要将任务禁止分配给特定的 HCU，在注释字段中分配 hygon.com/nouse-nodename。

### 2.2.4 将任务分配给特定HCU

要将任务分配给特定的 HCU，只需在注释字段中添加 hygon.com/use-gpuuuid注解。

```
apiVersion: v1
kind: Pod
metadata:
  name: vhcu-pytorch-demo
  annotations:
    hygon.com/use-gpuuuid: "HCU-TPXS320016070601,HCU-TPXS320010040501" #指定以逗号分隔的HCU-UUID，并添加HCU前缀
spec:
  containers:
    - name: vhcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1   #申请1个vHCU
          hygon.com/hcucores: 25  #指定申请vHCU算力为25%
          hygon.com/hcumem: 16384  #指定申请vHCU的显存为16384MB(16GB)
```

相似地，要将任务禁止分配给特定的 HCU，在注释字段中分配 hygon.com/nouse-gpuuuid。

### 2.2.5 将任务分配给指定HCU类型

要将任务分配给指定类型的HCU，只需在注解中添加hygon.com/use-hcutype注解。

```
apiVersion: v1
kind: Pod
metadata:
  name: vhcu-pytorch-demo
  annotations:
    hygon.com/use-hcutype: "HCU-K100_AI" #具体型号前添加HCU前缀
spec:
  containers:
    - name: vhcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1   #申请1个vHCU
          hygon.com/hcucores: 25  #指定申请vHCU算力为25%
          hygon.com/hcumem: 16384  #指定申请vHCU的显存为16384MB(16GB)
```

相似地，要将任务禁止分配给指定类型的 HCU，在注释字段中分配 hygon.com/nouse-hcutype。

### 2.2.6 仅指定显存申请vHCU

HCU驱动从6.3.16版本后开始支持vHCU仅指定显存共享物理卡算力的切分方式，vHCU-Scheduler从v2.4.0版本开始支持仅指定显存申请vHCU。仅指定显存申请vHCU，只需资源申请时缺省hygon.com/hcucores算力资源申请即可。(若需要指定显存切分请下载6.3.16-V1.1.0c.run版本驱动)

```
apiVersion: v1
kind: Pod
metadata:
  name: vhcu-pytorch-demo
  annotations:
    hygon.com/use-hcutype: "HCU-K100_AI" #具体型号前添加HCU前缀
spec:
  containers:
    - name: vhcu-pytorch-demo
      image: image.sourcefind.cn:5000/hcu/admin/base/pytorch:2.1.0-ubuntu22.04-dtk24.04.2-py3.10
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "sleep infinity & wait" ]
      resources:
        limits:
          hygon.com/hcunum: 1   #申请1个vHCU
          hygon.com/hcumem: 16384  #指定申请vHCU的显存为16384MB(16GB)
```

