#!/usr/bin/env bash

set -euo pipefail

RUN_CLEANUP="true"
VERBOSE="true"
CABOOSE_START_TIME_SECS=2
CABOOSE_TIME_LIMIT_SECS=4

TESTNAME="swoop-test-caboose-e2e"

# output controls
if [ "${VERBOSE}" == "true" ]; then
    exec 3>&2
    set -x
else
    exec 3>/dev/null
fi


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
LIB="${THIS_DIR}/test-lib.bash"

. "${LIB}"


caboose() {
    run_swoop_cmd \
        $((CABOOSE_TIME_LIMIT_SECS+CABOOSE_START_TIME_SECS)) \
        caboose argo -f "${CONFIG}" --namespace ${TESTNAME}
}


main() {
    CLEANUP=( "rmtestdb" "rmtestbucket" "rmtestns")
    trap cleanup EXIT

    # init
    mktestdb
    mktestbucket
    mktestns
    gobuild

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
        echo2 "FAILED: Action 1 status not equal to expected: ${status1} != ${expected1}"
        rc=1
    }
    [ "${status2}" == "${expected2}" ] || {
        echo2 "FAILED: Action 2 status not equal to expected: ${status2} != ${expected2}"
        rc=1
    }
    [ "${status3}" == "${expected3}" ] || {
        echo2 "FAILED: Action 3 status not equal to expected: ${status3} != ${expected3}"
        rc=1
    }
    [ "${status4}" == "${expected4}" ] || {
        echo2 "FAILED: Action 4 status not equal to expected: ${status4} != ${expected4}"
        rc=1
    }

    callback1="$(get_callback_uuid "${uuid1}")"
    callback2="$(get_callback_uuid "${uuid2}")"
    callback3="$(get_callback_uuid "${uuid3}")"
    callback4="$(get_callback_uuid "${uuid4}")"

    has_callback_params "${callback1}" || {
        echo2 "FAILED: Action 1 callback parameters not found; uuid: '${callback1}'"
        rc=1
    }
    has_callback_params "${callback2}" || {
        echo2 "FAILED: Action 2 callback parameters not found; uuid: '${callback2}'"
        rc=1
    }
    has_callback_params "${callback3}" || {
        echo2 "FAILED: Action 3 callback parameters not found; uuid: '${callback3}'"
        rc=1
    }

    [ "${callback4}" == "" ] || {
        echo2 "FAILED: Action 4 should not have a callback; uuid: '${callback4}'"
        rc=1
    }

    ! is_resource_in_cluster "${uuid1}" || {
        echo2 "FAILED: Action 1 resource not deleted from cluster"
        rc=1
    }
    ! is_resource_in_cluster "${uuid2}" || {
        echo2 "FAILED: Action 2 resource not deleted from cluster"
        rc=1
    }
    ! is_resource_in_cluster "${uuid3}" || {
        echo2 "FAILED: Action 3 resource not deleted from cluster"
        rc=1
    }

    is_resource_in_cluster "${uuid4}" || {
        echo2 "FAILED: Action 4 resource not found in cluster"
        rc=1
    }

    return ${rc}
}


main "$@"
