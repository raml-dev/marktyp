import type {
  AppConfig,
  DocumentState,
  ExportHTMLRequest,
  ExportPDFRequest,
  SaveDocumentRequest,
  RenameNoteRequest,
  UpdatePreferencesRequest,
  UpdateResponse,
  WorkspaceData,
} from '../types';
import {
  DeleteNote,
  ForceQuit,
  ExportHTML,
  ExportPDF,
  GetDocument,
  GetImagePreview,
  GetWorkspace,
  GetUpdatesFromRepo,
  NewDocument,
  OpenDocument,
  OpenDocumentAtPath,
  RenameNote,
  RevealNoteInFS,
  SaveDocument,
  SaveDocumentAs,
  SaveDraft,
  SelectImage,
  UpdatePreferences,
} from '../../wailsjs/go/main/App';

function normalizeConfig(config: Omit<AppConfig, 'theme'> & { theme: string }): AppConfig {
  return { ...config, theme: config.theme === 'light' || config.theme === 'dark' ? config.theme : 'marktyp' };
}
function normalizeWorkspace(
  workspace: Omit<WorkspaceData, 'config'> & { config: Parameters<typeof normalizeConfig>[0] },
): WorkspaceData {
  return { ...workspace, notes: workspace.notes ?? [], config: normalizeConfig(workspace.config) };
}

export async function forceQuit(): Promise<void> {
  return ForceQuit();
}

export async function selectImage(): Promise<string> {
  return SelectImage();
}

export async function getImagePreview(source: string, documentPath: string): Promise<string> {
  return GetImagePreview(source, documentPath);
}

export async function getWorkspace(): Promise<WorkspaceData> {
  return normalizeWorkspace(await GetWorkspace());
}

export async function getUpdatesFromRepo(): Promise<UpdateResponse | null> {
  return GetUpdatesFromRepo();
}

export async function newDocument(): Promise<WorkspaceData> {
  return normalizeWorkspace(await NewDocument());
}

export async function openDocument(): Promise<WorkspaceData> {
  return normalizeWorkspace(await OpenDocument());
}

export async function openDocumentAtPath(path: string): Promise<WorkspaceData> {
  return normalizeWorkspace(await OpenDocumentAtPath(path));
}

export async function deleteNote(path: string): Promise<WorkspaceData> {
  return normalizeWorkspace(await DeleteNote(path));
}

export async function renameNote(request: RenameNoteRequest): Promise<WorkspaceData> {
  return normalizeWorkspace(await RenameNote(request));
}

export async function revealNoteInFS(path: string): Promise<void> {
  return RevealNoteInFS(path);
}

export async function getDocument(path: string): Promise<DocumentState> {
  return GetDocument(path);
}

export async function saveDocument(request: SaveDocumentRequest): Promise<WorkspaceData> {
  return normalizeWorkspace(await SaveDocument(request));
}

export async function saveDocumentAs(request: SaveDocumentRequest): Promise<WorkspaceData> {
  return normalizeWorkspace(await SaveDocumentAs(request));
}

export async function saveDraft(path: string, markdown: string): Promise<void> {
  return SaveDraft(path, markdown);
}

export async function exportHTML(request: ExportHTMLRequest): Promise<void> {
  return ExportHTML(request);
}

export async function exportPDF(request: ExportPDFRequest): Promise<void> {
  return ExportPDF({ ...request, html: request.html ?? '' });
}

export async function updatePreferences(request: UpdatePreferencesRequest): Promise<AppConfig> {
  return normalizeConfig(await UpdatePreferences(request));
}
