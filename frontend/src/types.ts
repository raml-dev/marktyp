export type EditorMode = 'Document' | 'Source' | 'Dual';
export type AppTheme = 'marktyp' | 'light' | 'dark';

export type NoteSummary = {
  id: string;
  title: string;
  path: string;
  createdAt: string;
  updatedAt: string;
  updatedLabel: string;
};

export type AppInfo = {
  name: string;
  tagline: string;
  version: string;
  modes: string[];
};

export type AppConfig = {
  version: number;
  preferredMode: string;
  lastOpenedPath: string;
  autosave: boolean;
  theme: AppTheme;
};

export type DocumentState = {
  id: string;
  title: string;
  path: string;
  markdown: string;
  draftMarkdown?: string;
  hasDraft?: boolean;
};

export type WorkspaceData = {
  appInfo: AppInfo;
  config: AppConfig;
  notes: NoteSummary[];
  activeDoc: DocumentState;
};

export type SaveDocumentRequest = {
  path: string;
  title: string;
  markdown: string;
};

export type ExportHTMLRequest = {
  title: string;
  html: string;
};

export type ExportPDFRequest = {
  title: string;
  markdown: string;
  html?: string;
};

export type UpdatePreferencesRequest = {
  preferredMode: string;
  autosave: boolean;
  theme: AppTheme;
};
