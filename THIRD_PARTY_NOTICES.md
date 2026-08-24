<!--
SPDX-License-Identifier: Apache-2.0
Copyright 2026 Hygon Information Technology Co., Ltd.
-->

# 第三方软件声明（Third Party Notices）

本文件列出 Hygon HCU Device Plugin（`github.com/HYGON-AI/k8s-hcu-device-plugin`）所使用的全部第三方开源组件，
依据 `go.mod` 逐项生成，包含仓库地址、固定版本、许可证、本地路径、版权声明及海光（HYGON）修改情况。

- 依赖清单来源：仓库根目录 `go.mod`（module `github.com/HYGON-AI/k8s-hcu-device-plugin`，go 1.22.2）
- 许可证与版权信息来源：各依赖发行包内的 `LICENSE` / `LICENSE.md` / `LICENSE.txt` / `NOTICE` 文件及源码文件头
- 本项目自身采用 [Apache License 2.0](LICENSE)
- 除特别说明外，所有第三方组件均按上游原样引入，未做任何源码修改

> **关于「本地路径」**：本仓库当前未提交 `vendor/` 目录，依赖由 Go module 机制从 `go.sum` 校验后拉取。
> 表中 `vendor/<module path>` 为执行 `go mod vendor` 后各依赖在本地的落盘路径，用于离线构建与合规审计场景。

## 目录

- [直接依赖](#直接依赖)
- [间接依赖](#间接依赖)
- [许可证汇总](#许可证汇总)

## 直接依赖

### github.com/HYGON-AI/hcu-dcgm/v3

- 项目/仓库：https://github.com/HYGON-AI/hcu-dcgm
- 固定版本：v3.0.0
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/HYGON-AI/hcu-dcgm/v3
- 版权声明：Copyright (c) 2026 Hygon Information Technology Co., Ltd.; Licensed under the Apache License, Version 2.0
- HYGON 修改：海光自研模块，作为独立仓库按发布版本引入

### github.com/Project-HAMi/HAMi

- 项目/仓库：https://github.com/Project-HAMi/HAMi
- 固定版本：v0.0.0-20250125070959-ab547e40cc64
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/Project-HAMi/HAMi
- 版权声明：Copyright HAMi Contributors; Copyright (c) 2024, HAMi. All rights reserved.; Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.; Copyright 2019 The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：模块本身按上游原样引入，未修改其源码；本项目 `internal/pkg/util` 等部分代码参考 HAMi 实现改编，改编部分已在 [README.md](README.md#license) 声明并以 Apache-2.0 发布

### github.com/kubevirt/device-plugin-manager

- 项目/仓库：https://github.com/kubevirt/device-plugin-manager
- 固定版本：v1.19.5
- 许可证：MIT
- 本地路径：vendor/github.com/kubevirt/device-plugin-manager
- 版权声明：Copyright (c) 2017-2018 Red Hat, Inc.; The MIT License
- HYGON 修改：无（按上游原样引入）

### github.com/urfave/cli/v2

- 项目/仓库：https://github.com/urfave/cli
- 固定版本：v2.27.1
- 许可证：MIT
- 本地路径：vendor/github.com/urfave/cli/v2
- 版权声明：Copyright (c) 2022 urfave/cli maintainers; The MIT License
- HYGON 修改：无（按上游原样引入）

### golang.org/x/net

- 项目/仓库：https://github.com/golang/net
- 固定版本：v0.28.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/net
- 版权声明：Copyright 2009 The Go Authors.; Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met (BSD 3-Clause)
- HYGON 修改：无（按上游原样引入）

### k8s.io/api

- 项目/仓库：https://github.com/kubernetes/api
- 固定版本：v0.29.3
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/api
- 版权声明：Copyright The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/apimachinery

- 项目/仓库：https://github.com/kubernetes/apimachinery
- 固定版本：v0.29.3
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/apimachinery
- 版权声明：Copyright The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/client-go

- 项目/仓库：https://github.com/kubernetes/client-go
- 固定版本：v0.29.3
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/client-go
- 版权声明：Copyright The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/klog/v2

- 项目/仓库：https://github.com/kubernetes/klog
- 固定版本：v2.120.1
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/klog/v2
- 版权声明：Copyright 2013 Google Inc. All Rights Reserved.; Copyright 2014 The Kubernetes Authors.; Copyright 2020 Intel Coporation.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/kubelet

- 项目/仓库：https://github.com/kubernetes/kubelet
- 固定版本：v0.29.3
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/kubelet
- 版权声明：Copyright The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

## 间接依赖

### github.com/cpuguy83/go-md2man/v2

- 项目/仓库：https://github.com/cpuguy83/go-md2man
- 固定版本：v2.0.4
- 许可证：MIT
- 本地路径：vendor/github.com/cpuguy83/go-md2man/v2
- 版权声明：Copyright (c) 2014 Brian Goff; The MIT License (MIT)
- HYGON 修改：无（按上游原样引入）

### github.com/davecgh/go-spew

- 项目/仓库：https://github.com/davecgh/go-spew
- 固定版本：v1.1.1
- 许可证：ISC
- 本地路径：vendor/github.com/davecgh/go-spew
- 版权声明：Copyright (c) 2012-2016 Dave Collins <dave@davec.name>; ISC License
- HYGON 修改：无（按上游原样引入）

### github.com/emicklei/go-restful/v3

- 项目/仓库：https://github.com/emicklei/go-restful
- 固定版本：v3.11.3
- 许可证：MIT
- 本地路径：vendor/github.com/emicklei/go-restful/v3
- 版权声明：Copyright (c) 2012,2013 Ernest Micklei; MIT License
- HYGON 修改：无（按上游原样引入）

### github.com/fsnotify/fsnotify

- 项目/仓库：https://github.com/fsnotify/fsnotify
- 固定版本：v1.7.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/fsnotify/fsnotify
- 版权声明：Copyright © 2012 The Go Authors. All rights reserved.; Copyright © fsnotify Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/go-logr/logr

- 项目/仓库：https://github.com/go-logr/logr
- 固定版本：v1.4.1
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/go-logr/logr
- 版权声明：Copyright 2019-2022 The logr Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/go-openapi/jsonpointer

- 项目/仓库：https://github.com/go-openapi/jsonpointer
- 固定版本：v0.21.0
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/go-openapi/jsonpointer
- 版权声明：Copyright 2013 sigu-399 ( https://github.com/sigu-399 ); Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/go-openapi/jsonreference

- 项目/仓库：https://github.com/go-openapi/jsonreference
- 固定版本：v0.21.0
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/go-openapi/jsonreference
- 版权声明：Copyright 2013 sigu-399 ( https://github.com/sigu-399 ); Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/go-openapi/swag

- 项目/仓库：https://github.com/go-openapi/swag
- 固定版本：v0.23.0
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/go-openapi/swag
- 版权声明：Copyright 2015 go-swagger maintainers; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/gogo/protobuf

- 项目/仓库：https://github.com/gogo/protobuf
- 固定版本：v1.3.2
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/gogo/protobuf
- 版权声明：Copyright (c) 2013, The GoGo Authors. All rights reserved.; Copyright 2010 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/golang/glog

- 项目/仓库：https://github.com/golang/glog
- 固定版本：v1.2.2
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/golang/glog
- 版权声明：Copyright 2023 Google Inc. All Rights Reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/golang/protobuf

- 项目/仓库：https://github.com/golang/protobuf
- 固定版本：v1.5.4
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/golang/protobuf
- 版权声明：Copyright 2010 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/google/gnostic-models

- 项目/仓库：https://github.com/google/gnostic-models
- 固定版本：v0.6.8
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/google/gnostic-models
- 版权声明：Copyright 2017-2020 Google LLC. All Rights Reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/google/go-cmp

- 项目/仓库：https://github.com/google/go-cmp
- 固定版本：v0.6.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/google/go-cmp
- 版权声明：Copyright (c) 2017 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/google/gofuzz

- 项目/仓库：https://github.com/google/gofuzz
- 固定版本：v1.2.0
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/google/gofuzz
- 版权声明：Copyright 2014 Google Inc. All rights reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### github.com/google/uuid

- 项目/仓库：https://github.com/google/uuid
- 固定版本：v1.6.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/google/uuid
- 版权声明：Copyright (c) 2009,2014 Google Inc. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/imdario/mergo

- 项目/仓库：https://github.com/imdario/mergo
- 固定版本：v0.3.16
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/imdario/mergo
- 版权声明：Copyright (c) 2013 Dario Castañé. All rights reserved.; Copyright (c) 2012 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/josharian/intern

- 项目/仓库：https://github.com/josharian/intern
- 固定版本：v1.0.0
- 许可证：MIT
- 本地路径：vendor/github.com/josharian/intern
- 版权声明：Copyright (c) 2019 Josh Bleecher Snyder; MIT License
- HYGON 修改：无（按上游原样引入）

### github.com/json-iterator/go

- 项目/仓库：https://github.com/json-iterator/go
- 固定版本：v1.1.12
- 许可证：MIT
- 本地路径：vendor/github.com/json-iterator/go
- 版权声明：Copyright (c) 2016 json-iterator; MIT License
- HYGON 修改：无（按上游原样引入）

### github.com/mailru/easyjson

- 项目/仓库：https://github.com/mailru/easyjson
- 固定版本：v0.7.7
- 许可证：MIT
- 本地路径：vendor/github.com/mailru/easyjson
- 版权声明：Copyright (c) 2016 Mail.Ru Group; MIT License
- HYGON 修改：无（按上游原样引入）

### github.com/modern-go/concurrent

- 项目/仓库：https://github.com/modern-go/concurrent
- 固定版本：v0.0.0-20180306012644-bacd9c7ef1dd
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/modern-go/concurrent
- 版权声明：Licensed under the Apache License, Version 2.0（上游 LICENSE 与源码文件头均未标注具体版权人，版权归 modern-go 项目贡献者所有）
- HYGON 修改：无（按上游原样引入）

### github.com/modern-go/reflect2

- 项目/仓库：https://github.com/modern-go/reflect2
- 固定版本：v1.0.2
- 许可证：Apache-2.0
- 本地路径：vendor/github.com/modern-go/reflect2
- 版权声明：Licensed under the Apache License, Version 2.0（上游 LICENSE 与源码文件头均未标注具体版权人，版权归 modern-go 项目贡献者所有）
- HYGON 修改：无（按上游原样引入）

### github.com/munnerz/goautoneg

- 项目/仓库：https://github.com/munnerz/goautoneg
- 固定版本：v0.0.0-20191010083416-a7dc8b61c822
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/munnerz/goautoneg
- 版权声明：Copyright (c) 2011, Open Knowledge Foundation Ltd. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/russross/blackfriday/v2

- 项目/仓库：https://github.com/russross/blackfriday
- 固定版本：v2.1.0
- 许可证：BSD-2-Clause
- 本地路径：vendor/github.com/russross/blackfriday/v2
- 版权声明：Copyright © 2011 Russ Ross. All rights reserved.; Distributed under the Simplified BSD License
- HYGON 修改：无（按上游原样引入）

### github.com/spf13/pflag

- 项目/仓库：https://github.com/spf13/pflag
- 固定版本：v1.0.5
- 许可证：BSD-3-Clause
- 本地路径：vendor/github.com/spf13/pflag
- 版权声明：Copyright (c) 2012 Alex Ogier. All rights reserved.; Copyright (c) 2012 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### github.com/xrash/smetrics

- 项目/仓库：https://github.com/xrash/smetrics
- 固定版本：v0.0.0-20201216005158-039620a65673
- 许可证：MIT
- 本地路径：vendor/github.com/xrash/smetrics
- 版权声明：Copyright (C) 2016 Felipe da Cunha Gonçalves. All Rights Reserved.; MIT LICENSE
- HYGON 修改：无（按上游原样引入）

### go.etcd.io/bbolt

- 项目/仓库：https://github.com/etcd-io/bbolt
- 固定版本：v1.3.11
- 许可证：MIT
- 本地路径：vendor/go.etcd.io/bbolt
- 版权声明：Copyright (c) 2013 Ben Johnson; The MIT License (MIT)
- HYGON 修改：无（按上游原样引入）

### golang.org/x/oauth2

- 项目/仓库：https://github.com/golang/oauth2
- 固定版本：v0.17.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/oauth2
- 版权声明：Copyright (c) 2009 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### golang.org/x/sys

- 项目/仓库：https://github.com/golang/sys
- 固定版本：v0.24.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/sys
- 版权声明：Copyright 2009 The Go Authors.
- HYGON 修改：无（按上游原样引入）

### golang.org/x/term

- 项目/仓库：https://github.com/golang/term
- 固定版本：v0.23.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/term
- 版权声明：Copyright 2009 The Go Authors.
- HYGON 修改：无（按上游原样引入）

### golang.org/x/text

- 项目/仓库：https://github.com/golang/text
- 固定版本：v0.17.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/text
- 版权声明：Copyright 2009 The Go Authors.
- HYGON 修改：无（按上游原样引入）

### golang.org/x/time

- 项目/仓库：https://github.com/golang/time
- 固定版本：v0.5.0
- 许可证：BSD-3-Clause
- 本地路径：vendor/golang.org/x/time
- 版权声明：Copyright (c) 2009 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### google.golang.org/appengine

- 项目/仓库：https://github.com/golang/appengine
- 固定版本：v1.6.8
- 许可证：Apache-2.0
- 本地路径：vendor/google.golang.org/appengine
- 版权声明：Copyright 2011-2012 Google Inc. All rights reserved.; Copyright 2011 The Go Authors. All rights reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### google.golang.org/genproto/googleapis/rpc

- 项目/仓库：https://github.com/googleapis/go-genproto
- 固定版本：v0.0.0-20240227224415-6ceb2ff114de
- 许可证：Apache-2.0
- 本地路径：vendor/google.golang.org/genproto/googleapis/rpc
- 版权声明：Copyright 2022-2023 Google LLC; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### google.golang.org/grpc

- 项目/仓库：https://github.com/grpc/grpc-go
- 固定版本：v1.63.2
- 许可证：Apache-2.0
- 本地路径：vendor/google.golang.org/grpc
- 版权声明：Copyright 2014 gRPC authors.; Copyright 2015 The gRPC Authors; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### google.golang.org/protobuf

- 项目/仓库：https://github.com/protocolbuffers/protobuf-go
- 固定版本：v1.34.2
- 许可证：BSD-3-Clause
- 本地路径：vendor/google.golang.org/protobuf
- 版权声明：Copyright (c) 2018 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### gopkg.in/inf.v0

- 项目/仓库：https://github.com/go-inf/inf
- 固定版本：v0.9.1
- 许可证：BSD-3-Clause
- 本地路径：vendor/gopkg.in/inf.v0
- 版权声明：Copyright (c) 2012 Péter Surányi. Portions Copyright (c) 2009 The Go Authors. All rights reserved.
- HYGON 修改：无（按上游原样引入）

### gopkg.in/yaml.v2

- 项目/仓库：https://github.com/go-yaml/yaml （分支 v2）
- 固定版本：v2.4.0
- 许可证：Apache-2.0 AND MIT（`apic.go`、`emitterc.go` 等自 libyaml 移植的文件适用 MIT，见 `LICENSE.libyaml`）
- 本地路径：vendor/gopkg.in/yaml.v2
- 版权声明：Copyright 2011-2016 Canonical Ltd.; Copyright (c) 2006 Kirill Simonov; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### gopkg.in/yaml.v3

- 项目/仓库：https://github.com/go-yaml/yaml （分支 v3）
- 固定版本：v3.0.1
- 许可证：MIT AND Apache-2.0（自 libyaml 移植的文件适用 MIT，其余适用 Apache-2.0）
- 本地路径：vendor/gopkg.in/yaml.v3
- 版权声明：Copyright 2011-2016 Canonical Ltd.; Copyright (c) 2006-2010 Kirill Simonov; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/kube-openapi

- 项目/仓库：https://github.com/kubernetes/kube-openapi
- 固定版本：v0.0.0-20240227032403-f107216b40e2
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/kube-openapi
- 版权声明：Copyright The Kubernetes Authors.; Copyright 2020 The Go Authors. All rights reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### k8s.io/utils

- 项目/仓库：https://github.com/kubernetes/utils
- 固定版本：v0.0.0-20240102154912-e7106e64919e
- 许可证：Apache-2.0
- 本地路径：vendor/k8s.io/utils
- 版权声明：Copyright The Kubernetes Authors.; Copyright 2009-2010 The Go Authors. All rights reserved.; Copyright 2013 Google Inc.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### sigs.k8s.io/json

- 项目/仓库：https://github.com/kubernetes-sigs/json
- 固定版本：v0.0.0-20221116044647-bc3834ca7abd
- 许可证：Apache-2.0 AND BSD-3-Clause（`internal/golang/*` 为 Go 标准库移植代码，适用 BSD-3-Clause）
- 本地路径：vendor/sigs.k8s.io/json
- 版权声明：Copyright The Kubernetes Authors.; Copyright 2010-2016 The Go Authors. All rights reserved.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### sigs.k8s.io/structured-merge-diff/v4

- 项目/仓库：https://github.com/kubernetes-sigs/structured-merge-diff
- 固定版本：v4.4.1
- 许可证：Apache-2.0
- 本地路径：vendor/sigs.k8s.io/structured-merge-diff/v4
- 版权声明：Copyright 2018-2020 The Kubernetes Authors.; Licensed under the Apache License, Version 2.0
- HYGON 修改：无（按上游原样引入）

### sigs.k8s.io/yaml

- 项目/仓库：https://github.com/kubernetes-sigs/yaml
- 固定版本：v1.4.0
- 许可证：MIT AND Apache-2.0（内置 `goyaml.v2` / `goyaml.v3` 子目录沿用上游 Apache-2.0 与 MIT 双许可）
- 本地路径：vendor/sigs.k8s.io/yaml
- 版权声明：Copyright (c) 2014 Sam Ghods; Copyright 2011-2016 Canonical Ltd.; The MIT License (MIT)
- HYGON 修改：无（按上游原样引入）

## 许可证汇总

| 许可证 | 依赖数量 | 说明 |
|--------|----------|------|
| Apache-2.0 | 24 | 与本项目许可证一致，无兼容性问题 |
| BSD-3-Clause | 16 | 宽松许可，需保留版权声明与免责声明 |
| MIT | 11 | 宽松许可，需保留版权声明与许可声明 |
| BSD-2-Clause | 1 | `github.com/russross/blackfriday/v2` |
| ISC | 1 | `github.com/davecgh/go-spew` |

说明：`gopkg.in/yaml.v2`、`gopkg.in/yaml.v3`、`sigs.k8s.io/json`、`sigs.k8s.io/yaml` 为多许可证组合，在上表中按其主许可证归类计数。

所有依赖均使用宽松型开源许可证（Apache-2.0 / BSD / MIT / ISC），不含 GPL、LGPL、AGPL 等 Copyleft 许可证，
与本项目采用的 Apache License 2.0 兼容，满足二进制分发与容器镜像分发的合规要求。

## 声明与更新

- 本文件随 `go.mod` 变更同步更新；新增或升级依赖时，须补充/修订对应条目
- 若后续对任一第三方组件进行源码修改，须在该条目「HYGON 修改」中说明修改内容、原因及范围，并在被修改文件的文件头保留上游版权声明
- 各依赖完整的许可证正文以其发行包内的 `LICENSE` / `NOTICE` 文件为准；执行 `go mod vendor` 后可在 `vendor/` 对应目录下查阅
