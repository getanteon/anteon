# Shell Completion

## ZHS

`completions/_ddosify` provides a basic auto-completions. You can apply one of the steps to get an auto-completion successfully.

- You can locate the file in any directory referenced by `$fpath`.

  - It can be checked out through;

  ```SHELL
  echo $fpath | tr ' ' '\n'
  ```

  - For example, if you are using `oh-my-zsh` you can add it as a plugin after locating file under plugin related directory appeared in `$fpath`.

  ```
  # ~/.zshrc

  plugins=(
    git
    zsh-autosuggestions
    ddosify
    ...
  )
  ```

- If you don't have an appropriate directory,
  - It can be generated through;
  ```
  mkdir -p ${ZDOTDIR:-~}/.zsh_functions
  echo 'fpath+=${ZDOTDIR:-~}/.zsh_functions' >> ${ZDOTDIR:-~}/.zshrc
  ```
  - Then, the file should be copied to this directory;
  ```
  cp completions/_ddosify ${ZDOTDIR:-~}/.zsh_functions/_ddosify
  ```
