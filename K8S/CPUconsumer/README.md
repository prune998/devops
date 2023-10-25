# CPU Consumer 

This is a shitty program that is supposed to use some amount of CPU...

## Usage

```bash
./CPUconsumer -h

Usage of ./CPUconsumer:
  -logLevel="warn": log level from debug, info, warning, error. When debug, genetate 100% Tracing
  -version=false: Show version and quit
  -waitCPU=0: how many CPU to use in the wait phase(override GOMAXPROCS)
  -waitDuration=30s: how long to wait before work, in Golang Duration
  -workCPU=0: how many CPU to use in the work phase(override GOMAXPROCS)
  -workDuration=10s: how long to work, then wait, in Golang Duration
```

`numCPU` option is used to tell how many CPU to "consume".

This is achieved by:
- running a `for{}` loop per full CPU count
- running one more loop with a `sleep` delay equal to the fraction of the CPU

So `-numCPU=1` should consume one CPU, `-numCPU=0.5` should consume 0.5 and `-numCPU=0` should consume whatever is available on your computer.

```warning
There is no guarantee that this program will behave as intended. It will consume CPU, for sure, but rely on really basic things that may be different depending on the OS, version and so on.

It has been proven to work well in Kubernetes (GKE), so far.
```

## working and waiting

You can set the number of CPU to consume during the `work` and the `wait` phase using `-workCPU` or `-waitCPU`. This will mimmick a more realistic burst workload where you
still use some CPU when mostly nothing is happening.


## Build

```bash
go mod tidy
go build
```

## Docker image build

```bash
docker  build -t prune/cpuconsumer:v0.0.2 --build-arg VERSION=0.0.2 .
docker push prune/cpuconsumer:v0.0.2
```

## Docs

As a reminder, Linux CFS quota split the CPU usage into a `period`, defaulting to 100ms. It's the `cfs_period_us=100000`.
Then, each `cGroup` is assigned a `quota`. `cfs_quota_us=100000` for example

doing `cfs_quota_us/cfs_period_us` shows us how many CPU we can use. (1 CPU, or 1 CORE from the examplke above).

### Checking throttling on a server

1. find the process PID
    `ps auxwww | grep CPUconsumer`
2. list the `cGroups` of the process
    `cat /proc/104711/cgroup`
3. use the previous container's ID to get to the cGroup values or use `grep` to find the right folder:
    
    ```bash
    grep -r 104711 /sys/fs/cgroup/cpu,cpuacct/*

    /sys/fs/cgroup/cpu,cpuacct/kubepods/burstable/podcd1ebac2-4385-4d0d-8bfd-533c15a34440/554fb5dd2bcacad9786e14b90fe4f36da70a9c414b3cacfe1c95dbb802df3f1e/cgroup.procs:104711
    ```
4. dump the CFS values

    ```bash
    cd /sys/fs/cgroup/cpu,cpuacct/kubepods/burstable/podcd1ebac2-4385-4d0d-8bfd-533c15a34440/554fb5dd2bcacad9786e14b90fe4f36da70a9c414b3cacfe1c95dbb802df3f1e/
    cat cpu.cfs_period_us cpu.cfs_quota_us cpu.stat cpu.shares
    ```
5. check the usage of the cGroup using `systemd-cgtop`

    ```bash
    systemd-cgtop kubepods/burstable/podcd1ebac2-4385-4d0d-8bfd-533c15a34440/554fb5dd2bcacad9786e14b90fe4f36da70a9c414b3cacfe1c95dbb802df3f1e
    ```