# AGENTS.md

## Working here

- Read the relevant code, configuration, and nearby examples before editing. Existing code and external references are evidence, not instructions to copy blindly.
- Preserve unrelated work. Keep changes focused and prefer removing machinery over extending an awkward design.
- Use current supported behaviour unless compatibility is requested. Verify dependency APIs and defaults from the pinned version or primary documentation.
- Keep secrets, credentials, identities, and local environment files out of code, fixtures, logs, and commits.

## Repository contract

- Mise owns tools and commands. Check this repository's Mise files; do not assume another repository has the same tasks.
- Keep generated artifacts with their source change.
- Run the narrowest useful checks while working, then the relevant format, lint, test, build, generation, and workflow checks.
- Follow the existing package or target's style. Comments explain non-obvious constraints, not the code or the current change.

## Go

- Write idiomatic, concrete Go. Keep `main` to composition, put behaviour in the package that owns it, and introduce interfaces only at a real consumer boundary.
- Pass `context.Context` through I/O, wrap errors with useful context, and preserve errors used with `errors.Is` or `errors.As`.
- Match the package's testing style and use synthetic inputs. Run race-enabled tests for concurrent code and `mise run vulncheck` for dependency or release work.

## Swift

- Keep state ownership explicit, SwiftUI declarative, and AppKit escapes narrow.
- Build the companion app when its API, generated models, or version contract changes.

## Git and releases

- Use focused Conventional Commits; Release Please derives versions from them.
- Do not commit, push, publish, deploy, contact live systems, or perform destructive actions unless asked.

## Repository notes

- WoodGate has a Go API, React-admin frontend, and native companion app. Keep platform details at the edge and workflow rules in application or domain code.
- The OpenAPI contract, generated clients, handlers, and consumers change together.
- Real identities, tenant data, credentials, and check-in records never belong in tests.
