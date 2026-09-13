<!--
 Copyright 2026-present raml-dev
 SPDX-License-Identifier: AGPL-3.0-only
-->
# AGENTS.md

Directives for coding agents working on Marktyp, a Wails v2 desktop Markdown editor.

## Stack

Go backend; Svelte 4 and strict TypeScript frontend; Wails-generated IPC bindings.
Preserve Marktyp's existing CSS, branding, themes and Document/Source/Dual modes.

## Skills

Read the reference skills before working on the corresponding layer:
- Backend: `../yapla/.agents/skills/solo-backend-skill/SKILL.md`
- Frontend: `../yapla/.agents/skills/solo-frontend-skill/SKILL.md`

Apply their separation of concerns: a thin root `app.go` RPC facade, domain services
under `internal/`, centralized state and operations under `frontend/src/lib/stores/` (use Svelte 4 stores until the framework is upgraded),
focused UI under `lib/components/`, and reusable helpers under `lib/utils/`.
Solo-specific domains and UI libraries are not requirements for Marktyp: preserve
existing visuals and dependencies instead of replacing the UI with Flowbite.
If MCP tools are unavailable, explicitly report that limitation and use local checks.

## Change authorization

The requested full refactor authorizes file creation/removal, package moves and
component contract changes on the refactor branch. Preserve data formats and all
existing functionality. Outside that scope, ask before structural/API changes.
Never overwrite unrelated user changes (including untracked `hack/`).

## Hard rules

- Never commit directly to `main`.
- Never hand-edit generated `frontend/wailsjs/` files; regenerate with Wails.
- Do not add dependencies without approval.
- Do not modify branding, icons or app assets.
- Keep business logic and IPC out of presentation components.
- Clean up timers/listeners; protect unsaved edits against asynchronous races.
- Keep persistence compatible and guard uninitialized backend services.
- Use standard Go tests alongside source files and structured logging.

## Verification

Do not launch `wails dev` from the agent. Run:

```sh
go test ./...
go test -race ./...
go build ./...
cd frontend
npm run format
npm run check
npm run lint
npm run build
```

Use `wails build` to regenerate bindings and verify desktop packaging. Report any
unavailable tooling or unverified native UI behavior; do not claim visual testing
from compilation alone.
