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
    TEST_SERVER_PORT="${TEST_SERVER_PORT}" run_swoop_cmd \
        $((CONDUCTOR_TIME_LIMIT_SECS+CONDUCTOR_START_TIME_SECS)) \
        conductor run -n instance-a -f "${CONFIG}"
}


main() {
    CLEANUP=( "rmtestdb" "rmtestbucket" "rmtestns" )
    trap cleanup EXIT
    local uuid1 uuid2 uuid3 uuid4 cbid1 cbid2 cbid3 cbid4
    local cpid tspid

    # init
    mktestdb
    mktestbucket
    mktestns
    gobuild

    # pre-run
    #   workflow action test cases:
    uuid1='018734f6-c400-74a1-b826-f261c41f3861'
    uuid2='018734f6-c400-77db-a1dd-e245a0fc2c79'
    uuid3='018734f6-c400-715f-9214-879dbe6f73c2'
    uuid4='018734f6-c400-72e2-b908-19e188e7d0c6'

    #  callback action test cases:
    #    good
    cbid1='018734f6-c400-7e81-97ac-427e9ad27cbb'
    #    not handled
    cbid2='018734f6-c400-7e43-bef8-bfcb9a9c7971'
    #    backoff
    cbid3='018734f6-c400-7515-b13b-687b6658e137'
    #    fatal
    cbid4='018734f6-c400-7968-8d12-d7e3976959cb'

    insert_workflow "${uuid1}" mirror

    stage_callback_params "${cbid1}"
    stage_callback_params "${cbid3}"
    stage_callback_params "${cbid4}"

    insert_callback "${cbid1}" testCbHandler synchttp

    # start test http server for callbacks
    test_server $((CONDUCTOR_TIME_LIMIT_SECS+CONDUCTOR_START_TIME_SECS+2)) <<EOF &
{
  "${cbid1}": {"status": 200, "body": ""},
  "${cbid3}": {"status": 400, "body": "timeout"},
  "${cbid4}": {"status": 404, "body": "not found"}
}
EOF
    tspid=$!

    # run
    conductor &
    cpid=$!

    # give caboose a chance to start
    sleep $((CONDUCTOR_START_TIME_SECS))

    insert_workflow "${uuid2}" mirror

    sleep 1
    insert_callback "${cbid2}" unhandled-handler synchttp
    insert_callback "${cbid3}" testCbHandler synchttp
    insert_workflow "${uuid3}" badname
    insert_workflow "${uuid4}" mirror
    insert_callback "${cbid4}" testCbHandler synchttp

    # post-run
    wait "${cpid}"
    wait "${tspid}"

    rc=0
    expected1="QUEUED"
    expected2="QUEUED"
    expected3="FAILED"
    expected4="QUEUED"
    expected5="SUCCESSFUL"
    expected6="PENDING"
    expected7="BACKOFF"
    expected8="FAILED"
    status1="$(check_action_status "${uuid1}")"
    status2="$(check_action_status "${uuid2}")"
    status3="$(check_action_status "${uuid3}")"
    status4="$(check_action_status "${uuid4}")"
    status5="$(check_action_status "${cbid1}")"
    status6="$(check_action_status "${cbid2}")"
    status7="$(check_action_status "${cbid3}")"
    status8="$(check_action_status "${cbid4}")"

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
    [ "${status5}" == "${expected5}" ] || {
        echo2 "FAILED: Action 5 status not equal to expected: ${status5} != ${expected5}"
        rc=1
    }
    [ "${status6}" == "${expected6}" ] || {
        echo2 "FAILED: Action 6 status not equal to expected: ${status6} != ${expected6}"
        rc=1
    }
    [ "${status7}" == "${expected7}" ] || {
        echo2 "FAILED: Action 7 status not equal to expected: ${status7} != ${expected7}"
        rc=1
    }
    [ "${status8}" == "${expected8}" ] || {
        echo2 "FAILED: Action 8 status not equal to expected: ${status8} != ${expected8}"
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
