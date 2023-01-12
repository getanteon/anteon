export ZSH=$HOME/.oh-my-zsh

ZSH_THEME="cloud"

plugins=(
    git
    zsh-autosuggestions
)

source $ZSH/oh-my-zsh.sh

source /usr/share/doc/fzf/examples/key-bindings.zsh
source /usr/share/doc/fzf/examples/completion.zsh

alias ll='ls -alF'
alias hammer-clean='go clean -testcache'
alias hammer-test-n-cover='gotest -coverpkg=./... -coverprofile=coverage.out ./... && go tool cover -func coverage.out'

export PATH="$PATH:/go/bin"
