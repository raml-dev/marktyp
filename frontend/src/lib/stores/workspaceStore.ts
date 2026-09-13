import { get, writable } from 'svelte/store';
import { EventsOn, BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import * as backend from '../backend';
import { fallbackWorkspace } from '../../mockData';
import { titleFromMarkdown } from '../utils/markdown';
import { BoundedCache } from '../utils/boundedCache';
import type { AppTheme, EditorMode, WorkspaceData, UpdatePreferencesRequest } from '../../types';

export function normalizeMode(value?: string | null): EditorMode {
  const mode = value?.trim().toLowerCase();
  return mode === 'source' ? 'Source' : mode === 'dual' ? 'Dual' : 'Document';
}

// Svelte 4-compatible source of truth. Migrate to runes when the framework is upgraded.
export const editorState = writable({
  workspace: structuredClone(fallbackWorkspace),
  documentState: structuredClone(fallbackWorkspace.activeDoc),
  mode: 'Document' as EditorMode,
  markdown: fallbackWorkspace.activeDoc.markdown,
  sourceDraft: fallbackWorkspace.activeDoc.markdown,
  isDirty: false,
  statusMessage: 'Ready',
  noteQuery: '',
  loading: false,
  saving: false,
  exporting: false,
});
let documentGeneration = 0;
let saveQueue: Promise<unknown> = Promise.resolve();
let preferenceQueue: Promise<unknown> = Promise.resolve();
let draftTimer: ReturnType<typeof setTimeout> | undefined;
let autosaveTimer: ReturnType<typeof setTimeout> | undefined;
let disposed = false;
let blockedAutosaveText: string | undefined;
const imagePreviewCache = new BoundedCache<string, Promise<string>>(48);

function fallbackTitle(path: string): string {
  const filename = path.split(/[\\/]/).pop() ?? '';
  return filename.replace(/\.[^.]+$/, '') || 'Untitled';
}
function title(markdown: string, path: string): string {
  return titleFromMarkdown(markdown, fallbackTitle(path));
}

function message(statusMessage: string) {
  editorState.update((state) => ({ ...state, statusMessage }));
}
function cancelTimers() {
  clearTimeout(draftTimer);
  clearTimeout(autosaveTimer);
}
function applyWorkspace(workspace: WorkspaceData, statusMessage: string) {
  cancelTimers();
  blockedAutosaveText = undefined;
  documentGeneration++;
  const doc = workspace.activeDoc;
  const hasDraft = doc.hasDraft ?? !!doc.draftMarkdown;
  const markdown = hasDraft ? (doc.draftMarkdown ?? '') : doc.markdown;
  editorState.update((state) => ({
    ...state,
    workspace,
    documentState: { ...doc, title: title(markdown, doc.path) },
    markdown,
    sourceDraft: markdown,
    isDirty: hasDraft,
    statusMessage: hasDraft ? 'Draft loaded' : statusMessage,
  }));
}
function enqueue<T>(operation: () => Promise<T>): Promise<T> {
  const next = saveQueue.then(operation, operation);
  saveQueue = next.catch(() => undefined);
  return next;
}
async function preserveDraft(): Promise<boolean> {
  cancelTimers();
  const state = get(editorState);
  if (state.sourceDraft === state.documentState.markdown && !state.isDirty) return true;
  if (!state.documentState.path) {
    return window.confirm('Discard changes to the unsaved document?');
  }
  const path = state.documentState.path;
  const text = state.sourceDraft;
  await enqueue(() => backend.saveDraft(path, text));
  return true;
}
async function navigate(operation: () => Promise<WorkspaceData>, label: string) {
  if (get(editorState).loading) return false;
  editorState.update((state) => ({ ...state, loading: true }));
  try {
    if (!(await preserveDraft())) return false;
    const workspace = await enqueue(operation);
    if (disposed) return false;
    applyWorkspace(workspace, label);
    return true;
  } catch (error) {
    message(String(error));
    return false;
  } finally {
    editorState.update((state) => ({ ...state, loading: false }));
  }
}
async function save(as = false, automatic = false) {
  const state = get(editorState);
  if (state.saving || state.loading || (automatic && !state.documentState.path)) return;
  cancelTimers();
  const generation = documentGeneration;
  const request = {
    path: state.documentState.path,
    title: title(state.sourceDraft, state.documentState.path),
    markdown: state.sourceDraft,
  };
  editorState.update((current) => ({ ...current, saving: true }));
  try {
    const workspace = await enqueue(() =>
      as ? backend.saveDocumentAs(request) : backend.saveDocument(request),
    );
    if (disposed || generation !== documentGeneration) return;
    editorState.update((current) => {
      const unchanged = current.sourceDraft === request.markdown;
      return {
        ...current,
        workspace: {
          ...workspace,
          config: {
            ...workspace.config,
            preferredMode: current.workspace.config.preferredMode,
            autosave: current.workspace.config.autosave,
            theme: current.workspace.config.theme,
          },
        },
        documentState: {
          ...workspace.activeDoc,
          title: title(current.sourceDraft, workspace.activeDoc.path),
        },
        markdown: unchanged ? request.markdown : current.markdown,
        isDirty: !unchanged,
        statusMessage: automatic ? 'Autosaved' : 'Document saved',
      };
    });
  } catch (error) {
    if (automatic) blockedAutosaveText = request.markdown;
    message(`Save failed: ${String(error)}`);
  } finally {
    editorState.update((current) => ({ ...current, saving: false }));
  }
}
function preferences(patch: Partial<UpdatePreferencesRequest>) {
  preferenceQueue = preferenceQueue
    .then(async () => {
      const state = get(editorState);
      const config = await backend.updatePreferences({
        preferredMode: state.mode,
        autosave: state.workspace.config.autosave,
        theme: state.workspace.config.theme,
        ...patch,
      });
      editorState.update((current) => ({ ...current, workspace: { ...current.workspace, config } }));
    })
    .catch((error) => message(`Unable to save preferences: ${String(error)}`));
  return preferenceQueue;
}

export const workspaceStore = {
  editSource(text: string, updatePreview = false, statusMessage = 'Source draft changed') {
    if (text !== blockedAutosaveText) blockedAutosaveText = undefined;
    editorState.update((state) => ({
      ...state,
      sourceDraft: text,
      markdown: updatePreview ? text : state.markdown,
      documentState: {
        ...state.documentState,
        title: title(text, state.documentState.path),
      },
      isDirty: text !== state.documentState.markdown,
      statusMessage,
    }));
  },
  updatePreview(text: string) {
    editorState.update((state) =>
      state.mode === 'Dual' && state.sourceDraft === text ? { ...state, markdown: text } : state,
    );
  },
  applySource() {
    const state = get(editorState);
    workspaceStore.editSource(state.sourceDraft, true, 'Source applied to document');
  },
  revertSource() {
    const state = get(editorState);
    workspaceStore.editSource(state.markdown, false, 'Source draft reverted');
  },
  changeMode(mode: EditorMode) {
    if (mode === 'Document') workspaceStore.applySource();
    editorState.update((state) => ({ ...state, mode }));
    return preferences({ preferredMode: mode });
  },
  message,
  editDocument(text: string, statusMessage = 'Document changed') {
    workspaceStore.editSource(text, true, statusMessage);
  },
  listen(onAction: (action: string) => void) {
    if (!('runtime' in window)) return () => {};
    const offMenu = EventsOn('marktyp:menu-action', onAction);
    const offClose = EventsOn('marktyp:request-close', () => {
      if (get(editorState).loading || get(editorState).saving || get(editorState).exporting) {
        message('Wait for the current operation before closing');
        return;
      }
      void preserveDraft()
        .then(async (allowed) => {
          if (allowed) await backend.forceQuit();
        })
        .catch((error) => message(`Unable to preserve draft: ${String(error)}`));
    });
    return () => {
      offMenu();
      offClose();
    };
  },
  async selectImage(): Promise<string | null> {
    try {
      return await backend.selectImage();
    } catch (error) {
      message(`Unable to select image: ${String(error)}`);
      return null;
    }
  },
  imagePreview(source: string, documentPath = get(editorState).documentState.path): Promise<string> {
    if (/^(https?:|data:|blob:)/i.test(source)) return Promise.resolve(source);
    const key = `${documentPath}\u0000${source}`;
    const cached = imagePreviewCache.get(key);
    if (cached) return cached;
    const preview = backend.getImagePreview(source, documentPath).catch((error) => {
      imagePreviewCache.delete(key);
      throw error;
    });
    imagePreviewCache.set(key, preview);
    return preview;
  },
  openLink(url: string) {
    if (/^https?:\/\//i.test(url) || /^mailto:/i.test(url)) BrowserOpenURL(url);
  },
  async load() {
    disposed = false;
    editorState.update((state) => ({ ...state, loading: true }));
    try {
      const workspace = await backend.getWorkspace();
      if (disposed) return;
      applyWorkspace(workspace, 'Workspace loaded');
      editorState.update((state) => ({ ...state, mode: normalizeMode(workspace.config.preferredMode) }));
    } catch (error) {
      message(`Workspace unavailable: ${String(error)}`);
    } finally {
      editorState.update((state) => ({ ...state, loading: false }));
    }
  },
  open: () => navigate(backend.openDocument, 'Document opened'),
  select: (path: string) => navigate(() => backend.openDocumentAtPath(path), 'Document selected'),
  async create() {
    return navigate(
      async () => ({ ...get(editorState).workspace, activeDoc: await backend.newDocument() }),
      'New document',
    );
  },
  save,
  preferences,
  theme: (theme: AppTheme) => preferences({ theme }),
  async remove(path: string) {
    if (!path || get(editorState).loading) return false;
    const state = get(editorState);
    if (path !== state.documentState.path) {
      editorState.update((current) => ({ ...current, loading: true }));
      try {
        await saveQueue;
        const workspace = await backend.deleteNote(path);
        editorState.update((current) => ({
          ...current,
          workspace: { ...current.workspace, notes: workspace.notes },
        }));
      } catch (error) {
        message(`Unable to delete note: ${String(error)}`);
      } finally {
        editorState.update((current) => ({ ...current, loading: false }));
      }
      return false;
    }
    cancelTimers();
    return navigateWithoutDraft(() => backend.deleteNote(path));
  },
  scheduleDraft(path: string, text: string, dirty: boolean) {
    clearTimeout(draftTimer);
    if (!path || !dirty || disposed) return;
    draftTimer = setTimeout(() => {
      void enqueue(() => backend.saveDraft(path, text)).catch((error) =>
        message(`Draft failed: ${String(error)}`),
      );
    }, 800);
  },
  scheduleAutosave(enabled: boolean, dirty: boolean, text: string, saving: boolean, loading: boolean) {
    clearTimeout(autosaveTimer);
    if (enabled && dirty && text !== blockedAutosaveText && !saving && !loading && !disposed)
      autosaveTimer = setTimeout(() => void save(false, true), 1200);
  },
  dispose() {
    disposed = true;
    cancelTimers();
    imagePreviewCache.clear();
  },
  async exportDocument(kind: 'html' | 'pdf', path?: string) {
    const state = get(editorState);
    if (state.exporting || state.loading) return;
    const active = !path || path === state.documentState.path;
    const snapshot = {
      title: title(state.sourceDraft, state.documentState.path),
      markdown: state.sourceDraft,
    };
    editorState.update((current) => ({ ...current, exporting: true }));
    try {
      const doc = active ? snapshot : await backend.getDocument(path!);
      const { renderExportHtml } = await import('../utils/exportDocument');
      const html = await renderExportHtml(doc.markdown, (source) =>
        workspaceStore.imagePreview(source, active ? state.documentState.path : path!),
      );
      if (kind === 'html') await backend.exportHTML({ title: doc.title, html });
      else await backend.exportPDF({ title: doc.title, markdown: doc.markdown, html });
      message(`${kind.toUpperCase()} exported`);
    } catch (error) {
      message(`Export failed: ${String(error)}`);
    } finally {
      editorState.update((current) => ({ ...current, exporting: false }));
    }
  },
  async reveal(path: string) {
    try {
      await backend.revealNoteInFS(path);
      message('Opened in file manager');
    } catch (error) {
      message(`Unable to reveal file: ${String(error)}`);
    }
  },
};
async function navigateWithoutDraft(operation: () => Promise<WorkspaceData>) {
  editorState.update((state) => ({ ...state, loading: true }));
  try {
    const workspace = await enqueue(operation);
    if (disposed) return false;
    applyWorkspace(workspace, 'Note deleted');
    return true;
  } catch (error) {
    message(`Unable to delete note: ${String(error)}`);
    return false;
  } finally {
    editorState.update((state) => ({ ...state, loading: false }));
  }
}
