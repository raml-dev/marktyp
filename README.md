# marktyp

marktyp is a desktop Markdown editor built with Wails, Go, Svelte, and TypeScript. It combines a visual document editor with direct Markdown editing, so you can write in a richer interface without losing the underlying `.md` file.

## What It Does

- Opens and saves local Markdown files
- Keeps a small local note library with the last opened documents
- Supports three editing modes: `Document`, `Source`, and `Dual`
- Renders and edits code blocks, tables, Mermaid diagrams, inline math, and block math
- Exports documents to HTML and PDF
- Includes autosave, theme switching, and native desktop menus

## Install

Prerequisites:

- Go `1.26.1+`
- Node.js and npm
- Wails CLI matching `go.mod` (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`)

Install dependencies:

```bash
cd frontend
npm install
cd ..
```

Run in development:

```bash
wails dev
```

Build the app:

```bash
wails build
```

## Architecture

- `app.go` is the thin Wails RPC facade
- `internal/document`, `internal/exporter`, and `internal/tools` contain backend logic
- `frontend/src/lib/stores` owns frontend state, persistence, and Wails calls
- `frontend/src/lib/components` contains focused UI components
- `frontend/src/lib/utils` contains Markdown, rendering, and safety helpers

## Verification

```bash
go test -race ./...
go vet ./...
go build ./...
cd frontend
npm run format
npm run check
npm run lint
npm run build
npm test
```

## Notes

- App state is stored in the user config directory under `marktyp`
- PDF export uses a headless Chromium-based browser when available and falls back to the built-in PDF writer
