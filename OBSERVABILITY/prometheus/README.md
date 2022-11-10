# Prometheus related stuff

## Compute the pods throttling

```bash
100 * irate(container_cpu_cfs_throttled_periods_total{container="prometheus"}[2m]) / irate(container_cpu_cfs_periods_total{container="prometheus"}[2h])
```