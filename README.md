# TeleWorker - simple job worker service #

- [Description](#description)
- [Build](#build)
- [Usage](#usage)

## Description

TeleWorker is a job worker service solution, which includes a library for managing jobs, a server using the library and a simple CLI for the communication with the server. 

## Build
TBD

## Usage
Everything can be managed via the command line, no advance preparation is required.

### Start a job
Starts a job, returns uuid. Optional flags are available for limiting the job resources:

* **mem** - memory limit in megabytes
* **cpu** - cpu share in percents (1-100)
* **io** - I/O access proportion in percents (1-100)
```
$ teleworker start
$ db759134-e42e-4b39-8c88-c2359219b9ed
```

### Stop some job
For the default behavior (terminating the task by sending SIGTERM):
```
$ teleworker stop <uuid>
```
Flags **--gentle** and **--force** can be used to terminate the task by sending an interruption request (SIGINT) and killing it (SIGKILL) respectively:
```
$ teleworker stop --gentle <uuid>
$ teleworker stop --force <uuid>
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