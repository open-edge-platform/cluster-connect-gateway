#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

## Modified from Ethan Davidson
## https://stackoverflow.com/questions/71584005/
## how-to-run-multi-fuzz-test-cases-wirtten-in-one-source-file-with-go1-18

# clean all subprocesses on ctl-c
trap "trap - SIGTERM && kill -- -$$ || true" SIGINT SIGTERM

set -e

fuzzTime="${1:-1m}"  # read from argument list or fallback to default - 1 minute
reevalRetries="${REEVAL_RETRIES:-2}"  # retry count when a crashing corpus input is reported

if ! [[ "$reevalRetries" =~ ^[1-9][0-9]*$ ]]; then
    exit 2
fi

files=$(grep -r --include='*_test.go' --files-with-matches 'func Fuzz' internal cmd)

cat <<EOF
Starting fuzzing tests.
    One test timeout: $fuzzTime
    Files:
$files
EOF

go clean --cache

pids=()
for file in ${files}
do
    funcs="$(grep -oP 'func \K(Fuzz\w*)' "$file")"
    for func in ${funcs}
    do
        {
            echo "Fuzzing $func in $file"
            parentDir="$(dirname "$file")"
            mkdir -p fuzz-logs
            fuzzLog="fuzz-logs/${func}.log"
            if go test "./$parentDir" -fuzz="$func" -run="$func" -fuzztime="${fuzzTime}" -v -parallel 1 >"$fuzzLog" 2>&1; then
                cat "$fuzzLog"
                exit 0
            fi

            cat "$fuzzLog"
            if grep -qE 'Failing input written to|found a crashing input' "$fuzzLog"; then
                for _attempt in $(seq 1 "$reevalRetries")
                do
                    if go test "./$parentDir" -run="^${func}$" -count=1 -v; then
                        exit 0
                    fi
                done
                exit 1
            fi

            exit 1
        } &
        pids+=("$!")
        # Throttle: keep at most 14 fuzz jobs running concurrently to avoid OOM
        while [ "${#pids[@]}" -ge 14 ]; do
            running=()
            for pid in "${pids[@]}"; do
                kill -0 "$pid" 2>/dev/null && running+=("$pid")
            done
            pids=("${running[@]+${running[@]}}")
            [ "${#pids[@]}" -ge 14 ] && sleep 1
        done
    done
done

# wait -n waits for ANY job to finish and returns its exit code.
# This gives true fail-fast: a failure in job 5 is caught immediately,
# not after jobs 0-4 have run to completion.
while [ "${#pids[@]}" -gt 0 ]; do
    if ! wait -n "${pids[@]}"; then
        kill "${pids[@]}" 2>/dev/null || true
        exit 1
    fi
    # Prune any PIDs that have now finished from the tracking array
    running=()
    for pid in "${pids[@]}"; do
        kill -0 "$pid" 2>/dev/null && running+=("$pid")
    done
    pids=("${running[@]+${running[@]}}")
done

echo "All fuzz jobs completed successfully."
