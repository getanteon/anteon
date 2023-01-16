#!/bin/bash

LIMIT=15
IS_FAILED=0
time_op=$(grep -A1 'time/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo -e "Max. Delta Time op: $time_op / $LIMIT" | tee benchstat.txt
if (( $(echo "$time_op > $LIMIT" | bc -l) )); then
    IS_FAILED=1
fi

alloc_op=$(grep -A1 'alloc/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo -e "Max. Delta Alloc op: $alloc_op / $LIMIT" | tee --append benchstat.txt
if (( $(echo "$alloc_op > $LIMIT" | bc -l) )); then
    IS_FAILED=1
fi

allocs_op=$(grep -A1 'allocs/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo -e "Max. Delta Allocs op: $allocs_op / $LIMIT" | tee --append benchstat.txt
if (( $(echo "$allocs_op > $LIMIT" | bc -l) )); then
    IS_FAILED=1
fi

github_comment=`jq -Rs '.' benchstat.txt`
curl -s -H "Authorization: token $1" \
                -X POST -d "{\"body\": $github_comment" \
                "https://api.github.com/repos/ddosify/ddosify/issues/$2/comments"

if [ $IS_FAILED -eq 1 ]; then
    exit 1
fi
