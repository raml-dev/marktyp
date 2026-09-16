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
  companyName: string;
  productName: string;
  productVersion: string;
  license: string;
  docsLink: string;
  ghLink: string;
  orgLink: string;
};

export type AppConfig = {
  version: number;
  preferredMode: string;
  lastOpenedPath: string;
  autosave: boolean;
  theme: AppTheme;
  checkForUpdates: boolean;
  includePrereleaseUpdates: boolean;
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

export type RenameNoteRequest = {
  path: string;
  title: string;
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
  checkForUpdates: boolean;
  includePrereleaseUpdates: boolean;
};

export type GitHubRelease = {
  body: string;
  created_at: string;
  html_url: string;
  updated_at: string;
  name: string;
  tag_name: string;
  prerelease: boolean;
};

export type UpdateResponse = {
  Release?: GitHubRelease | null;
};
