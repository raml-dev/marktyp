# marktyp

marktyp is a desktop Markdown editor built with Wails, Go, Svelte, and TypeScript. It combines a visual document editor with direct Markdown editing, so you can write in a richer interface without losing the underlying `.md` file.

## What It Does

- Opens and saves local Markdown files
- Keeps a small local note library with the last opened documents
- Supports three editing modes: `Document`, `Source`, and `Dual`
- Renders code blocks, Mermaid diagrams, inline math, and block math
- Exports documents to HTML and PDF
- Includes autosave, theme switching, and native desktop menus

## Install

Prerequisites:

- Go `1.23+`
- Node.js and npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

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

## Notes

- App state is stored in the user config directory under `marktyp`
- PDF export uses a headless Chromium-based browser when available
