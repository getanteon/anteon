# Shell completions

## Zsh

`completions/_ddosify` provides a basic auto-completions. You can apply one of the steps to get an auto-completion successfully. 

You can locate the file in any directory referenced by `$fpath`. You can use the following command to list directories in `$fpath`.

```bash
echo $fpath | tr ' ' '\n'
```

For example, if you are using [oh-my-zsh](https://ohmyz.sh/) you can add it as a plugin after locating file under plugin related directory appeared in `$fpath`. You can create a directory named `ddosify` under `~/.oh-my-zsh/plugins` and copy `_ddosify` file to it.

```bash
mkdir -p ~/.oh-my-zsh/plugins/ddosify
cp completions/_ddosify ~/.oh-my-zsh/plugins/ddosify
```

Then, you can add `ddosify` to your plugins list in `~/.zshrc` file.

```
# ~/.zshrc

plugins=(
  ...
  ddosify
)
```

If you don't have an appropriate directory, you can create one and add it to `$fpath`.
  
```
mkdir -p ${ZDOTDIR:-~}/.zsh_functions
echo 'fpath+=${ZDOTDIR:-~}/.zsh_functions' >> ${ZDOTDIR:-~}/.zshrc
```

Then, you can copy `_ddosify` file to the directory you created.

```
cp completions/_ddosify ${ZDOTDIR:-~}/.zsh_functions/_ddosify
```
