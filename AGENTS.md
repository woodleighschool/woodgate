# AGENTS.md

Repository guidance for WoodGate.

## Approach

- Stay within the requested scope and preserve unrelated local changes.
- This is a purpose-built internal visitor application, not a SaaS platform. Prefer direct code for current school workflows.
- Simplify and modernize existing code before adding abstractions, compatibility layers, or speculative configuration.
- Follow the shared Woodstar tooling baseline while respecting React-admin and iOS ownership.

## Repository Map

- Go process composition: `cmd/woodgate`
- Application behavior: `internal/app`
- Domain and configuration: `internal/domain` and `internal/config`
- External systems: `internal/platform`
- Persistence: `internal/store`
- HTTP and API transport: `internal/transport`
- OpenAPI contract: `api/openapi.yaml`
- React-admin frontend: `web/`
- iOS application: `app/`

Keep platform details at the edge and business rules in the application/domain packages. Don't build a generic identity or workflow framework.

## Commands

Use Mise tasks as the repository contract.

- Dependencies: `mise run deps`
- Build: `mise run build`
- Tests: `mise run test`
- Lint: `mise run lint`; fixes: `mise run lint-fix`
- Format: `mise run format`; check: `mise run fmt-check`
- Regenerate API and SQL outputs: `mise run generate`
- Module and workflow checks: `mise run tidy-check`, `mise run workflow-lint`
- iOS build and analysis: `mise run //app:build`, `mise run //app:lint`

Run frontend commands through `//web:*` and iOS commands through `//app:*` when only one stack is in scope.

## Engineering Rules

- Select every Microsoft Graph field used during reconciliation; absent snapshot fields can overwrite stored values.
- Search and filtering use values visible to users, not hidden reference-only fields.
- API changes update `api/openapi.yaml`, generated web types, handlers, and tests together.
- Frontend code uses React-admin, Oxc, and the generated API types. Don't add a parallel formatting, state, or API-client stack.
- Keep real identities, tenant IDs, credentials, check-in data, and local environment files out of tests and version control.

## Commits

- Use focused Conventional Commits.
- Don't push, deploy, publish, or contact live tenants unless explicitly requested.
- Report checks run, skipped checks, and unresolved failures.
