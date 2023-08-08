#!/usr/bin/env bash

set -euxo pipefail

RUN_CLEANUP="true"
CABOOSE_START_TIME_SECS=2
CABOOSE_TIME_LIMIT_SECS=4


find_this () {
    THIS="${1:?'must provide script path, like "${BASH_SOURCE[0]}" or "$0"'}"
    trap "fatal 'FATAL: could not resolve parent directory of ${THIS}'" EXIT
    [ "${THIS:0:1}"  == "/" ] || THIS="$(pwd -P)/${THIS}"
    THIS_DIR="$(dirname -- "${THIS}")"
    THIS_DIR="$(cd -P -- "${THIS_DIR}" && pwd)"
    THIS="${THIS_DIR}/$(basename -- "${THIS}")"
    trap "" EXIT
}

find_this "${BASH_SOURCE[0]}"
ROOT="${THIS_DIR}/.."

ENV="${ROOT}/.env"

set -a
. .env
set +a

TESTNAME="swoop-test-caboose-e2e"

export PGDATABASE="${TESTNAME}"
export SWOOP_S3_BUCKET="${TESTNAME}"
export AWS_ACCESS_KEY_ID="${MINIO_ACCESS_KEY}"
export AWS_SECRET_ACCESS_KEY="${MINIO_SECRET_KEY}"
export KUBECONFIG="${ROOT}/kubeconfig.yaml"

FIXTURES="${ROOT}/fixtures"
PAYLOADS="${FIXTURES}/payloads"
RESOURCES="${FIXTURES}/resource"

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
    swoop_db up \
        || fatal "Failed to create/migrate database"
    swoop_db load-fixture "base_01" \
        || fatal "Failed to load base database fixture"
}


rmtestdb() {
    swoop_db drop
}


mktestbucket() {
    aws s3api create-bucket --bucket "${TESTNAME}"
}


rmtestbucket() {
    aws s3 rm "s3://${TESTNAME}" --recursive
    aws s3api delete-bucket --bucket "${TESTNAME}"
}


mktestns() {
    kubectl create namespace "${TESTNAME}"
}


rmtestns() {
    # note that running this script in quick succession will
    # fail because the namespace will still be deleting
    kubectl delete namespace "${TESTNAME}" --wait="false"
}


# caboose-related functions
build_caboose() {
    go build -C "${ROOT}" -o swoop
}


caboose() {
    local pid
    (
        cd "${ROOT}"
        ./swoop caboose argo -f "${CONFIG}" --namespace ${TESTNAME} &
        pid=$!
        sleep $((CABOOSE_TIME_LIMIT_SECS + CABOOSE_START_TIME_SECS))
        kill "${pid}"
    )
}


# testing helpers
insert_action() {
    local uuid="${1?"must provide an action_uuid"}"
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
            'mirror',
            'argo-handler',
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
    aws s3 cp "${PAYLOADS}/input.json" "s3://${TESTNAME}/executions/${uuid}/input.json"
}


stage_action_output() {
    local uuid="${1?"must provide an action_uuid"}"
    aws s3 cp "${PAYLOADS}/output.json" "s3://${TESTNAME}/executions/${uuid}/output.json"
}


has_callback_params() {
    local uuid="${1?"must provide an action_uuid"}"
    aws s3 ls "s3://${TESTNAME}/callbacks/${uuid}/parameters.json" >/dev/null 2>&1
}


apply_workflow_resource() {
    local uuid="${1?"must provide an action_uuid"}"
    local state="${2?"must provide workflow state (init, started, successful, failed)"}"
    <"${RESOURCES}/${state}.json" sed "s/\${UUID}/${uuid}/" | kubectl -n "${TESTNAME}" apply -f -
}

is_resource_in_cluster() {
    local uuid="${1?"must provide an action_uuid"}"
    kubectl -n "${TESTNAME}" get workflows "${uuid}" >/dev/null 2>&1
}


main() {
    CLEANUP=( "rmtestdb" "rmtestbucket" "rmtestns")
    trap cleanup EXIT

    # init
    mktestdb
    mktestbucket
    mktestns
    build_caboose

    # pre-run
    #   test cases:
    #     1) successful before start
    uuid1='018734f6-c400-74a1-b826-f261c41f3861'
    #     2) start before, succussful during
    uuid2='018734f6-c400-77db-a1dd-e245a0fc2c79'
    #     3) start during, fail during
    uuid3='018734f6-c400-715f-9214-879dbe6f73c2'
    #     4) init during only (should still be in cluster at end)
    uuid4='018734f6-c400-72e2-b908-19e188e7d0c6'

    insert_action "${uuid1}"
    insert_action "${uuid2}"
    insert_action "${uuid3}"
    insert_action "${uuid4}"

    stage_action_input "${uuid1}"
    stage_action_input "${uuid2}"
    stage_action_input "${uuid3}"

    stage_action_output "${uuid1}"
    stage_action_output "${uuid2}"

    apply_workflow_resource "${uuid1}" successful
    apply_workflow_resource "${uuid2}" started

    # run
    caboose &
    local pid=$!

    # give caboose a chance to start
    sleep $((CABOOSE_START_TIME_SECS + 1))

    apply_workflow_resource "${uuid3}" failed
    apply_workflow_resource "${uuid4}" init
    sleep 1
    apply_workflow_resource "${uuid2}" successful

    # post-run
    wait "${pid}"

    rc=0
    expected1="SUCCESSFUL"
    expected2="SUCCESSFUL"
    expected3="FAILED"
    expected4="PENDING"
    status1="$(check_action_status "${uuid1}")"
    status2="$(check_action_status "${uuid2}")"
    status3="$(check_action_status "${uuid3}")"
    status4="$(check_action_status "${uuid4}")"

    [ "${status1}" == "${expected1}" ] || {
        echo2 "Action 1 status not equal to expected: ${status1} != ${expected1}"
        rc=1
    }
    [ "${status2}" == "${expected2}" ] || {
        echo2 "Action 2 status not equal to expected: ${status2} != ${expected2}"
        rc=1
    }
    [ "${status3}" == "${expected3}" ] || {
        echo2 "Action 3 status not equal to expected: ${status3} != ${expected3}"
        rc=1
    }
    [ "${status4}" == "${expected4}" ] || {
        echo2 "Action 4 status not equal to expected: ${status4} != ${expected4}"
        rc=1
    }

    callback1="$(get_callback_uuid "${uuid1}")"
    callback2="$(get_callback_uuid "${uuid2}")"
    callback3="$(get_callback_uuid "${uuid3}")"
    callback4="$(get_callback_uuid "${uuid4}")"

    [ "${callback4}" == "" ] || {
        echo2 "Action 4 should not have a callback; uuid: '${callback4}'"
        rc=1
    }

    has_callback_params "${callback1}" || {
        echo2 "Action 1 callback parameters not found; uuid: '${callback1}'"
        rc=1
    }
    has_callback_params "${callback2}" || {
        echo2 "Action 2 callback parameters not found; uuid: '${callback2}'"
        rc=1
    }
    has_callback_params "${callback3}" || {
        echo2 "Action 3 callback parameters not found; uuid: '${callback3}'"
        rc=1
    }

    ! is_resource_in_cluster "${uuid1}" || {
        echo2 "Action 1 resource not deleted from cluster"
        rc=1
    }
    ! is_resource_in_cluster "${uuid2}" || {
        echo2 "Action 2 resource not deleted from cluster"
        rc=1
    }
    ! is_resource_in_cluster "${uuid3}" || {
        echo2 "Action 3 resource not deleted from cluster"
        rc=1
    }

    is_resource_in_cluster "${uuid4}" || {
        echo2 "Action 4 resource not found in cluster"
        rc=1
    }

    return ${rc}
}


main "$@"
