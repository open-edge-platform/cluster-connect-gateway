#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

# Creates a named kind cluster given a k8s version.
# The KIND_CLUSTER variable defines the cluster name and
# is expected to be defined in the calling environment.
#
# Usage:
#
#   export KIND_CLUSTER=<kind cluster name>
#   create_cluster <k8s version>
function create_cluster {
  echo "Getting kind config..."
  KIND_VERSION=$1
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  : ${1:?"k8s version must be set as arg 1"}
  if ! kind get clusters | grep -q $KIND_CLUSTER ; then
    version_prefix="${KIND_VERSION%.*}"
    kind_config=$(dirname "$0")/kind-config.yaml
    if test -f $(dirname "$0")/kind-config-${version_prefix}.yaml; then
      kind_config=$(dirname "$0")/kind-config-${version_prefix}.yaml
    fi
    echo "Creating cluster..."
    kind create cluster --name $KIND_CLUSTER --retain --wait=1m --config ${kind_config} --image=kindest/node:$1
  fi
}

# Deletes a kind cluster by cluster name.
# The KIND_CLUSTER variable defines the cluster name and
# is expected to be defined in the calling environment.
#
# Usage:
#
#   export KIND_CLUSTER=<kind cluster name>
#   delete_cluster
function delete_cluster {
  : ${KIND_CLUSTER:?"KIND_CLUSTER must be set"}
  kind delete cluster --name $KIND_CLUSTER
}

# Installing kind in a temporal dir if no previously installed to GOBIN.
function install_kind {
  if ! is_installed kind ; then
    header_text "Installing kind to $(go env GOPATH)/bin"

    go install sigs.k8s.io/kind@v$kind_version
  fi
}

# Check if a program is previously installed
function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}

function test_e2e {
  local flags="$@"
  go test $(dirname "$0")/e2e $flags -timeout 30m
}