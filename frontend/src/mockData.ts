import type { WorkspaceData } from './types';

// Safe bootstrap state shown only while the Wails workspace is loading.
export const fallbackWorkspace: WorkspaceData = {
  appInfo: {
    companyName: 'raml-dev',
    productName: 'marktyp',
    productVersion: 'dev',
    license: 'GNU AGPL-3.0-only license',
    docsLink: '',
    ghLink: 'https://github.com/raml-dev/marktyp',
    orgLink: 'https://github.com/raml-dev',
  },
  config: {
    version: 1,
    preferredMode: 'Document',
    lastOpenedPath: '',
    autosave: false,
    theme: 'marktyp',
    checkForUpdates: true,
    includePrereleaseUpdates: false,
  },
  notes: [],
  activeDoc: {
    id: 'untitled',
    title: 'Untitled',
    path: '',
    markdown: '# Untitled\n\n',
  },
};
