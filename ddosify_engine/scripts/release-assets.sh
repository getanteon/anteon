#!/bin/sh
# Builds what the release archives carry besides the binary: the licence, shell completions and
# man pages. Run it before goreleaser, which runs no hook of its own and so never hands a token to
# go run. Usage, in ddosify_engine: scripts/release-assets.sh
set -eu
go mod verify
rm -rf build
mkdir -p build/completions build/manpages
cp ../LICENSE build/LICENSE
go run . completion bash > build/completions/ddosify.bash
go run . completion zsh > build/completions/_ddosify
go run . completion fish > build/completions/ddosify.fish
go run . docs man --dir build/manpages
