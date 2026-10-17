# Security Policy

## Reporting a vulnerability

Please do **not** report security vulnerabilities through public issues, discussions or pull requests.

Report them privately through GitHub's private vulnerability reporting:
**[Report a vulnerability](https://github.com/synclet-io/synclet/security/advisories/new)** (the *Security* tab of this repository → *Report a vulnerability*).

Include what you can of:

- the affected version (`synclet --version`, the image tag or the Helm chart version);
- how Synclet is deployed (binary, Docker Compose, Kubernetes/Helm) and the relevant configuration, with secrets removed;
- steps to reproduce, a proof of concept, and the impact you expect.

## What to expect

| Step | Target time |
|---|---|
| Acknowledgement of your report | within 3 business days |
| First assessment (confirmed or not, severity) | within 7 days |
| Fix or mitigation for a confirmed critical or high severity issue | within 30 days |
| Fix for medium or low severity | in a following release, usually within 90 days |

We keep you updated in the advisory thread, agree on a disclosure date with you, and credit you in the published GitHub Security Advisory unless you prefer to stay anonymous.

## Supported versions

Security fixes are made on the latest minor release. Please upgrade to the latest release before reporting, if you can.

| Version | Supported |
|---|---|
| latest 0.x minor | yes |
| older releases | no |

## Scope

In scope: the Synclet server and UI, the coordinator, the Helm chart and the official container images in this repository.

Out of scope: third-party connector images (including connectors that implement the Airbyte Protocol and are published by others) — report those to their maintainers; findings that need an already compromised host or administrator credentials.
