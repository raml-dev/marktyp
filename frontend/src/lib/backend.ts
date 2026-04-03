import type {
  AppConfig,
  DocumentState,
  ExportHTMLRequest,
  ExportPDFRequest,
  SaveDocumentRequest,
  UpdatePreferencesRequest,
  WorkspaceData,
} from '../types';
import {
  DeleteNote,
  ExportHTML,
  ExportPDF,
  GetDocument,
  GetWorkspace,
  NewDocument,
  OpenDocument,
  OpenDocumentAtPath,
  RevealNoteInFS,
  SaveDocument,
  SaveDocumentAs,
  UpdatePreferences,
} from '../../wailsjs/go/app/App';

export async function getWorkspace(): Promise<WorkspaceData> {
  return GetWorkspace();
}

export async function newDocument(): Promise<DocumentState> {
  return NewDocument();
}

export async function openDocument(): Promise<WorkspaceData> {
  return OpenDocument();
}

export async function openDocumentAtPath(path: string): Promise<WorkspaceData> {
  return OpenDocumentAtPath(path);
}

export async function deleteNote(path: string): Promise<WorkspaceData> {
  return DeleteNote(path);
}

export async function revealNoteInFS(path: string): Promise<void> {
  return RevealNoteInFS(path);
}

export async function getDocument(path: string): Promise<DocumentState> {
  return GetDocument(path);
}

export async function saveDocument(request: SaveDocumentRequest): Promise<WorkspaceData> {
  return SaveDocument(request);
}

export async function saveDocumentAs(request: SaveDocumentRequest): Promise<WorkspaceData> {
  return SaveDocumentAs(request);
}

export async function exportHTML(request: ExportHTMLRequest): Promise<void> {
  return ExportHTML(request);
}

export async function exportPDF(request: ExportPDFRequest): Promise<void> {
  return ExportPDF(request);
}

export async function updatePreferences(request: UpdatePreferencesRequest): Promise<AppConfig> {
  return UpdatePreferences(request);
}
