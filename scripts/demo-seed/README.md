# Demo seed

`demo-seed` fills a fresh Synclet with a demo workspace on mock data. The screenshots and the GIF in `.github/assets/` come from it.

It creates:

- the user `demo@example.com` (password `demo-password`) and renames the user's workspace to "Demo Shop";
- the connectors `airbyte/source-faker:6.2.38`, `airbyte/source-postgres:3.8.7` and `airbyte/destination-postgres:3.0.23`, the versions in the README's "Tested with" table;
- three PostgreSQL databases with generated rows, recreated on every run: `demo_inventory` (warehouses and stock items), `demo_partner_feed` (shipments) and `demo_warehouse` (the destination);
- four hourly connections into "Analytics warehouse": "Storefront users → Warehouse" and "Storefront orders → Warehouse" from the Faker source, "Inventory DB → Warehouse", and "Partner feed → Warehouse", whose database is dropped after setup so its sync fails;
- two sync runs of each working connection, with rows added to the inventory between them, and one failed run of the partner feed.

Every name and record is made up. The command refuses a workspace that already has connections, so run it against an empty Synclet database.

## Run it

Against the [Quick Start](../../README.md#quick-start) stack (`docker compose up -d`, Synclet on :8080, its PostgreSQL on 127.0.0.1:5437):

```bash
go run ./scripts/demo-seed -pg-dsn 'postgres://synclet:synclet@127.0.0.1:5437/postgres?sslmode=disable'
```

`-pg-dsn` is required and has no default: the command drops and recreates the `demo_*` databases there, so point it at a scratch PostgreSQL, never one that holds data you keep. The demo databases can live in Synclet's own PostgreSQL. Against `task dev` or another server, pass its address with `-url`.

Connector containers reach that PostgreSQL at `-connector-pg-host` (default `host.docker.internal`, which Docker Desktop and OrbStack provide) and `-connector-pg-port` (default: the port of `-pg-dsn`). On Linux, pass an address the containers can reach, such as the Docker bridge gateway, and publish the PostgreSQL port on it.

The other flags (`-email`, `-password`, `-name`, `-workspace`, `-runs`) are listed by `go run ./scripts/demo-seed -h`. Registration must be enabled (`AUTH_REGISTRATION_ENABLED=true`, the default); when the user exists, the command logs in with the given password.

A run takes about a minute once the connector images are pulled. The dashboard's totals come from a stats rollup that runs every 5 minutes by default (`PIPELINE_JOB_STATS_ROLLUP_INTERVAL`) and recomputes the last 2 hours of hourly buckets (`PIPELINE_STATS_ROLLUP_HOURLY_LOOKBACK`), so take the screenshots after one rollup and within 2 hours of the seed. Then the 24h dashboard shows 7 syncs, 1 failed and 2.5K records, and "Inventory DB → Warehouse" shows 1.4K (1357) records.

## Capture the images

`capture.mjs` drives a headless Chromium through Playwright from `front/node_modules` (`cd front && bun install`):

```bash
# 10 screenshots at 1440x900: dashboard, connection, topology, audit log and job logs, light and dark.
# Pass the id of "Inventory DB → Warehouse" and of its second job (from the Jobs page).
SYNCLET_URL=http://localhost:8080 node scripts/demo-seed/capture.mjs screenshots .github/assets/screenshots <connection-id> <job-id>

# The "create a connection → first sync" flow in the light theme, recorded as WebM.
SYNCLET_URL=http://localhost:8080 node scripts/demo-seed/capture.mjs gif /tmp/synclet-demo-video
```

The gif mode prints the video path and the seconds at which the flow starts and ends. Cut the GIF from the start mark to the end of the video (about 29 s), at 960 px and 12 fps, and check that no frame between wizard steps is blank:

```bash
V=/tmp/synclet-demo-video/<video>.webm; F="fps=12,scale=960:-1:flags=lanczos"
ffmpeg -ss 2.4 -t 29.4 -i "$V" -vf "$F,palettegen=stats_mode=diff:max_colors=128" /tmp/palette.png
ffmpeg -ss 2.4 -t 29.4 -i "$V" -i /tmp/palette.png -lavfi "$F[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle" -loop 0 .github/assets/demo-first-sync.gif
```

The social preview is `.github/assets/social-preview.svg`, rendered to `social-preview.png` at 1280x640. GitHub has no API for it: upload the PNG under the repository's Settings → General → Social preview.
