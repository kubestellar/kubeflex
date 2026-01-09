#!/usr/bin/env bash

set -euo pipefail
set -x

function waitfor() {
    cmd="$1"
    let count=1
    while true; do
        sleep 5
        if { eval "$cmd" ; } ; then return 0; fi
        let count=count+1
        if (( count > 15 )); then
            echo 'Timeout waiting for `'"$cmd" >&2
            return 1
        fi
    done
}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR"; cd ../..; pwd)

logfile=log-$$
trap "rm $logfile" EXIT

go run "$REPO_ROOT/cmd/watch-objs" -n default -v=4 &> $logfile &


if ! waitfor 'grep -q "Notified of add.*PostCreateHook.*name=\"synthetic-crd\"" '$logfile; then
    cat $logfile
    exit 1
fi


./bin/kflex create cptest --type k8s --chatty-status=false
./bin/kflex ctx

if ! waitfor 'grep -q "Notified of add.*ControlPlane.*name=\"cptest\"" '$logfile; then
    cat $logfile
    exit 1
fi

kubectl delete cp cptest

kill %1
