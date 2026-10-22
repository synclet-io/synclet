# Runtime config

A source's or destination's `runtime_config` sets the resources and placement of its connector containers. It is a JSON object passed as a *string*: `update_source` and `update_destination` take it as `"runtime_config": "{\"memory_limit\":\"1Gi\"}"`. `create_source` and `create_destination` do not take it; create first, then set it with `update_source` or `update_destination`. `get_source` and `get_destination` return the current value; read it first and send back every key you keep.

Every key is optional. A key left out uses the instance default (`get_system_info` returns the runtime defaults). An empty string `""` removes the runtime config. A key not listed on this page is refused, at the top level and inside a `docker_volumes` entry.

## Resources (both executors)

| Key | Value | Docker executor | Kubernetes executor |
|---|---|---|---|
| `cpu_request` | CPU quantity | ignored | container CPU request |
| `cpu_limit` | CPU quantity | container CPU limit (default 1 core) | container CPU limit |
| `memory_request` | memory quantity | ignored | container memory request |
| `memory_limit` | memory quantity | container memory limit (default 2Gi) | container memory limit |

Quantities use the Kubernetes format, given as strings. CPU: `"500m"` is half a core, `"2"` is two cores. Memory: `"512Mi"`, `"1Gi"`, `"1.5Gi"` (binary units `Ki Mi Gi Ti`) or `"500M"`, `"1G"` (decimal units).

## Docker executor

| Key | Value |
|---|---|
| `docker_network` | Name of the Docker network the connector containers join. |
| `docker_volumes` | Array of bind mounts: `{"host_path": "...", "container_path": "...", "read_only": true}`. `host_path` and `container_path` are required; `container_path` cannot be `/tmp` or under it. |

The Kubernetes executor ignores these keys.

```json runtime_config
{
  "cpu_limit": "2",
  "memory_limit": "4Gi",
  "docker_network": "warehouse",
  "docker_volumes": [
    {"host_path": "/srv/exports", "container_path": "/data/exports", "read_only": true}
  ]
}
```

## Kubernetes executor

| Key | Value |
|---|---|
| `tolerations` | Array of Kubernetes `Toleration` objects (`key`, `operator`, `value`, `effect`). |
| `node_selector` | Object of label → value, all strings. |
| `affinity` | A Kubernetes `Affinity` object (`nodeAffinity`, `podAffinity`, `podAntiAffinity`), in the Kubernetes JSON field names. |
| `service_account_name` | Service account the connector pods run as; a DNS-1123 name. |

`tolerations`, `node_selector` and `affinity` replace the instance default when set; they are not merged with it. The Docker executor ignores these keys.

```json runtime_config
{
  "cpu_request": "500m",
  "cpu_limit": "1",
  "memory_request": "512Mi",
  "memory_limit": "1Gi",
  "tolerations": [
    {"key": "dedicated", "operator": "Equal", "value": "sync", "effect": "NoSchedule"}
  ],
  "node_selector": {"pool": "sync"},
  "affinity": {
    "nodeAffinity": {
      "requiredDuringSchedulingIgnoredDuringExecution": {
        "nodeSelectorTerms": [
          {"matchExpressions": [{"key": "kubernetes.io/arch", "operator": "In", "values": ["amd64"]}]}
        ]
      }
    }
  },
  "service_account_name": "synclet-connectors"
}
```

## Checks

`update_source` and `update_destination` refuse a runtime config that is not a single JSON object, one with a key not listed on this page, one with an invalid quantity, a malformed `tolerations`, `node_selector` or `affinity`, a `service_account_name` that is not a DNS-1123 name, or a Docker volume without both paths or mounted under `/tmp`. A refused value is not saved, and the error names the invalid JSON or the unknown key.
