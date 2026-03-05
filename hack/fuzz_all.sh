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

for file in ${files}
do
    funcs="$(grep -oP 'func \K(Fuzz\w*)' "$file")"
    for func in ${funcs}
    do
        {
            echo "Fuzzing $func in $file"
            parentDir="$(dirname "$file")"
            go test "./$parentDir" -fuzz="$func" -run="$func" -fuzztime="${fuzzTime}" -v -parallel 4
        } &
    done
done

for job in `jobs -p`
do
    echo "Waiting for PID $job to finish"
    wait $job
done
