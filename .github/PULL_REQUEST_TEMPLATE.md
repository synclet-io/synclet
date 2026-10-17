## What does this change?

<!-- One or two sentences. Link the issue it closes: "Closes #123". -->

## Why?

<!-- The problem it solves or the user it helps. -->

## How was it tested?

<!-- Commands you ran, tests you added, screenshots for UI changes. -->

## Checklist

- [ ] The pull request title follows [Conventional Commits](https://www.conventionalcommits.org/) (`feat(scope): …`, `fix(scope): …`, `docs: …`)
- [ ] Tests added or updated for the change
- [ ] `task lint` and `task test` pass (and `cd front && bun run test:run` for frontend changes)
- [ ] Generated code regenerated, not edited by hand (`task proto`, `task boilerplate-go`)
- [ ] Schema changes are in a new migration file
- [ ] Documentation updated if behaviour or configuration changed
