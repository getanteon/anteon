#!/bin/sh
# Downloads the reference tokenizers at their pinned revisions and checks their sha256.
# Usage: scripts/fetch-tokenizers.sh DIR, then DDOSIFY_TOKENIZERS_DIR=DIR go test ./core/llm/tokenizer
set -eu
dir=${1:?usage: fetch-tokenizers.sh DIR}
list=$(dirname "$0")/../core/llm/tokenizer/testdata/tokenizers.txt
mkdir -p "$dir"
grep -v '^#' "$list" | while read -r model rev sum; do
	[ -n "$model" ] || continue
	out="$dir/$(echo "$model" | tr / _).json"
	curl -fsSL --retry 3 -o "$out" "https://huggingface.co/$model/resolve/$rev/tokenizer.json"
	got=$( (sha256sum "$out" 2>/dev/null || shasum -a 256 "$out") | cut -d' ' -f1)
	if [ "$got" != "$sum" ]; then
		echo "$model@$rev: sha256 $got, expected $sum" >&2
		exit 1
	fi
	echo "$model@$rev ok"
done
