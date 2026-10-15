本手册主要针对vHCU虚拟化设备插件的安装与初步使用测试进行介绍，包括物理机和容器环境下vHCU设备的使用。

## 1.1 HCU软件虚拟化介绍

HCU 软件虚拟化的主要功能为提供HCU设备的软件虚拟化支持，下文简称为virtual，virtual能够将HCU设备最多分为4个虚拟HCU设备。配置及管理HCU设备软件虚拟化的相关命令行接口由hy-smi的virtual子命令组提供，使用方式为hy-smi virtual [options]。

### 1.1.1 virtual基本参数说明

virtual软件虚拟化配置及管理工具的命令行基本参数如下所示：

```
[root@localhost ~]# hy-smi virtual -h
Usage: hy-smi virtual [options] 

Description:                            The virtual is C-3000 HCU software virtualization.

Options:
 -d --device <number>                   Execute command on specified device.
 -v --vdevice <number>                  Execute command on specified virtual device.
 -version --version                     Print version information and exit.
 -h --help                              Print available options and exit.
 -log-level --log-level <level>         Set the log level: none, warning, info(default), debug.
```

命令行基本参数说明如下：

| 参数                            | 说明                                                         | 示例                                           |
| ------------------------------- | ------------------------------------------------------------ | ---------------------------------------------- |
| -d, --device <number>           | 指定特定的物理设备，与其他device相关命令配合使用。           | hy-smi virtual --show-device-info --device 0   |
| -v, --vdevice <number>          | 指定特定的虚拟设备，与其他vdevice相关命令配合使用。          | hy-smi virtual --show-vdevice-info --vdevice 0 |
| -version, --version             | 打印virtual命令行接口的版本号。                              |                                                |
| -h, --help                      | 打印virtual命令行接口的帮助信息。                            |                                                |
| -log-level, --log-level <level> | 指定log的打印等级，有none, warning, info, debug四个等级，默认为info。 | hy-smi virtual --log-level debug               |

### 1.1.2 virtual虚拟化参数说明

virtual软件虚拟化配置及管理工具的虚拟化参数如下所示：

```
[root@localhost ~]# hy-smi virtual -h
Usage: hy-smi virtual [options] 

Description:                            The virtual is C-3000 HCU software virtualization.

Virtualization options:
 -create-vdevices --create-vdevices <number> 
                                        Create a specified number of virtual devices.
 -destroy-vdevices --destroy-vdevices   Destroy the created virtual devices.
 -update-vdevice --update-vdevice <cu>,<memory> 
                                        Update virtual device's configuration with comma-seperated 
                                        compute units and memory size(MiB), * means do not update 
                                        corresponding field, must specify the target virtual device 
                                        by '-v/--vdevice' parameter.
 -vdevice-compute-units --vdevice-compute-units <string> 
                                        When creating virtual devices, specify the number of 
                                        comma-separated compute units for the virtual device.
 -vdevice-memory-size --vdevice-memory-size <string> 
                                        When creating virtual devices, specify the comma-split 
                                        memory size(MiB) for the virtual device.
 -show-device-info --show-device-info   Show the memory size(Bytes) and compute units of a specified 
                                        physical device.
 -show-vdevice-info --show-vdevice-info Show the memory size(Bytes) and compute units of a specified 
                                        virtual device.
 -show-remaining-info --show-remaining-info
                                        Show the remaining memory size(Bytes) and compute units of a 
                                        specified device for virtual device's further use.
```

HCU设备虚拟化参数说明如下：

| 参数                                                     | 说明                                                         | 示例                                                         |
| -------------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| -create-vdevices, --create-vdevices <number>             | 创建虚拟设备，数量为1~4，单个物理设备最大支持4个；默认为所有设备创建；配合-d参数，可以指定物理设备进行创建；<br>该命令行选项仅支持在宿主机中使用，且需要root权限。 | hy-smi virtual --create-vdevices 4 -d 0                      |
| -vdevice-compute-units, --vdevice-compute-units <string> | 创建虚拟设备时，指定每个虚拟设备的CU个数；<br>该参数可选，若未指定，则将物理设备的CU平均分配给每个虚拟设备。 | hy-smi virtual --create-vdevices 4 --vdevice-compute-units 15,15,15,15 |
| -vdevice-memory-size, --vdevice-memory-size <string>     | 创建虚拟设备时，指定每个虚拟设备的VRAM大小（MiB）；<br>该参数可选，若未指定，则将物理设备的VRAM平均分配给每个虚拟设备<br>。 | hy-smi virtual --create-vdevices 4 --vdevice-memory-size 1024,1024,1024,1024 |
| -update-vdevice, --update-vdevice <cu>,<memory>          | 更新创建的虚拟设备的配置信息，必须指定命令参数cu和memory；<br>cu表示虚拟设备需要更新的CU数量，*表示不更新CU；<br>memory表示虚拟设备需要更新的VRAM大小（MiB），*表示不更新VRAM；<br>使用该命令行选项时，必须通过-v参数指定虚拟设备；<br>该命令行选项仅支持在宿主机中使用，且需要root权限。 | #仅更新CU数量<br>hy-smi virtual --update-vdevice 30,* -v 0<br>#仅更新VRAM大小<br>hy-smi virtual --update-vdevice *,2048 -v 1<br>#更新CU和VRAM<br>hy-smi virtual --update-vdevice 30,2048 -v 2 |
| -destroy-vdevices, --destroy-vdevices                    | 销毁创建的虚拟设备，默认销毁所有虚拟设备；<br>配合-d参数，可以销毁指定物理设备的所有虚拟设备，无法与-v参数共存；<br>配合-v参数，可以销毁指定的虚拟设备，无法与-d参数共存；该命令行选项仅支持在宿主机中使用，且需要root权限。 | #销毁所有虚拟设备<br>hy-smi virtual --destroy-vdevices<br>#销毁指定物理设备的所有虚拟设备<br>hy-smi virtual --destroy-vdevices -d 0<br>#销毁指定的虚拟设备<br>hy-smi virtual --destroy-vdevices -v 0 |
| -show-device-info, --show-device-info                    | 显示物理设备相关信息，默认显示所有物理设备的相关信息；<br>配合-d参数，可显示指定的物理设备相关信息；<br>该命令行选项支持在宿主机及容器中使用，在容器中使用时，该选项显示绑定至容器中的虚拟设备相关信息。 | hy-smi virtual --show-device-info -d 0                       |
| -show-vdevice-info, --show-vdevice-info                  | 显示虚拟设备相关信息，默认显示所有虚拟设备的相关信息；<br>配合-d参数，可显示指定物理设备的所有虚拟设备相关信息；<br>配合-v参数，可显示指定虚拟设备的相关信息；<br>该命令行选项仅支持在宿主机中使用。 | #显示所有虚拟设备信息<br>hy-smi virtual --show-vdevice-info<br>#显示指定物理设备的虚拟设备<br>hy-smi virtual --show-vdevice-info -d 0<br>#显示指定虚拟设备的信息<br>hy-smi virtual --show-vdevice-info -v 0 |
| -show-remaining-info, --show-remaining-info              | 显示物理设备的剩余资源信息，默认显示所有物理设备的剩余资源信息；<br>配合-d参数，可显示指定物理设备的剩余资源信息；该命令行选项仅支持在宿主机中使用。 | hy-smi virtual --show-remaining-info -d 0                    |

### 1.1.3 virtual支持加密虚拟机参数说明

virtual软件虚拟化配置及管理工具的支持加密虚拟机参数如下所示：

```
Enctypted VM options:
 -enable-encrypted --enable-encrypted   Enable encryption virtual machine.
 -disable-encrypted --disable-encrypted Disable encryption virtual machine.
 -show-encrypted-status --show-encrypted-status
                                        Show encryption virtual machine status.
```

 HCU支持加密虚拟机参数说明如下：

| 参数                                            | 说明                       |
| ----------------------------------------------- | -------------------------- |
| -enable-encrypted, --enable-encrypted           | 使能加密虚拟机支持。       |
| -disable-encrypted, --disable-encrypted         | 禁用加密虚拟机支持。       |
| -show-encrypted-status, --show-encrypted-status | 显示当前加密虚拟机支持状态 |

## 1.2 HCU软件虚拟化使用示例

### 1.2.1 软件虚拟化配置

virtual命令行工具的虚拟化配置功能支持多种应用场景，方便在宿主机及容器下运行计算程序。下文仅介绍在容器中使用虚拟设备的示例，若需要在宿主机中使用虚拟设备运行计算程序，需要借助hdms工具，具体操作方法见《hdms介绍及使用示例》。

当虚拟设备在宿主机或容器中被使用时，无法销毁虚拟设备或更新虚拟设备配置。若需要对虚拟设备进行上述操作，应首先结束宿主机进程或容器进程，若为docker容器，还需进一步销毁容器。

#### 1.2.1.1 计算单元（CU）共享的虚拟设备

该应用场景支持为每个虚拟设备配置不同的VRAM大小，每个虚拟设备使用的计算单元（CU）共享。

配置步骤主要包含创建虚拟设备、绑定虚拟设备到容器以及容器中验证设备信息。

1. 创建虚拟设备

```
hy-smi virtual --create-vdevices 4 --vdevice-compute-units 0,0,0,0 --vdevice-memory-size 4096,4096,4096,4096 -d 0
```

该示例指定创建4个虚拟设备，每个虚拟设备的VRAM大小为4096MiB，指定CU个数为0代表该虚拟设备会和其他虚拟设备共享CU资源。

2. 绑定虚拟设备到容器

若要将虚拟设备绑定至容器，需要在创建容器时，通过容器的卷映射命令参数将/etc/vdev/vdev*.conf映射至容器中的/etc/vdev/docker/vdev*.conf。当成功启动容器后，即可直接在容器中运行计算程序，操作过程与宿主机环境一致。

示例一，绑定虚拟设备v0到Docker容器1：

```
docker run -it --name test_virtual_1 \
    --hostname test_virtual_1 \
    --device=/dev/dri --device=/dev/kfd --device=/dev/mkfd \
    -v /etc/vdev/vdev0.conf:/etc/vdev/docker/vdev0.conf:ro \
    {镜像名称} /bin/bash
```

示例二，绑定虚拟设备v1,v2,v3到Docker容器2：

```
docker run -it --name test_virtual_2 \
    --hostname test_virtual_2 \
    --device=/dev/dri --device=/dev/kfd --device=/dev/mkfd \
    -v /etc/vdev/vdev1.conf:/etc/vdev/docker/vdev1.conf:ro \
    -v /etc/vdev/vdev2.conf:/etc/vdev/docker/vdev2.conf:ro \
    -v /etc/vdev/vdev3.conf:/etc/vdev/docker/vdev3.conf:ro \
    {镜像名称} /bin/bash
```

示例三，验证设备信息（Docker容器1）：

```
#进入容器验证设备信息
hy-smi virtual --show-device-info
root@test_virtual_1:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 0
    Global memory: 4294967296 bytes
```

示例四，验证设备信息（Docker容器2）：

```
#进入容器验证设备信息
hy-smi virtual --show-device-info
root@test_virtual_2:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 0
    Global memory: 12884901888 bytes
```

注意事项：

• 所有虚拟设备的VRAM大小之和不能超过物理设备的VRAM大小。

• 一个虚拟设备仅能被绑定至一个容器。

• 6.3.16版本之前的驱动，计算单元（CU）共享的虚拟设备仅支持绑定至Docker容器。

• 单个容器中可以绑定多个虚拟设备，当多个计算单元（CU）共享的虚拟设备绑定至同一个容器后，执行程序时，相同物理设备上的虚拟设备被自动合并，合并后虚拟设备的VRAM大小为各虚拟设备的VRAM大小之和。

#### 1.2.1.2 计算单元（CU）独立的虚拟设备

该应用场景支持为每个虚拟设备配置不同的CU数量和VRAM大小，步骤如下：

1. 创建虚拟设备

```
 hy-smi virtual --create-vdevices 4 --vdevice-compute-units 10,10,20,20 --vdevice-memory-size 4096,4096,4096,4096 -d 0
```

该示例为索引号为0的物理卡创建了4个虚拟设备，每个虚拟设备的CU个数分别为10、10、20、20，VRAM大小分别为4096、4096、4096、4096（MiB）。

2. 绑定虚拟设备到docker容器，该步骤操作与上文一致：

示例一，绑定虚拟设备v0到Docker容器：

```
docker run -it --name test_virtual_docker \
    --hostname test_virtual_docker \
    --device=/dev/dri --device=/dev/kfd --device=/dev/mkfd \
    -v /etc/vdev/vdev0.conf:/etc/vdev/docker/vdev0.conf:ro \
    {镜像名称} /bin/bash
```

示例二，绑定虚拟设备v1,v2,v3到Containerd容器：

```
ctr run -t --hostname test_virtual_containerd \
    --device=/dev/dri --device=/dev/kfd --device=/dev/mkfd \
    --mount type=bind,src=/etc/vdev/vdev1.conf,dst=/etc/vdev/docker/vdev1.conf,options=rbind:ro \
    --mount type=bind,src=/etc/vdev/vdev2.conf,dst=/etc/vdev/docker/vdev2.conf,options=rbind:ro \
    --mount type=bind,src=/etc/vdev/vdev3.conf,dst=/etc/vdev/docker/vdev3.conf,options=rbind:ro \
    {镜像名称}  test_virtual_containerd /bin/bash
```

示例三，验证设备信息（Docker容器）：

```
#进入容器验证设备信息
hy-smi virtual --show-device-info
root@test_virtual_docker:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 10
    Global memory: 4294967296 bytes
```

示例四，验证设备信息（Containerd容器）：

```
#进入容器验证设备信息
hy-smi virtual --show-device-info
root@test_virtual_containerd:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 10
    Global memory: 4294967296 bytes

Device 1:
    Physical Device: 0
    Compute units: 20
    Global memory: 4294967296 bytes

Device 2:
    Physical Device: 0
    Compute units: 20
    Global memory: 4294967296 bytes
```

注意事项：

•所有虚拟设备的VRAM大小之和不能超过物理设备的VRAM大小。

•所有虚拟设备的CU数量之和不能超过物理设备的CU数量。

•一个虚拟设备仅能被绑定到一个容器。

•单个容器中可以绑定多个虚拟设备，当多个虚拟设备绑定至同一个容器时，每个虚拟设备均被视为独立的异构加速设备。

#### 1.2.1.3 虚拟设备更新

该场景支持在虚拟设备空闲的情况下更新指定虚拟设备的配置信息，包括更新CU数量和VRAM大小，具体示例如下：

示例一，更新虚拟设备v0的CU数量：

```
#将虚拟设备v0的CU数量更新至30，并且保持其VRAM大小不变
hy-smi virtual --update-vdevice 30,* -v 0
$ hy-smi virtual --update-vdevice 30,* -v 0
The virtual device is updated successfully!
```

该示例将虚拟设备v0的CU数量更新至30，并且保持其VRAM大小不变。

示例二，更新虚拟设备v1的VRAM大小：

```
#将虚拟设备v1的VRAM大小更新至1024MiB，并且保持其CU数量不变
hy-smi virtual --update-vdevice *,1024 -v 1
$ hy-smi virtual --update-vdevice *,1024 -v 1
The virtual device is updated successfully!
```

该示例将虚拟设备v1的VRAM大小更新至1024MiB，并且保持其CU数量不变。

示例三，更新虚拟设备v2的CU数量和VRAM大小：

```
#将虚拟设备v2的CU数量更新至30，并且将VRAM大小更新至2048MiB
hy-smi virtual --update-vdevice 30,2048 -v 1
$ hy-smi virtual --update-vdevice 30,2048 -v 1
The virtual device is updated successfully!
```

该示例将虚拟设备v2的CU数量更新至30，并且将VRAM大小更新至2048MiB。

注意：

•更新配置后所有虚拟设备的CU数量之和不能超过物理设备的CU数量。

•更新配置后所有虚拟设备的VRAM大小之和不能超过物理设备的VRAM大小。

### 1.2.2 显示设备信息

宿主机显示物理设备信息的示例如下：

```
 hy-smi virtual --show-device-info
$ hy-smi virtual --show-device-info
Device 0:
    Actual Device: 0
    Compute units: 120
    Global memory: 34342961152 bytes
```

宿主机显示虚拟设备信息的示例如下：

```
hy-smi virtual --show--vdevice-info
$ hy-smi virtual --show-vdevice-info
Virtual Device 0:
    Actual Device: 0
    Compute units: 30
    Global memory: 4294967296 bytes
Virtual Device 1:
    Actual Device: 0
    Compute units: 10
    Global memory: 1073741824 bytes
Virtual Device 2:
    Actual Device: 0
    Compute units: 30
    Global memory: 2147483648 bytes
Virtual Device 3:
    Actual Device: 0
    Compute units: 20
    Global memory: 4294967296 bytes
```

宿主机显示设备剩余可划分虚拟设备的资源信息示例如下：

```
hy-smi virtual --show-remaining-info
$ hy-smi virtual --show-remaining-info
Device 0:
    Remaining compute units: 30
    Remaining memories: 22531801088 bytes
```

容器中显示所分配虚拟设备信息的示例如下：

```
#容器中执行
hy-smi virtual --show-device-info
root@test_virtual_1:/# hy-smi virtual --show-device-info
Device 0:
    Physical Device: 0
    Compute units: 0
    Global memory: 4294967296 bytes
```

### 1.2.3 加密虚拟机设置

使能加密虚拟机的示例如下：

```
hy-smi virtual --enable-encrypted
[root@localhost ~]# hy-smi virtual --enable-encrypted
Enable successfully!
```

禁用加密虚拟机的示例如下：

```
 hy-smi virtual --disable-encrypted
[root@localhost ~]# hy-smi virtual --disable-encrypted
Disable successfully!
```

查看加密虚拟机配置状态的示例如下：

```
 hy-smi virtual --show-encrypted-status
[root@localhost ~]#  hy-smi virtual --show-encrypted-status
Encrypted status: Enable!
```

## 1.3 hdms介绍及使用示例

### 1.3.1 hdms介绍

hdms是一个用于在宿主机环境下启动HCU虚拟设备并执行计算程序的命令行工具。

### 1.3.2 hdms参数说明

hdms工具的命令行参数如下图所示：

```
[root@localhost ~]# hdms -h
Usage: hdms [options] 

Description:                            The hdms is C-3000 HCU device management service.

Options:
 -log-level --log-level <level>         Set the log level: info(default), debug, trace.
 -log-file --log-file <file>            Set the file used to write the log.
 --version                              Print version information and exit.
 -h --help                              Print available options and exit.

Virtualization options:
 -p --process <process>                 Specifies the running process.
 -v --vdevice <number>                  Specifies the running vdevs.
```

命令行参数说明如下：

| 参数                            | 说明                                                         | 示例                       |
| ------------------------------- | ------------------------------------------------------------ | -------------------------- |
| -log-level, --log-level <level> | 指定log的打印等级，有info, debug, trace四个等级，默认为info等级。 | hdms --log-level info      |
| -log-file, --log-file <file>    | 指定log输出至特定文件，若不指定，则log默认输出至stdout。     | hdms --log-file ~/hdms.log |
| --version                       | 打印hdms命令行工具的版本号。                                 |                            |
| -h, --help                      | 打印hdms命令行工具的帮助信息。                               |                            |
| -p, --process <process>         | 指定使用hdms命令行工具运行的计算程序，该参数必选。           |                            |
| -v, --vdevice <number>          | 指定使用hdms命令行工具绑定的虚拟设备，该参数必选。           |                            |

### 1.3.3 hdms使用示例

示例一，将计算程序绑定至虚拟设备v0执行：

```
hdms -v 0 -p ./demo
$ hdms -v 0 -p ./demo
num of cus: 30
vram size: 2147483648 bytes
PASSED!
```

示例二，将计算程序绑定至虚拟设备v1执行：

```
hdms -v 0 -p ./demo
$ hdms -v 1 -p ./demo
num of cus: 10
vram size: 1073741824 bytes
PASSED!
```

