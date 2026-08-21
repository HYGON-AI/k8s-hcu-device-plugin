# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Hygon Information Technology Co., Ltd.

FROM ubuntu:22.04 AS dp

WORKDIR /root

RUN  apt update && apt install -y kmod pciutils

COPY LICENSE /licenses/

COPY k8s-device-plugin .

#COPY internal/pkg/shim/lib ./lib

RUN chmod +x /root/k8s-device-plugin
#    && ln -s /root/lib/librocm_smi64.so.2.8 /root/lib/librocm_smi64.so.2 \
#    && ln -s /root/lib/librocm_smi64.so.2 /root/lib/librocm_smi64.so

ENV LD_LIBRARY_PATH=/root/lib:/opt/hyhal/lib:$LD_LIBRARY_PATH

CMD ["./k8s-device-plugin"]
