# Synclet

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/synclet)](https://artifacthub.io/packages/search?repo=synclet)

[Synclet](https://github.com/synclet-io/synclet) is an open-source data synchronization platform that moves data between sources and destinations with Airbyte-compatible connectors.
On Kubernetes every sync runs as its own Kubernetes Job, so no Docker daemon is needed.

## Prerequisites

- Kubernetes 1.21+
- Helm 3.8+ (OCI registry support)
- A default StorageClass, for the bundled PostgreSQL, or an external PostgreSQL 16 database
- An Ingress controller (the default class is `nginx`) or a Gateway API implementation to expose the UI
- Optional: Prometheus Operator for `serviceMonitor`, Vault Secrets Operator for `vso`

## Install

```bash
helm install synclet oci://ghcr.io/synclet-io/charts/synclet \
  --namespace synclet --create-namespace \
  -f values.yaml
```

Add `--version <chart version>` to pin a release.
The JWT secret, the encryption key, the internal API secret and the PostgreSQL password are generated on first install and kept across upgrades when left empty.
To set them yourself, generate each with `openssl rand -base64 32`:

```yaml
auth:
  jwtSecret: "<at least 32 bytes>"
encryption:
  key: "<base64 of 32 bytes>"
```

## Upgrade

```bash
helm upgrade synclet oci://ghcr.io/synclet-io/charts/synclet \
  --namespace synclet \
  -f values.yaml
```

Read the [CHANGELOG](https://github.com/synclet-io/synclet/blob/main/deploy/helm/synclet/CHANGELOG.md) before upgrading.
Charts before 0.4.0 ran the executor as a Deployment: in distributed mode, delete it first with `kubectl delete deployment <fullname>-executor`.

### Upgrading from 0.9.0

The MCP exposure keys were renamed, as they apply to the Ingress and the HTTPRoute alike: `mcp.ingress.mode` → `mcp.exposure`, `mcp.ingress.paths` → `mcp.paths`.
The old keys are no longer read: move their values to the new keys before upgrading, or MCP renders with the defaults (`shared` and the MCP and OAuth paths).

With `mcp.exposure: separate`, each enabled way the server is exposed (Ingress, HTTPRoute) now gets its own MCP twin. Two 0.9.0 setups render differently:

- With `server.gateway.enabled`, separate mode renders an MCP HTTPRoute and requires `mcp.gateway.parentRefs`; set it before upgrading, or rendering fails.
- With `server.ingress.enabled: false`, separate mode no longer renders an MCP Ingress. Expose MCP through the gateway (`server.gateway.enabled` and `mcp.gateway.parentRefs`), or keep the Ingress enabled.

## Uninstall

```bash
helm uninstall synclet --namespace synclet
```

The PostgreSQL volume, and the bundled Garage volume and Secret, are kept.

## Modes

**Standalone** (`mode: standalone`, the default): one server pod runs the API, the scheduler and the executor.
The simplest setup, suited to small and medium workloads.

**Distributed** (`mode: distributed`): the server and jobs Deployments and an executor StatefulSet run separately.
The API scales on its own, and a failing sync cannot take it down.
Set `executor.enabled: false` to run the executor outside the chart.

In both modes the executor launches each sync as a Kubernetes Job in `k8s.namespace`, the release namespace when empty.
A namespace set there must exist before install.

## Main values

| Value | Description | Default |
|-------|-------------|---------|
| `mode` | `standalone` or `distributed` | `standalone` |
| `image.registry`, `image.repository`, `image.tag` | Synclet image; an empty tag uses the chart's `appVersion` | `ghcr.io`, `synclet-io/synclet`, `""` |
| `server.ingress.enabled` | Create an Ingress | `true` |
| `server.ingress.hostname`, `server.ingress.ingressClassName` | Ingress host and class | `synclet.local`, `nginx` |
| `server.ingress.tls` | TLS on the Ingress, with the `<fullname>-tls` Secret | `false` |
| `server.gateway.enabled` | Create a Gateway API HTTPRoute attached to `server.gateway.parentRefs` | `false` |
| `mcp.enabled` | Run the MCP server for AI clients, with a `<fullname>-mcp` Service | `false` |
| `mcp.exposure` | `shared`: `mcp.paths` on the main Ingress and HTTPRoute; `separate`: each enabled exposure of the server gets an MCP-only twin, and the main Ingress and HTTPRoute carry no MCP paths: with `server.ingress.enabled` an Ingress on `mcp.ingress.hostname` (required), with `server.gateway.enabled` an HTTPRoute attached to `mcp.gateway.parentRefs` (required; `mcp.gateway.hostnames`, `mcp.gateway.timeouts`, `mcp.gateway.labels` and `mcp.gateway.annotations` optional); with neither, only the MCP Service | `shared` |
| `mcp.paths` | Paths routed to the MCP Service in both exposures | `/mcp`, `/.well-known/oauth-protected-resource`, `/.well-known/oauth-authorization-server`, `/oauth/register`, `/oauth/token` |
| `server.publicUrl` | URL users reach Synclet at; required when both Ingress and HTTPRoute are enabled | derived |
| `postgresql.enabled` | Run the bundled PostgreSQL, meant for development and testing | `true` |
| `externalDatabase.*` | External database (host, port, user, password, database, sslmode) or `existingSecret` with a full DSN | empty |
| `cors.*` | CORS of the public API (`allowAllOrigins`, `allowedOrigins`, `allowedOriginGlobs`, ...); unset keys keep server defaults | unset |
| `oidc.providers` | SSO providers; each entry sets `slug`, `issuer`, `clientID`, `clientSecret` and optional role mapping (`roleClaim`, `roleMapAdmin`, `roleMapEditor`, `roleMapViewer`, `defaultRole`, `roleSync`) | `[]` |
| `extraEnv` | Extra env vars of every Synclet container; `server.extraEnv`, `jobs.extraEnv` and `executor.extraEnv` add to one component | `[]` |
| `executor.enabled` | Run the executor StatefulSet in distributed mode | `true` |
| `executor.replicaCount` | Executor replicas in distributed mode | `1` |
| `k8s.namespace` | Namespace of the sync Jobs | release namespace |
| `logArchive.backend` | Where finished job logs go: `none` (stay in PostgreSQL), `filesystem`, `s3`, `gcs` | `none` |
| `logArchive.garage.enabled` | Run a bundled single-node Garage S3 store for log archives | `false` |

`values.yaml` documents every value.

## OIDC role mapping

In single-workspace mode an OIDC login sets the user's role in the default workspace from a claim of the ID token.
`roleClaim` names the claim, a string or a list such as authentik's `groups`, and `roleMapAdmin`, `roleMapEditor` and `roleMapViewer` name the claim value of each role, checked in that order.
A claim that matches no mapping gives `defaultRole` (`viewer` when unset).
The workspace's first member always joins as admin.
`roleSync: login`, the default, applies the claim on every login, so a role changed in the UI is overwritten; `roleSync: first_login` applies it only when the user joins.
A sync never demotes the workspace's last admin.
In multi-workspace mode the role keys are ignored and the server logs a warning.

```yaml
oidc:
  providers:
    - slug: authentik
      issuer: https://auth.example.com/application/o/synclet/
      clientID: synclet
      clientSecret: "<client secret>"
      roleClaim: groups
      roleMapAdmin: synclet-admins
      roleMapEditor: synclet-editors
      roleSync: login
```

## Documentation

- [Kubernetes deployment](https://synclet-io.github.io/synclet-site/docs/deployment/kubernetes/)
- [Production checklist](https://synclet-io.github.io/synclet-site/docs/deployment/production/)
- [Configuration](https://synclet-io.github.io/synclet-site/docs/getting-started/configuration/)
- [Environment variables](https://synclet-io.github.io/synclet-site/docs/reference/environment-variables/)
- [Source and issues](https://github.com/synclet-io/synclet)
