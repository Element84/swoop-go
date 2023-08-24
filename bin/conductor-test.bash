#!/usr/bin/env bash

set -euo pipefail

RUN_CLEANUP="true"
VERBOSE="false"
CONDUCTOR_START_TIME_SECS=2
CONDUCTOR_TIME_LIMIT_SECS=4

TESTNAME="swoop-test-conductor-e2e"

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


conductor() {
    run_swoop_cmd \
        $((CONDUCTOR_TIME_LIMIT_SECS+CONDUCTOR_START_TIME_SECS)) \
        conductor run instance-a -f "${CONFIG}"
}


main() {
    CLEANUP=( "rmtestdb" "rmtestns")
    trap cleanup EXIT

    # init
    mktestdb
    mktestns
    gobuild

    # pre-run
    #   test cases:
    uuid1='018734f6-c400-74a1-b826-f261c41f3861'
    uuid2='018734f6-c400-77db-a1dd-e245a0fc2c79'
    uuid3='018734f6-c400-715f-9214-879dbe6f73c2'
    uuid4='018734f6-c400-72e2-b908-19e188e7d0c6'

    insert_action "${uuid1}" mirror

    # run
    conductor &
    local pid=$!

    # give caboose a chance to start
    sleep $((CONDUCTOR_START_TIME_SECS))

    insert_action "${uuid2}" mirror
    sleep 1

    insert_action "${uuid3}" badname
    insert_action "${uuid4}" mirror

    # post-run
    wait "${pid}"

    rc=0
    expected1="QUEUED"
    expected2="QUEUED"
    expected3="FAILED"
    expected4="QUEUED"
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

    is_resource_in_cluster "${uuid1}" || {
        echo2 "FAILED: Action 1 resource not found in cluster"
        rc=1
    }
    is_resource_in_cluster "${uuid2}" || {
        echo2 "FAILED: Action 2 resource not found in cluster"
        rc=1
    }
    ! is_resource_in_cluster "${uuid3}" || {
        echo2 "FAILED: Action 3 resource found in cluster but should not exist"
        rc=1
    }

    is_resource_in_cluster "${uuid4}" || {
        echo2 "FAILED: Action 4 resource not found in cluster"
        rc=1
    }

    return ${rc}
}


main "$@"
