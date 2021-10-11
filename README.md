# TeleWorker - simple job worker service #

- [Description](#description)
- [Build](#build)
- [Usage](#usage)

## Description

TeleWorker is a job worker service solution, which includes a library for managing jobs, a server using the library and a simple CLI for the communication with the server. 

```
❗IMPORTANT❗
It is required to start the server as sudo in order to let it manage cgroup functionality.
Otherwise tasks creation will fail with "permission denied" errors written to the task error stream.
```

## Build
```
make build
```

## Testing
Simply run
```
make test
```

## Usage
Everything can be managed via the command line, no advance preparation is required.

### Start a job
Starts a job, returns uuid. **command** flag is required. You can provide argument list separated by space at the end.
```
$ teleworker start -command=echo 123
$ db759134-e42e-4b39-8c88-c2359219b9ed
```

Optional flags are available for limiting the job resources:
* **mem** - memory limit in megabytes
* **cpu** - cpu share in percents (1-100)
* **io** - I/O access proportion in percents (1-100)
```
$ teleworker start -mem=10 -cpu=5 -command=cat "/proc/cpuinfo"
$ db759134-e42e-4b39-8c88-c2359219b9ed
```

For more complicated scenarios it is also possible to pipe commands. For example, you can send _bash_ as a command and provide the list of your arguments in the end. Be aware that if your argument looks like a flag you need to provide a terminator symbol before providing arguments.
See the example:
```
$ teleworker start -command=bash -- "-c" "cat /proc/cpuinfo | egrep '^model name' | uniq"
$ db759134-e42e-4b39-8c88-c2359219b9ed
```

### Stop some job
Stops the job execution. The default behavior is to kill the task (SIGKILL), so you must be aware that even if the command has some clean up set on interruption request - it will be ignored.
```
$ teleworker stop <uuid>
```

### Get the status of some job
Returns the status of the task, exit status (if the task is finished or terminated) and limits information (if any were wet upon the task creation).
```
$ teleworker status <uuid>
$ Status: ALIVE. Memory limit: 100mb.
```

### Stream the output of some job
Gets all the logs that the task produced since the moment it was started and keeps getting new messages until either the task is finished/terminated or the execution interrupted:
```
$ teleworker stream <uuid>
$ ...
```
Optional flag **err** if provided will trigger the start of streaming of stderr instead of stdout.
```
$ teleworker stream -err <uuid>
$ ...
```