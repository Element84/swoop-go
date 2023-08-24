#!/usr/bin/env bash

set -euo pipefail

ROOT="${THIS_DIR}/.."

ENV="${ROOT}/.env"

set -a
. .env
set +a

export PGDATABASE="${TESTNAME}"
export SWOOP_S3_BUCKET="${TESTNAME}"
export AWS_ACCESS_KEY_ID="${MINIO_ACCESS_KEY}"
export AWS_SECRET_ACCESS_KEY="${MINIO_SECRET_KEY}"
export KUBECONFIG="${ROOT}/kubeconfig.yaml"

FIXTURES="${ROOT}/fixtures"
PAYLOADS="${FIXTURES}/payloads"
RESOURCES="${FIXTURES}/resource"
TEMPLATES="${FIXTURES}/workflow-templates"

CONFIG="${FIXTURES}/swoop-config.yml"

# general helpers
echo2 () {
    local msg="${1?"message string required"}"
    echo -e >&2 "${msg:-}"
}


fatal () {
    local msg="${1?"message string required"}"; shift
    local rc=${1:-1}
    echo2 "${msg:-}"
    exit "$rc"
}


CLEANUP=()
cleanup() {
    [ "${RUN_CLEANUP}" == "true" ] || { echo2 "Skipping cleanup" && return ; }
    for fn in "${CLEANUP[@]}"; do
        ${fn} || echo2 "Failed to run cleanup function '${fn}'"
    done
}


# command overrides
aws() {
    AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
        AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
        command aws --endpoint "${SWOOP_S3_ENDPOINT}" "${@}"
}


pgexec() {
    docker compose exec -e PGDATABASE="${PGDATABASE}" -T postgres "${@}"
}


swoop_db() {
    pgexec swoop-db "${@}"
}


psql() {
    pgexec psql -Aqt "${@}"
}


# init/cleanup functions
mktestdb() {
    swoop_db up 1>&3 2>&3 \
        || fatal "Failed to create/migrate database"
    swoop_db load-fixture "base_01" 1>&3 2>&3 \
        || fatal "Failed to load base database fixture"
}


rmtestdb() {
    swoop_db drop 1>&3 2>&3
}


mktestbucket() {
    aws s3api create-bucket --bucket "${TESTNAME}" 1>&3 2>&3
}


rmtestbucket() {
    aws s3 rm "s3://${TESTNAME}" --recursive 1>&3 2>&3
    aws s3api delete-bucket --bucket "${TESTNAME}"
}


mktestns() {
    KUBECONFIG="${KUBECONFIG}" kubectl create namespace "${TESTNAME}" 1>&3 2>&3
    apply_workflow_templates
}


rmtestns() {
    # note that running this script in quick succession will
    # fail because the namespace will still be deleting
    KUBECONFIG="${KUBECONFIG}" kubectl delete namespace "${TESTNAME}" --wait="false" 1>&3 2>&3
}


# go-related functions
gobuild() {
    go build -C "${ROOT}" -o swoop
}


run_swoop_cmd() {
    local timelimit="${1:?"provide the time limit for the execution in seconds"}"
    shift
    local pid
    (
        cd "${ROOT}"
        export KUBECONFIG
        ./swoop "${@}" 1>&3 2>&3 &
        pid=$!
        sleep "${timelimit}"
        kill "${pid}"
    )
}


# testing helpers
insert_action() {
    local uuid="${1:?"must provide an action_uuid"}"
    local action_name="${2:?"must provide an action_name"}"
    psql <<EOF
        insert into swoop.action (
            action_uuid,
            created_at,
            action_type,
            action_name,
            handler_name,
            handler_type,
            payload_uuid,
            workflow_version
        ) values (
            '${uuid}',
            '2023-03-31'::timestamp,
            'workflow',
            '${action_name}',
            'argoHandler',
            'argoWorkflow',
            'ade69fe7-1d7d-572e-9f36-7242cc2aca77',
            1
        );
EOF
}


check_action_status() {
    local uuid="${1?"must provide an action_uuid"}"
    psql -c "select status from swoop.thread where action_uuid = '${uuid}';"
}


get_callback_uuid() {
    local uuid="${1?"must provide an action_uuid"}"
    psql -c "select action_uuid from swoop.action where parent_uuid = '${uuid}';"
}


stage_action_input() {
    local uuid="${1?"must provide an action_uuid"}"
    aws s3 cp "${PAYLOADS}/input.json" "s3://${TESTNAME}/executions/${uuid}/input.json" 1>&3 2>&3
}


stage_action_output() {
    local uuid="${1?"must provide an action_uuid"}"
    aws s3 cp "${PAYLOADS}/output.json" "s3://${TESTNAME}/executions/${uuid}/output.json" 1>&3 2>&3
}


has_callback_params() {
    local uuid="${1?"must provide an action_uuid"}"
    aws s3 ls "s3://${TESTNAME}/callbacks/${uuid}/parameters.json" 1>&3 2>&3
}


apply_workflow_templates() {
    KUBECONFIG="${KUBECONFIG}" kubectl -n "${TESTNAME}" apply -Rf "${TEMPLATES}" 1>&3 2>&3
}


apply_workflow_resource() {
    local uuid="${1?"must provide an action_uuid"}"
    local state="${2?"must provide workflow state (init, started, successful, failed)"}"
    <"${RESOURCES}/${state}.json" sed "s/\${UUID}/${uuid}/" \
        | KUBECONFIG="${KUBECONFIG}" kubectl -n "${TESTNAME}" apply -f - 1>&3 2>&3
}

is_resource_in_cluster() {
    local uuid="${1?"must provide an action_uuid"}"
    KUBECONFIG="${KUBECONFIG}" kubectl -n "${TESTNAME}" get workflows "${uuid}" 1>&3 2>&3
}
