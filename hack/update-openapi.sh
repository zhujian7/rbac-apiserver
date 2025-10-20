#!/bin/bash

# Copyright 2024 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
OPENAPI_PKG="github.com/stolostron/rbac-apiserver"

# Install openapi-gen if not present
OPENAPI_GEN="${SCRIPT_ROOT}/_output/tools/bin/openapi-gen"
if [ ! -f "${OPENAPI_GEN}" ]; then
    echo "Installing openapi-gen..."
    mkdir -p "${SCRIPT_ROOT}/_output/tools/bin"

    # Try to get openapi-gen from the latest Kubernetes tools
    go install k8s.io/kube-openapi/cmd/openapi-gen@latest
    cp "$(go env GOPATH)/bin/openapi-gen" "${OPENAPI_GEN}"
fi

# Create output directory
mkdir -p "${SCRIPT_ROOT}/pkg/generated/openapi"

# Generate OpenAPI definitions
echo "Generating OpenAPI definitions..."

"${OPENAPI_GEN}" \
    --output-dir="${SCRIPT_ROOT}/apis/generated/openapi" \
    --output-pkg="${OPENAPI_PKG}/apis/generated/openapi" \
    --go-header-file="${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    --output-file="zz_generated.openapi.go" \
    --report-filename="${SCRIPT_ROOT}/violations.report" \
    -v 2 \
    "${OPENAPI_PKG}/apis/widget/v1alpha1" \
    "${OPENAPI_PKG}/apis/rbac/v1alpha1" \
    "k8s.io/apimachinery/pkg/apis/meta/v1" \
    "k8s.io/apimachinery/pkg/runtime" \
    "k8s.io/apimachinery/pkg/version"

echo "OpenAPI generation completed successfully!"
