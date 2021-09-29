# TeleWorker design document #

- [Library](#library)
- [Server](#server)
- [Client](#client)
- [Security](#security)
- [Testing](#testing)
- [Trade-offs](#trade-offs)

## Library

The library will contain all the functionality for starting, terminating, getting status, and streaming output of some jobs. The implementation can be split into 3 modules:

1. Core - the main logic for jobs management.
1. Broker - as we want to be able to stream some task output we need to provide all the logs produced by the task before the user connected as well as send all the upcoming logs.
This can be handled by using os.Pipe() and a simple custom broker. Using a pipe we basically publish job stdout to a channel - there always will be at least one subscriber reading that channel, in such a way we can have a buffer with all the logs the job produced.
Later, in case if the user wants to get the output - all we need to do is to create a copy of that buffer, subscribe so the copy could be also always updated with the latest logs in real time and simply stream that buffer.   
1. Resource control - since the user is able to limit the task resources we need to have a small layer for working with the file system.  

### Resource control
Three subsystems will be used for allowing a user to limit a job:
1.  blkio - parameter _weight_ is used.
1.  cpu - parameter _shares_ is used. 
1. memory - parameter _limit_in_bytes_ is used. It sets the upper limit of memory available to a particular job. 

Initially, we set up a **teleworker** group with blkio.weight and cpu.shares parameters set to 1000 (i.e. maximum).
Every job launched without limiting any of the parameters will be placed directly to **cgroup.procs** of that group, so basically we will write the job PID into the following:
```
/sys/fs/cgroup/cpu/teleworker/cgroup.procs
/sys/fs/cgroup/blkio/teleworker/cgroup.procs
/sys/fs/cgroup/memory/teleworker/cgroup.procs
```
If a user provides any of the limits when creating a task, we create a new group inside of the _teleworker_ named by the ID of that task. For consistency we do that for all the resources (i.e. the group _cpu/teleworker/uuid_ will be created even if only memory and io limits were set by the user). This means that we write PID of any limited job to
```
/sys/fs/cgroup/<resource>/teleworker/<uuid>/cgroup.procs
```

When the job is finished or terminated we also remove the related directories.


## Server

Server will extensively use the library and will probably have only some basic auth logic (see [Security](#security)) and a layer of logic for storing the current system state in memory.  


## Client

Client is a simple CLI for the communication with the server. It will allow running a set of commands against the server:

1. Start a job: passes a command provided by the user for direct execution on the server side. The ID of the job will be sent in response.
If the user wants to limit resources that will be available to a job upon execution he is required to do it while sending a start request.
No update functionality will be provided in terms of resources control - if the user wants to add or change some limits he is required to stop the job and create a new one with the required parameters.
The following set of flags is used for this purpose, user can set the required limit in one of the groups:
    * **mem** - memory limit for a job in megabytes
    * **cpu** - cpu share in percents (1-100) available to this job
    * **io** - proportion of I/O access (1-100) available to this job
1. Stop the job: a user is required to provide a job ID for the job termination. The default behavior is to kill the task as it will trigger _SIGKILL_ to be sent for the command termination.
1. Get the status of the job. Requires only the job ID to be sent, the user gets in return the job status, all the job resource limits set upon job creation and exit code (applies only if the job is in the finished or stopped status)
1. Stream the output of the job. Requires only the job ID, starts the stream of the job stdout - the user gets everything that was written by the command until that moment and continues to get the command logs in real time until either the job is finished/terminated or the user interrupts the stream command execution (CTRL-C).
It is a completely valid scenario to request the logs of both stdout and stderr (or even stdout and once again stdout) of the same job at the same time. 
    * **err** - optional flag, if provided starts the stream of stderr instead of stdout

Some examples can be found under the Usage section of the readme file.


## Security

Client and server keys and certificates will be either presented in the repository or generated using the provided script. 

mTLS authentication will be used. As we don't want to introduce some sophisticated auth system in this exercise we might consider all the users with valid certificates to be authorized within the system. 

In order to identify the user and to connect him to his jobs we might extract CN and use it as a user login within the system. 

## Testing

Unit tests as well as integration tests will be provided.

## Trade-offs

### Lack of configuration options
Writing proper configs will be skipped on purpose, the values will be hardcoded or provided via flags when it will make sense.

### Pre-generated secrets
All the required secrets will be provided in order to simplify local launches and testing. 

### Lack of persistency
The system state is stored in memory, so there is a possibility of irreversible data loss and system clogging in case of unforeseen server outage.

### Possibility of jobs ids collisions
It was decided to choose UUIDs to use as job identificators. For the sake of simplicity, the possibility of collisions will not be taken into account in the current implementation.

### Control groups v2 are not supported
cgroups v1 are used for limiting processes resources - support for v2 is not taken into account.

### All the outputs are stored in memory
As mentioned above, buffers will be used to store everything the task produces while it is alive.
In the current state it obviously has no use in production, since we don't clean up the buffers at all (i.e. removing the old logs) as well as we don't limit the maximum job execution time, which will quickly result in the system running out of memory in real-life use. 