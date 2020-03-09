fswatcher
===
fawatcher is an useful cross-platform command tool which will trigger the user's specified command when the watched files and pathes change. Thanks for the awesome library written in Go [fsnotify](https://github.com/fsnotify/fsnotify).

### Requirements
Go 1.18 or newer.

### Installation:

```shell
git clone https://github.com/qshuai/fswatcher.git $GOPATH/src/github.com/qshuai/fswatcher
cd GOPATH/src/github.com/qshuai/fswatcher
go mod download
# please ensure the $GOPATH/bin is in your $PATH environment
go install
```

### Usage:

```shell
> fswatcher --help
fswatcher watches the specified files or directories, and any changing event will trigger the user's command

Usage:
  fswatcher [flags]

Examples:
fswatcher -c 'echo ***' /tmp/foo

Flags:
  -c, --command string    the command to execute when change event notified
  -h, --help              help for fswatcher
  -i, --ignore strings    comma separated list of files and paths to ignore
  -v, --interval string   the user command only executes once during an interval, 0 represents every event will trigger the execution of user's command
  -n, --notify            enable system notify while event triggered (NOT IMPLEMENT)
  -r, --recursive         watch folders recursively (default true)
      --version           version for fswatcher
```

#### Example:

```shell
fswatcher -c "git add . && git commit -m 'update' && git push origin master" .
```



