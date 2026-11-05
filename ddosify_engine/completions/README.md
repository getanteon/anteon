# Shell completions

Ddosify generates its own completions for bash, zsh, fish and PowerShell:

```bash
ddosify completion bash > /usr/local/etc/bash_completion.d/ddosify   # bash
ddosify completion zsh > "${fpath[1]}/_ddosify"                       # zsh
ddosify completion fish > ~/.config/fish/completions/ddosify.fish     # fish
ddosify completion powershell | Out-String | Invoke-Expression        # PowerShell
```

`ddosify completion <shell> --help` explains each shell's setup. The release archives and the Homebrew formula carry the bash, zsh and fish files and the man pages.
