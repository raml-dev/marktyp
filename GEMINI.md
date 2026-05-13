# marktyp: Project Context & Guidelines

`marktyp` is a desktop Markdown editor built with **Wails v2**, **Go**, **Svelte**, and **TypeScript**. It offers a hybrid editing experience with visual, source-level, and dual-pane modes.

## 🏗️ Architecture Overview

- **Backend (Go):** Located in `internal/app/`.
    - `App`: The main Wails binding point, handling startup and event orchestration.
    - `Store`: Manages application state, user configuration, and the local note library (stored in the user's config directory).
    - `Exporter`: Handles document conversion to HTML and PDF (utilizing headless browsers for high-fidelity PDF output).
- **Frontend (Svelte/TS):** Located in `frontend/src/`.
    - `App.svelte`: The primary UI component managing editor modes, toolbars, and the sidebar.
    - `lib/markdown.ts`: Core logic for Markdown-to-HTML and HTML-to-Markdown conversion.
    - `lib/backend.ts`: Abstracted bridge to Wails-generated Go functions.
- **Inter-process Communication:** Wails provides the bridge between Go and Svelte via `wailsjs` bindings and native event emissions (`EmitMenuAction`).

## 🚀 Key Commands

- **Development:** `wails dev` (starts the Go backend with a hot-reloading Svelte frontend).
- **Build:** `wails build` (produces a production-ready desktop binary).
- **Frontend Setup:** `cd frontend && npm install` (required before the first run).

## 🛠️ Development Conventions

### Backend (Go)
- **State Persistence:** Logic for saving notes and preferences should reside in `internal/app/store.go`.
- **Wails Bindings:** Any new public method on the `App` struct in `internal/app/` will be automatically exposed to the frontend after running `wails dev` or `wails generate module`.

### Frontend (Svelte)
- **Editor Modes:** The application supports `Document` (visual), `Source` (raw Markdown), and `Dual` (split-screen) modes.
- **Visual Editing:** The `Document` mode uses a `contenteditable` container. Logic for syncing this back to Markdown is handled in `syncDocumentEditorToMarkdown`.
- **Rich Content:** Mermaid diagrams and KaTeX math are rendered via client-side libraries and cached for performance.

### Project Standards
- **Styling:** Vanilla CSS is preferred. The app supports themes (`marktyp`, `light`, `dark`).
- **Data Storage:** Configuration and note metadata are stored in `~/.config/marktyp/` (or platform equivalent).
- **Tests:** Backend logic (like the Store) includes unit tests (e.g., `store_delete_test.go`).

## 📝 Usage for AI Interactions

- When modifying **Markdown parsing**, focus on `frontend/src/lib/markdown.ts`.
- When adding **filesystem features**, update `internal/app/store.go` and the `App` struct.
- When updating the **UI/UX**, focus on `frontend/src/App.svelte` and ensure compatibility across all three editor modes.
