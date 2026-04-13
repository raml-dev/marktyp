import type {WorkspaceData} from './types';

export const fallbackWorkspace: WorkspaceData = {
  appInfo: {
    name: 'marktyp',
    tagline: 'Editor documentale visuale con Markdown sotto il cofano',
    version: '0.1.0',
    modes: ['Document', 'Source', 'Dual'],
  },
  config: {
    version: 1,
    preferredMode: 'Document',
    lastOpenedPath: '',
    autosave: false,
    theme: 'marktyp',
  },
  notes: [
    {
      id: 'welcome',
      title: 'Welcome to marktyp',
      path: '~/notes/welcome.md',
      createdAt: '2026-04-01T12:00:00Z',
      updatedAt: '2026-04-01T12:00:00Z',
      updatedLabel: 'Today',
    },
    {
      id: 'roadmap',
      title: 'Roadmap',
      path: '~/notes/roadmap.md',
      createdAt: '2026-03-31T12:00:00Z',
      updatedAt: '2026-03-31T12:00:00Z',
      updatedLabel: 'Yesterday',
    },
    {
      id: 'writing',
      title: 'Writing principles',
      path: '~/notes/writing.md',
      createdAt: '2026-03-30T12:00:00Z',
      updatedAt: '2026-03-30T12:00:00Z',
      updatedLabel: '2026-03-30',
    },
  ],
  activeDoc: {
    id: 'welcome',
    title: 'Welcome to marktyp',
    path: '~/notes/welcome.md',
    markdown: `# Welcome to marktyp

Marktyp is starting as a clean document editor.

## What is already here

- A desktop shell with Wails
- A React frontend
- A first TipTap editor surface
- Document, Source and Dual modes

## Next steps

- File open and save
- Local notes archive
- Mermaid blocks
- Inline and block math
`,
  },
};
