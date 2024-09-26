Configuration Walkthrough
===========================

In order to integrate leader elector container to Prometheus operator, you will need to follow this steps:

## Add leader elector configuration file

Create a Secret that contain prometheus leader configuration, in this example we would like for the leader pod to send metrics using remote write

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prometheus-leader-config-secret
  namespace: infra
  annotations:
    vault.security.banzaicloud.io/vault-addr: "https://vault.infra:8200"
    vault.security.banzaicloud.io/vault-role: "infra"
    vault.security.banzaicloud.io/vault-skip-verify: "true"
    vault.security.banzaicloud.io/vault-path: "kubernetes"
type: Opaque
stringData:
  leader.yaml: |-
    remote_write:
      - url: coralogix-endpoint
        remote_timeout: 120s
        name: coralogix
        tls_config:
          insecure_skip_verify: true
        authorization:
          type: Bearer
          credentials: api-key-stored-in-vault
        follow_redirects: true
        enable_http2: true
        queue_config:
          capacity: 10000
          max_shards: 50
          min_shards: 1
          max_samples_per_send: 2000
          batch_send_deadline: 5s
          min_backoff: 30ms
          max_backoff: 5s
        metadata_config:
          send: true
          send_interval: 1m
          max_samples_per_send: 2000
```

### Add Additional volumes to Prometheus Statefulset

in order to mount the leader elector configuration file, you need to add the following configuration to the `values.yaml` file:

```yaml
    volumes:
      - name: leader-volume-secret
        secret:
          secretName: prometheus-leader-config-secret
```

### Add prometheus-elector container

in order to add the leader elector container to the prometheus statefulset, you need to add the following configuration to the `values.yaml` file:

container configuration contain 3 volumeMounts:

1. config: this volume contain the Prometheus configuration generate by the Prometheus operator
2. config-out: this volume contain the Prometheus configuration generate by the leader elector container
3. leader-volume-secret: this volume contain the leader elector configuration file

for every product line you will need to set a different lease-name


```yaml
    containers:
      - name: prometheus-elector
        image: image
        imagePullPolicy: Always
        args:
          - -lease-name=my-prometheus-elector-lease
          - -leader-config=/etc/config_leader/leader.yaml
          - -lease-namespace=infra
          - -config=/etc/prometheus/config_out/prometheus.env.yaml
          - -output=/etc/prometheus/config_out/prometheus_config.yaml
          - -notify-http-url=http://127.0.0.1:9090/-/reload
          - -readiness-http-url=http://127.0.0.1:9090/-/ready
          - -healthcheck-http-url=http://127.0.0.1:9090/-/healthy
          - -api-listen-address=:9095
        command:
        - ./elector-cmd
        ports:
          - name: http-elector
            containerPort: 9095
            protocol: TCP
        securityContext:
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
        resources:
          {}
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /_elector/healthz
            port: http-elector
          initialDelaySeconds: 30
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 4
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /_elector/healthz
            port: http-elector
          initialDelaySeconds: 30
          periodSeconds: 15
          successThreshold: 1
          timeoutSeconds: 10
        volumeMounts:
          - mountPath: /etc/prometheus/config_out
            name: config-out
          - mountPath: /etc/config_leader
            name: leader-volume-secret
          - mountPath: /etc/prometheus/config
            name: config
```

### Add prometheus-elector container as initContainer

in order to add the leader elector container as initContainer to the prometheus statefulset, you need to add the following configuration to the `values.yaml` file:
this container verify that when prometheus container start it will have a valid configuration file

```yaml
    initContainers:
      - name: init-prometheus-elector
        image: image
        imagePullPolicy: Always
        args:
          - -config=/etc/prometheus/config_out/prometheus.env.yaml
          - -output=/etc/prometheus/config_out/prometheus_config.yaml
          - -leader-config=/etc/config_leader/leader.yaml
          - -init
        command:
        - ./elector-cmd
        securityContext:
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
        volumeMounts:
          - mountPath: /etc/prometheus/config_out
            name: config-out
          - mountPath: /etc/config_leader
            name: leader-volume-secret
          - mountPath: /etc/prometheus/config
            name: config
```

### Override the prometheus container configuration

in order to override the prometheus container configuration (config.file), the only way to do it is
to add the all prometheus args the the `values.yaml` file:

```yaml
    containers:
      - name: prometheus
        args:
           - --web.console.templates=/etc/prometheus/consoles
           - --web.console.libraries=/etc/prometheus/console_libraries
           - --config.file=/etc/prometheus/config_out/prometheus_config.yaml
           - --web.enable-lifecycle
           - --web.enable-remote-write-receiver
           - --web.external-url=https://prometheus.k8s.yotpodev.com/
           - --web.route-prefix=/
           - --log.level=debug
           - --log.format=json
           - --storage.tsdb.retention.time=7d
           - --storage.tsdb.path=/prometheus
           - --no-storage.tsdb.wal-compression
           - --web.config.file=/etc/prometheus/web_config/web-config.yaml
           - --query.max-concurrency=40
           - --query.timeout=110s
           - --storage.tsdb.max-block-duration=2h
           - --storage.tsdb.min-block-duration=2h
```

### Add permissions to prometheus elector

in order for the prometheus elector to be able create and update leases
a new role, clusterrole, rolebinding and clusterrolebinding will need to be created


```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: prometheus-elector-role
  namespace: infra
rules:
  - apiGroups:
      - coordination.k8s.io
    resources:
      - leases
    verbs:
      - get
      - list
      - watch
      - create
      - update

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus-elector-clusterrole
rules:
  - apiGroups:
      - ""
    resources:
      - nodes
      - nodes/proxy
      - nodes/metrics
      - services
      - endpoints
      - pods
      - ingresses
      - configmaps
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - extensions
      - networking.k8s.io
    resources:
      - ingresses/status
      - ingresses
    verbs:
      - get
      - list
      - watch
  - nonResourceURLs:
      - /metrics
    verbs:
      - get

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: prometheus-elector-rolebinding
  namespace: infra
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: prometheus-elector-role
subjects:
  - kind: ServiceAccount
    name: my-prometheus-operator
    namespace: infra

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus-elector-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus-elector-clusterrole
subjects:
  - kind: ServiceAccount
    name: my-prometheus-operator
    namespace: infra

```

### Configuring additional service monitor

in order to scrape the metrics from the leader elector container, you need to add the following configuration to the `values.yaml` file:

```yaml
additionalServiceMonitors:
    - name: "prometheus-elector"
      endpoints:
      - path: /_elector/metrics
        port: http-elector
      namespaceSelector:
        matchNames:
        - infra
      selector:
        matchLabels:
          app: kube-prometheus-stack-prometheus
          release: my-prometheus-operator
          self-monitor: "true"
```

### Configuring additional port for prometheus service

in order to expose the leader elector metrics, you need to add the following configuration to the `values.yaml` file:

```yaml
    additionalPorts:
    - name: http-elector
      port: 9095
      targetPort:  http-elector
```