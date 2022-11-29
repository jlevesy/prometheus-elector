## prometheus-elector

`prometheus-elector` leverages Kubernetes Leader Election to make sure that only one instance of Prometheus in a replicated workload has a specific configuration enabled.

### Main Use Case: Mimir Agent High Availability

Prometheus (in agent mode) is commonly used to push metrics to a  storage backend like [Mimir](https://grafana.com/oss/mimir/). 

If you want to get agent high availability with [Mimir](https://grafana.com/docs/mimir), you need to enable a feature called [HA deduplication](https://grafana.com/docs/mimir/latest/operators-guide/configure/configuring-high-availability-deduplication/), that requires nothing less than a KV store (could it be etcd or consul), which is a difficult thing to run and maintain...

Using `prometheus-elector`, we can instead make sure that only one instance has `remote_write` enabled at any point of time and guarantee a reasonable fallback delay when the prometheus leader becomes unavailable.


A known limitation at the moment is that prometheus in agent mode requires at least one [remote_write to start](https://github.com/prometheus/prometheus/blob/main/config/config.go#L115), which is the most important issue right now. This demo is running a [patched version of prometheus](https://github.com/jlevesy/prometheus/tree/allow-agent-no-remote-write) that removes this constraint without issues.

I reopened [the discussion on that topic](https://github.com/prometheus/prometheus/issues/9611) https://github.com/prometheus/prometheus/issues/11665.

### How it Works?

It is implemented using a sidecar container that rewites the configuration and injects `remote_write` rules in the configuration when elected leader. The setup is very similar to the usual `configmap-reloader` sidecar in Kubernetes deployment.

The configuration file differs a little from Prometheus, it is actually two Prometheus configurations in the same file:

- The `follower` section indicates the prometheus configuration to apply in follower mode
- The `leader` section indicates the changes to apply to the follower configuration when the instance is in elected leader. Please note that those changes gets "appended" to the follower configuration.

Here's an example that enables a `remote_write` target only when leader.

```yaml
# Follower is the configuration being applied when the instance is only follower.
follower:
  scrape_configs:
  - job_name:       'some job'
    scrape_interval: 5s
    static_configs:
    - targets: ['localhost:8080']

# Follower is the configuration being applied when the instance is leader.
leader:
  remote_writes:
    - url: http://remote.write.com
```

### Current status

This is a proof of concept, use this at your own risks! If you like the idea feel free to let me know!

Here's a TODO list

- (mandatory) Write some tests on the current implementation
- (mandatory) Trigger a reload when the config file is changed
- (mandatory) Retries on notification
- (mandatory) CI/CD pipeline
- (optional) Notify prometheus using signal ?

### Running this Demo

You need `ko`, `kubectl` and `k3d`, from there run `make run`

This will setup:

- [one agent](./examples/k8s/agent/agent.yaml) statefulset with the leader election going on, only one of them will push metrics to the storage
- [one storage](./examples/k8s/storage/storage.yaml) (prometheus [with the remote_write receiver enabeld](https://prometheus.io/docs/prometheus/latest/querying/api/#remote-write-receiver)) where all the metrics get pushed to

You can then port-forward to the storage pod via `kubectl port-forward -n storage storage-0 9090:9090` and see some kube metrics flowing.
