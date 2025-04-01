#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

source "$(dirname "$0")/setup.sh"

export KIND_CLUSTER="${KIND_CLUSTER:-"local-e2e"}"
export KIND_K8S_VERSION="${KIND_K8S_VERSION:-"v1.31.0"}"
export NAMESPACE="${NAMESPACE:-"e2e-test"}"

export PROMETHEUS_INSTALL_SKIP=true
export CERT_MANAGER_INSTALL_SKIP=false
export CLUSTER_API_INSTALL_SKIP=false

install_kind
create_cluster ${KIND_K8S_VERSION}
if [ -z "${SKIP_KIND_CLEANUP:-}" ]; then
  trap delete_cluster EXIT
fi

kubeconfig=$(cd "$(dirname "$0")" && pwd)/kubeconfig
kind export kubeconfig --kubeconfig $kubeconfig --name $KIND_CLUSTER

KUBECONFIG=$kubeconfig test_e2e -v -ginkgo.v
