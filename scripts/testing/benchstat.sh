#!/bin/bash

LIMIT=10
time_op=$(grep -A1 'time/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo "Time op: $time_op / $LIMIT"
if (( $(echo "$time_op > $LIMIT" | bc -l) )); then
    exit 1
fi

alloc_op=$(grep -A1 'alloc/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo "Alloc op: $alloc_op / $LIMIT"
if (( $(echo "$alloc_op > $LIMIT" | bc -l) )); then
    exit 2
fi

allocs_op=$(grep -A1 'allocs/op' gobench_branch_result.txt |tail -1 | awk '{print $8}' | tr -d + | tr -d %)
echo "Allocs op: $allocs_op / $LIMIT"
if (( $(echo "$allocs_op > $LIMIT" | bc -l) )); then
    exit 3
fi
