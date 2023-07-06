#!/bin/sh

set -eu

K8S_TIMEOUT="${K8S_TIMEOUT:-120}"

ARGO_VERSION="${ARGO_VERSION:-"latest"}"
ARGO_RELEASES="https://github.com/argoproj/argo-workflows/releases"
ARGO_MANIFEST="install.yaml"

YAML_SEP='---'

CACHE_DIR="./.argo-manifests"


get_argo_crds() {
    local argo_url resource='' line argo_version="${ARGO_VERSION}" manifest

    if [ "${argo_version}" == "latest" ]; then
        # resolve latest redirect to actual version
        local url="${ARGO_RELEASES}/latest/download/${ARGO_MANIFEST}"
        argo_version="$(basename "$(dirname "$(curl -sw '%{redirect_url}' "${url}")")")"
    fi

    manifest="${CACHE_DIR}/argo-workflows-${argo_version}.yaml"

    if [ -f "${manifest}" ]; then
        cat "${manifest}"
        return
    fi

    argo_url="${ARGO_RELEASES}/download/${argo_version}/${ARGO_MANIFEST}"

    resource=''
    curl -sL "${argo_url}" | while IFS= read -r line; do
        resource="${resource}${line}\n"

        if [ "${line}" == "${YAML_SEP}" ]; then
            echo "${resource}" | grep -q 'CustomResourceDefinition' \
                && printf '%b' "${resource}"
            resource=''
        fi
    done | tee "${manifest}"
}


main() {
    mkdir -p "${CACHE_DIR}"

    # wait for cluster to be responsive
    while [ "${K8S_TIMEOUT}" -gt 0 ]; do
        resp="$(kubectl --server="${SERVER_URL:-}" get pods 2>&1)" && break
        echo "${resp}"
        sleep 1
        K8S_TIMEOUT=$((K8S_TIMEOUT-1))
    done

    get_argo_crds | kubectl --server="${SERVER_URL:-}" apply -f -
}

main "$@"
