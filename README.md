## prometheus-elector

`prometheus-elector` leverages Kubernetes Leader Election to make sure that only one instance of Prometheus in a replicated workload has a specific configuration enabled.

### Use Case: Prometheus Agent High Availability

Prometheus (in agent mode) is commonly used to push metrics to a  storage backend like [Mimir](https://grafana.com/oss/mimir/). 

If you want to get agent high availability with [Mimir](https://grafana.com/docs/mimir), you need to enable a feature called [HA deduplication](https://grafana.com/docs/mimir/latest/operators-guide/configure/configuring-high-availability-deduplication/), that requires nothing less than a KV store (could it be etcd or consul), which is a difficult thing to run and maintain...

Using `prometheus-elector`, we can instead make sure that only one instance has `remote_write` enabled at any point of time and guarantee a reasonable fallback delay when the prometheus leader becomes unavailable.


A known limitation at the moment is that prometheus in agent mode requires at least one [remote_write to start](https://github.com/prometheus/prometheus/blob/main/config/config.go#L115), which is the most important issue right now. This demo is running a [patched version of prometheus](https://github.com/jlevesy/prometheus/tree/allow-agent-no-remote-write) that removes this constraint without issues.
I reopened [the discussion on that topic](https://github.com/prometheus/prometheus/issues/9611) https://github.com/prometheus/prometheus/issues/11665, and got positive feedback!

The next version of Prometheus will fully support this use case!

### Use Case: Prometheus High Availability

This one is more theoretical at the moment, but we could have an Active passive setup of prometheus, with the leader only having alerts enabled.
The problem in that case is the read path, because metrics consumers like Grafana, don't properly support having multiple sources.
However, if I could find a way to have prometheus-elector proxy all the requests to the current active leader, this would solve this issue! If the leader receives the requests, it fowards it to its prometheus instance. If the follower receives the request, it forwards it to the leader!

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

### Installing Prometheus Elector

You can find [an helm chart](./helm) in this repository, as well as [values for the HA agent example](./example/k8s/agent-values.yaml).

### Current Status

This is still a proof of concept, until Prometheus releases https://github.com/prometheus/prometheus/pull/11709, use this at your own risks! If you like the idea feel free to let me know!

Here's what will come next!

- Proxy to route requests to the leader!
- Release pipeline for the Helm chart?
- (optional) Notify prometheus using signal ?

### Running an Example Locally

You need `ko`, `kubectl` and `k3d`, from there run `make run`

This will setup:

- [An agent workload](./examples/k8s/agent/agent.yaml) statefulset with the leader election going on, only one of them will push metrics to the storage
- [A storage storage](./examples/k8s/storage/storage.yaml) (prometheus [with the remote_write receiver enabeld](https://prometheus.io/docs/prometheus/latest/querying/api/#remote-write-receiver)) where all the metrics get pushed to

You can then port-forward to the storage pod via `kubectl port-forward -n storage storage-0 9090:9090` and see some kube metrics flowing.
