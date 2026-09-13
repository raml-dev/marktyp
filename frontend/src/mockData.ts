import type { WorkspaceData } from './types';

// Safe bootstrap state shown only while the Wails workspace is loading.
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
  notes: [],
  activeDoc: {
    id: 'untitled',
    title: 'Untitled',
    path: '',
    markdown: '# Untitled\n\n',
  },
};
