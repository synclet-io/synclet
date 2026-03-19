# Frontend

Vue 3, TypeScript, TanStack Vue Query, ConnectRPC, Tailwind CSS.

## Import Aliases

`@` = `./src`, `@entities` = `./src/entities`, `@features` = `./src/features`, `@shared` = `./src/shared`, `@pages` = `./src/pages`, `@widgets` = `./src/widgets`

## Entity Layer (`src/entities/`)

Each entity directory contains:
- `api.ts` — ConnectRPC client calls
- `composables.ts` — Vue Query hooks
- `types.ts` — Frontend-friendly TypeScript types

Proto generation: edit `.proto` files in `proto/`, run `task proto`. Output lands in `src/gen/`.
