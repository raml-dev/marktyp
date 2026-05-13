<script lang="ts">
  import {onMount, tick} from 'svelte';
  import katex from 'katex';
  import hljs from 'highlight.js';
  import {EventsOn} from '../wailsjs/runtime/runtime';
  import {fallbackWorkspace} from './mockData';
  import {
    deleteNote,
    exportHTML,
    exportPDF,
    getDocument,
    getWorkspace,
    newDocument,
    openDocument,
    openDocumentAtPath,
    revealNoteInFS,
    saveDocument,
    saveDocumentAs,
    saveDraft,
    updatePreferences,
  } from './lib/backend';
  import {htmlToMarkdown, markdownToHtml, titleFromMarkdown} from './lib/markdown';
  import type {AppTheme, DocumentState, EditorMode, NoteSummary, WorkspaceData} from './types';

  type NoteContextMenuState = {
    note: NoteSummary;
    x: number;
    y: number;
  };

  type RichEditorKind = 'math-inline' | 'math-block' | 'mermaid';

  let workspace: WorkspaceData = fallbackWorkspace;
  let documentState: DocumentState = fallbackWorkspace.activeDoc;
  let mode: EditorMode = 'Document';
  let markdown = fallbackWorkspace.activeDoc.markdown;
  let sourceDraft = fallbackWorkspace.activeDoc.markdown;
  let isDirty = false;
  let statusMessage = 'Ready';
  let noteQuery = '';
  let noteContextMenu: NoteContextMenuState | null = null;
  let documentPreviewEl: HTMLElement | null = null;
  let sourceTextarea: HTMLTextAreaElement | null = null;
  let sourceHistory: string[] = [sourceDraft];
  let sourceHistoryIndex = 0;
  let applyingSourceHistory = false;
  let autosaveTimer: number | null = null;
  let openMenu: string | null = null;
  let mermaidInitialized = false;
  let mermaidApi: {initialize: Function; render: Function} | null = null;
  let mermaidRenderTimer: number | null = null;
  let mermaidRenderGeneration = 0;
  let dualPreviewTimer: number | null = null;
  const mathInlineCache = new Map<string, string>();
  const mathBlockCache = new Map<string, string>();
  const mermaidSvgCache = new Map<string, string>();
  let lastSourceSelectionStart = 0;
  let lastSourceSelectionEnd = 0;
  let previewRenderScheduled = false;
  let isEditingDocument = false;
  let documentHtmlDraft = markdownToHtml(markdown);
  let previewDecorationsHtml = documentHtmlDraft;
  let lastDocumentRange: Range | null = null;
  let richEditorOpen = false;
  let richEditorKind: RichEditorKind = 'math-inline';
  let richEditorSource = '';
  let richEditorTarget: HTMLElement | null = null;
  let markdownHistory: string[] = [markdown];
  let markdownHistoryIndex = 0;
  let applyingMarkdownHistory = false;
  let documentHistory: string[] = [];
  let documentHistoryIndex = -1;
  let applyingDocumentHistory = false;
  let documentEditorChanged = false;
  let suppressNextDocumentBlur = false;
  let appMainEl: HTMLElement | null = null;
  let dualBodyEl: HTMLElement | null = null;
  let sidebarCollapsed = false;
  let sidebarWidth = 320;
  let dualSplit = 58;
  let resizingSidebar = false;
  let resizingDual = false;
  let draftTimer: number | null = null;

  function normalizeMode(value: string | null | undefined): EditorMode {
    const raw = (value || '').trim().toLowerCase();
    if (raw === 'source') {
      return 'Source';
    }
    if (raw === 'dual') {
      return 'Dual';
    }
    return 'Document';
  }

  $: hasSourceDraftChanges = sourceDraft !== markdown;
  $: activeNote = workspace.notes.find((note) => note.path === documentState.path) ?? null;
  $: filteredNotes = (() => {
    const query = noteQuery.trim().toLowerCase();
    if (!query) {
      return workspace.notes;
    }

    return workspace.notes.filter((note) => {
      return note.title.toLowerCase().includes(query) || note.path.toLowerCase().includes(query);
    });
  })();
  $: statusLabel = hasSourceDraftChanges ? 'Source draft pending' : isDirty ? 'Unsaved changes' : statusMessage;
  $: renderedDocumentHtml = markdownToHtml(markdown);
  $: showDocumentPanel = mode === 'Document' || mode === 'Dual';
  $: showSourcePanel = mode === 'Source' || mode === 'Dual';
  $: if (!isEditingDocument) {
    documentHtmlDraft = renderedDocumentHtml;
  }
  $: previewDecorationsHtml = mode === 'Document' ? documentHtmlDraft : renderedDocumentHtml;
  $: if (documentPreviewEl && previewDecorationsHtml !== undefined && !(mode === 'Document' && isEditingDocument)) {
    schedulePreviewDecorations();
  }

  $: {
    document.documentElement.dataset.theme = workspace.config.theme || 'marktyp';
  }

  $: {
    if (draftTimer) {
      window.clearTimeout(draftTimer);
      draftTimer = null;
    }

    if (isDirty && documentState.path && documentState.path.trim() !== '') {
      draftTimer = window.setTimeout(async () => {
        try {
          await saveDraft(documentState.path, markdown);
        } catch {
          console.error('Failed to save draft');
        }
      }, 800);
    }
  }

  $: {
    if (autosaveTimer) {
      window.clearTimeout(autosaveTimer);
      autosaveTimer = null;
    }

    if (workspace.config.autosave && isDirty && !hasSourceDraftChanges && documentState.path) {
      autosaveTimer = window.setTimeout(async () => {
        try {
          const nextWorkspace = await saveDocument({
            path: documentState.path,
            title: documentState.title,
            markdown,
          });
          applyWorkspace(nextWorkspace, 'Autosaved');
        } catch {
          statusMessage = 'Autosave failed';
        }
      }, 1200);
    }
  }

  onMount(() => {
    void loadWorkspace();

    const unsubscribe = EventsOn('marktyp:menu-action', (action: string) => {
      switch (action) {
        case 'file:new':
          void handleNewDocument();
          break;
        case 'file:open':
          void handleOpenDocument();
          break;
        case 'file:save':
          void handleSaveDocument();
          break;
        case 'file:save-as':
          void handleSaveDocumentAs();
          break;
        case 'file:delete':
          void handleDeleteActiveNote();
          break;
        case 'export:html':
          void handleExportHTML();
          break;
        case 'export:pdf':
          void handleExportPDF();
          break;
        case 'view:document':
          void handleModeChange('Document');
          break;
        case 'view:source':
          void handleModeChange('Source');
          break;
        case 'view:dual':
          void handleModeChange('Dual');
          break;
        case 'view:toggle-autosave':
          void handleToggleAutosave();
          break;
        case 'settings:theme-marktyp':
          void handleThemeChange('marktyp');
          break;
        case 'settings:theme-light':
          void handleThemeChange('light');
          break;
        case 'settings:theme-dark':
          void handleThemeChange('dark');
          break;
      }
    });

    function onPointerDown(event: PointerEvent) {
      const target = event.target as Element | null;
      if (target?.closest('.note-context-menu') || target?.closest('.toolbar-menu')) {
        return;
      }
      noteContextMenu = null;
      openMenu = null;
    }

    function onPointerMove(event: PointerEvent) {
      if (resizingSidebar && appMainEl && !sidebarCollapsed) {
        const rect = appMainEl.getBoundingClientRect();
        sidebarWidth = clamp(event.clientX - rect.left, 220, 520);
      }

      if (resizingDual && dualBodyEl && mode === 'Dual') {
        const rect = dualBodyEl.getBoundingClientRect();
        dualSplit = clamp(((event.clientX - rect.left) / rect.width) * 100, 32, 78);
      }
    }

    function onPointerUp() {
      if (resizingSidebar || resizingDual) {
        resizingSidebar = false;
        resizingDual = false;
        document.body.classList.remove('is-resizing');
      }
    }

    function onResizeOrScroll() {
      noteContextMenu = null;
      openMenu = null;
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        noteContextMenu = null;
        openMenu = null;
      }

      const hasPrimaryModifier = event.ctrlKey || event.metaKey;
      const key = event.key.toLowerCase();
      const wantsUndo = key === 'z' && !event.shiftKey;
      const wantsRedo = (key === 'z' && event.shiftKey) || key === 'y';
      if (mode === 'Document' && hasPrimaryModifier && (wantsUndo || wantsRedo)) {
        const target = event.target as HTMLElement | null;
        const tag = target?.tagName?.toLowerCase() || '';
        const isFormControl = tag === 'input' || tag === 'textarea';
        if (!isFormControl) {
          event.preventDefault();
          event.stopPropagation();
          wantsUndo ? performDocumentUndo() : performDocumentRedo();
          return;
        }
      }

      if (handleEditorHistoryShortcut(event)) {
        return;
      }

      const isSaveKey = event.key.toLowerCase() === 's' && (event.ctrlKey || event.metaKey);
      if (isSaveKey) {
        event.preventDefault();
        void handleSaveFromShortcut();
      }
    }

    function onSelectionChange() {
      const selection = window.getSelection();
      if (!selection || selection.rangeCount === 0 || !documentPreviewEl) {
        return;
      }
      const range = selection.getRangeAt(0);
      if (!documentPreviewEl.contains(range.commonAncestorContainer)) {
        return;
      }
      lastDocumentRange = range.cloneRange();
    }

    window.addEventListener('pointerdown', onPointerDown);
    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', onPointerUp);
    window.addEventListener('resize', onResizeOrScroll);
    window.addEventListener('scroll', onResizeOrScroll, true);
    window.addEventListener('keydown', onKeyDown, true);
    document.addEventListener('selectionchange', onSelectionChange);

    return () => {
      unsubscribe();
      if (dualPreviewTimer) {
        window.clearTimeout(dualPreviewTimer);
        dualPreviewTimer = null;
      }
      if (mermaidRenderTimer) {
        window.clearTimeout(mermaidRenderTimer);
        mermaidRenderTimer = null;
      }
      window.removeEventListener('pointerdown', onPointerDown);
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      window.removeEventListener('pointercancel', onPointerUp);
      window.removeEventListener('resize', onResizeOrScroll);
      window.removeEventListener('scroll', onResizeOrScroll, true);
      window.removeEventListener('keydown', onKeyDown, true);
      document.removeEventListener('selectionchange', onSelectionChange);
    };
  });

  function clamp(value: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, value));
  }

  function isDocumentEditorFocused(): boolean {
    if (mode !== 'Document' || !documentPreviewEl) {
      return false;
    }

    const active = document.activeElement as Node | null;
    if (active && documentPreviewEl.contains(active)) {
      return true;
    }

    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) {
      return false;
    }

    return documentPreviewEl.contains(selection.getRangeAt(0).commonAncestorContainer);
  }

  function resetMarkdownHistory(seed: string) {
    markdownHistory = [seed];
    markdownHistoryIndex = 0;
  }

  function recordMarkdownHistory(next: string) {
    if (applyingMarkdownHistory) {
      return;
    }

    if (markdownHistoryIndex < 0 || markdownHistory.length === 0) {
      markdownHistory = [next];
      markdownHistoryIndex = 0;
      return;
    }

    if (markdownHistory[markdownHistoryIndex] === next) {
      return;
    }

    const snapshots = markdownHistory.slice(0, markdownHistoryIndex + 1);
    snapshots.push(next);
    if (snapshots.length > 300) {
      snapshots.shift();
    }
    markdownHistory = snapshots;
    markdownHistoryIndex = markdownHistory.length - 1;
  }

  function applyMarkdownSnapshot(next: string) {
    applyingMarkdownHistory = true;
    sourceDraft = next;
    markdown = next;
    documentState = {...documentState, title: titleFromMarkdown(next)};
    isDirty = next !== documentState.markdown;
    statusMessage = 'Document changed';
    documentHtmlDraft = markdownToHtml(next);
    if (mode === 'Document' && documentPreviewEl) {
      documentPreviewEl.innerHTML = documentHtmlDraft;
      renderPreviewDecorations(documentPreviewEl);
    }
    applyingMarkdownHistory = false;
  }

  function resetDocumentHistory() {
    if (!documentPreviewEl) {
      documentHistory = [];
      documentHistoryIndex = -1;
      return;
    }

    const html = documentPreviewEl.innerHTML;
    documentHistory = [html];
    documentHistoryIndex = 0;
  }

  function recordDocumentHistory() {
    if (!documentPreviewEl || applyingDocumentHistory || mode !== 'Document') {
      return;
    }

    const html = documentPreviewEl.innerHTML;
    if (documentHistoryIndex < 0) {
      documentHistory = [html];
      documentHistoryIndex = 0;
      return;
    }
    if (documentHistoryIndex >= 0 && documentHistory[documentHistoryIndex] === html) {
      return;
    }

    const nextHistory = documentHistory.slice(0, documentHistoryIndex + 1);
    nextHistory.push(html);
    if (nextHistory.length > 200) {
      nextHistory.shift();
    }

    documentHistory = nextHistory;
    documentHistoryIndex = documentHistory.length - 1;
  }

  function applyDocumentHistorySnapshot(index: number) {
    if (!documentPreviewEl || index < 0 || index >= documentHistory.length) {
      return;
    }

    applyingDocumentHistory = true;
    documentHistoryIndex = index;
    documentPreviewEl.innerHTML = documentHistory[index];
    renderPreviewDecorations(documentPreviewEl);
    syncDocumentEditorToMarkdown('Document changed');
    applyingDocumentHistory = false;
  }

  function performDocumentUndo(): boolean {
    if (documentHistoryIndex <= 0) {
      return false;
    }
    applyDocumentHistorySnapshot(documentHistoryIndex - 1);
    return true;
  }

  function performDocumentRedo(): boolean {
    if (documentHistoryIndex < 0 || documentHistoryIndex >= documentHistory.length - 1) {
      return false;
    }
    applyDocumentHistorySnapshot(documentHistoryIndex + 1);
    return true;
  }

  function handleEditorHistoryShortcut(event: KeyboardEvent): boolean {
    const hasPrimaryModifier = event.ctrlKey || event.metaKey;
    if (!hasPrimaryModifier || !isDocumentEditorFocused()) {
      return false;
    }

    const key = event.key.toLowerCase();
    const wantsUndo = key === 'z' && !event.shiftKey;
    const wantsRedo = (key === 'z' && event.shiftKey) || key === 'y';
    if (!wantsUndo && !wantsRedo) {
      return false;
    }

    const handled = wantsUndo ? performDocumentUndo() : performDocumentRedo();
    if (!handled) {
      return false;
    }

    event.preventDefault();
    event.stopPropagation();
    return true;
  }

  function handleDocumentKeyDown(event: KeyboardEvent) {
    if (mode !== 'Document') {
      return;
    }

    const hasPrimaryModifier = event.ctrlKey || event.metaKey;
    if (!hasPrimaryModifier) {
      return;
    }

    const key = event.key.toLowerCase();
    const wantsUndo = key === 'z' && !event.shiftKey;
    const wantsRedo = (key === 'z' && event.shiftKey) || key === 'y';
    if (!wantsUndo && !wantsRedo) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    const handled = wantsUndo ? performDocumentUndo() : performDocumentRedo();
    if (!handled) {
      const command = wantsUndo ? 'undo' : 'redo';
      const nativeApplied = document.execCommand(command);
      if (nativeApplied) {
        syncDocumentEditorToMarkdown('Document changed');
        recordDocumentHistory();
      }
    }
  }

  function handleDocumentBeforeInput(event: InputEvent) {
    if (mode !== 'Document') {
      return;
    }

    if (event.inputType === 'historyUndo' || event.inputType === 'historyRedo') {
      event.preventDefault();
    }
  }

  function toggleSidebar() {
    sidebarCollapsed = !sidebarCollapsed;
  }

  $: sidebarColumnWidth = sidebarCollapsed ? 46 : Math.round(sidebarWidth);

  function startSidebarResize(event: PointerEvent) {
    if (sidebarCollapsed) {
      return;
    }
    event.preventDefault();
    resizingSidebar = true;
    document.body.classList.add('is-resizing');
  }

  function startDualResize(event: PointerEvent) {
    if (mode !== 'Dual') {
      return;
    }
    event.preventDefault();
    resizingDual = true;
    document.body.classList.add('is-resizing');
  }

  async function loadWorkspace() {
    try {
      const nextWorkspace = await getWorkspace();
      workspace = nextWorkspace;
      documentState = nextWorkspace.activeDoc;
      const initialMarkdown = nextWorkspace.activeDoc.draftMarkdown || nextWorkspace.activeDoc.markdown;
      markdown = initialMarkdown;
      sourceDraft = initialMarkdown;
      isDirty = !!nextWorkspace.activeDoc.draftMarkdown;
      mode = normalizeMode(nextWorkspace.config.preferredMode);
      resetSourceHistory(initialMarkdown || '');
      resetMarkdownHistory(initialMarkdown || '');
      documentEditorChanged = false;
      statusMessage = nextWorkspace.activeDoc.draftMarkdown ? 'Draft loaded' : 'Workspace loaded';
    } catch {
      resetSourceHistory(fallbackWorkspace.activeDoc.markdown || '');
      resetMarkdownHistory(fallbackWorkspace.activeDoc.markdown || '');
      statusMessage = 'Using local fallback data';
    }
  }

  function applyWorkspace(nextWorkspace: WorkspaceData, nextMessage: string) {
    workspace = nextWorkspace;
    documentState = nextWorkspace.activeDoc;
    const initialMarkdown = nextWorkspace.activeDoc.draftMarkdown || nextWorkspace.activeDoc.markdown;
    markdown = initialMarkdown;
    sourceDraft = initialMarkdown;
    isDirty = !!nextWorkspace.activeDoc.draftMarkdown;
    noteContextMenu = null;
    statusMessage = nextWorkspace.activeDoc.draftMarkdown ? 'Draft loaded' : nextMessage;
    resetSourceHistory(initialMarkdown || '');
    resetMarkdownHistory(initialMarkdown || '');
    documentHistory = [];
    documentHistoryIndex = -1;
    documentEditorChanged = false;
  }

  async function handleNewDocument() {
    try {
      const freshDocument = await newDocument();
      documentState = freshDocument;
      markdown = freshDocument.markdown;
      sourceDraft = freshDocument.markdown;
      isDirty = false;
      statusMessage = 'New document';
      mode = 'Document';
      resetSourceHistory(freshDocument.markdown || '');
      resetMarkdownHistory(freshDocument.markdown || '');
      documentHistory = [];
      documentHistoryIndex = -1;
      documentEditorChanged = false;
    } catch {
      statusMessage = 'Unable to create a new document';
    }
  }

  async function handleOpenDocument() {
    try {
      const nextWorkspace = await openDocument();
      applyWorkspace(nextWorkspace, 'Document opened');
    } catch {
      statusMessage = 'Open cancelled or failed';
    }
  }

  async function handleSaveDocument() {
    try {
      const nextWorkspace = await saveDocument({
        path: documentState.path,
        title: documentState.title,
        markdown,
      });
      applyWorkspace(nextWorkspace, 'Document saved');
    } catch {
      statusMessage = 'Save failed';
    }
  }

  async function handleSaveFromShortcut() {
    const nextMarkdown = hasSourceDraftChanges ? sourceDraft : markdown;
    const nextTitle = titleFromMarkdown(nextMarkdown);

    if (hasSourceDraftChanges) {
      markdown = nextMarkdown;
      documentState = {...documentState, title: nextTitle};
      isDirty = nextMarkdown !== documentState.markdown;
    }

    const request = {
      path: documentState.path,
      title: nextTitle,
      markdown: nextMarkdown,
    };

    try {
      const nextWorkspace =
        request.path && request.path.trim() !== '' ? await saveDocument(request) : await saveDocumentAs(request);
      applyWorkspace(nextWorkspace, 'Document saved');
    } catch {
      statusMessage = 'Save failed';
    }
  }

  async function handleSaveDocumentAs() {
    try {
      const nextWorkspace = await saveDocumentAs({
        path: documentState.path,
        title: documentState.title,
        markdown,
      });
      applyWorkspace(nextWorkspace, 'Document saved as');
    } catch {
      statusMessage = 'Save as cancelled or failed';
    }
  }

  async function handleExportHTML() {
    try {
      const htmlForExport = await buildRenderedHtmlForExport(markdown, true);
      await exportHTML({
        title: documentState.title,
        html: htmlForExport,
      });
      statusMessage = 'HTML exported';
    } catch {
      statusMessage = 'Export cancelled or failed';
    }
  }

  async function handleExportPDF() {
    try {
      flushDualPreviewUpdate();
      const htmlForExport = buildPdfHtmlFromPreview();
      await exportPDF({
        title: documentState.title,
        markdown,
        html: htmlForExport,
      });
      statusMessage = 'PDF exported';
    } catch {
      statusMessage = 'PDF export cancelled or failed';
    }
  }

  async function handleSelectNote(notePath: string) {
    if (notePath === documentState.path) {
      return;
    }

    try {
      const nextWorkspace = await openDocumentAtPath(notePath);
      applyWorkspace(nextWorkspace, 'Document selected');
    } catch {
      statusMessage = 'Unable to load selected note';
    }
  }

  function handleSourceDraftInput(event: Event) {
    const target = event.currentTarget as HTMLTextAreaElement;
    const value = target.value;
    sourceDraft = value;
    if (mode === 'Document') {
      markdown = value;
      documentState = {...documentState, title: titleFromMarkdown(value)};
      isDirty = value !== documentState.markdown;
    } else if (mode === 'Dual') {
      documentState = {...documentState, title: titleFromMarkdown(value)};
      isDirty = value !== documentState.markdown;
      scheduleDualPreviewUpdate(value);
    }
    lastSourceSelectionStart = target.selectionStart ?? value.length;
    lastSourceSelectionEnd = target.selectionEnd ?? value.length;
    statusMessage = 'Source draft changed';
    recordSourceHistory(value);
  }

  function scheduleDualPreviewUpdate(value: string) {
    if (dualPreviewTimer) {
      window.clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
    }

    dualPreviewTimer = window.setTimeout(() => {
      dualPreviewTimer = null;
      if (mode === 'Dual') {
        markdown = value;
      }
    }, 140);
  }

  function flushDualPreviewUpdate() {
    if (dualPreviewTimer) {
      window.clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
      if (mode === 'Dual') {
        markdown = sourceDraft;
      }
    }
  }

  function rememberSourceSelection(event: Event) {
    const target = event.currentTarget as HTMLTextAreaElement;
    lastSourceSelectionStart = target.selectionStart ?? 0;
    lastSourceSelectionEnd = target.selectionEnd ?? 0;
  }

  function resolveSourceSelection() {
    if (sourceTextarea && document.activeElement === sourceTextarea) {
      return {
        start: sourceTextarea.selectionStart ?? 0,
        end: sourceTextarea.selectionEnd ?? 0,
      };
    }

    return {
      start: lastSourceSelectionStart,
      end: lastSourceSelectionEnd,
    };
  }

  function resetSourceHistory(seed: string) {
    sourceHistory = [seed];
    sourceHistoryIndex = 0;
  }

  function recordSourceHistory(next: string) {
    if (applyingSourceHistory) {
      return;
    }
    if (sourceHistoryIndex < 0 || sourceHistory.length === 0) {
      sourceHistory = [next];
      sourceHistoryIndex = 0;
      return;
    }
    if (sourceHistory[sourceHistoryIndex] === next) {
      return;
    }
    const snapshots = sourceHistory.slice(0, sourceHistoryIndex + 1);
    snapshots.push(next);
    if (snapshots.length > 300) {
      snapshots.shift();
    }
    sourceHistory = snapshots;
    sourceHistoryIndex = sourceHistory.length - 1;
  }

  function applySourceHistorySnapshot(next: string) {
    applyingSourceHistory = true;
    sourceDraft = next;
    documentState = {...documentState, title: titleFromMarkdown(next)};
    isDirty = next !== documentState.markdown;
    statusMessage = 'Source draft changed';
    if (mode === 'Dual') {
      scheduleDualPreviewUpdate(next);
    }
    applyingSourceHistory = false;
  }

  function performSourceUndo(): boolean {
    if (sourceHistoryIndex <= 0) {
      return false;
    }
    sourceHistoryIndex -= 1;
    applySourceHistorySnapshot(sourceHistory[sourceHistoryIndex]);
    return true;
  }

  function performSourceRedo(): boolean {
    if (sourceHistoryIndex < 0 || sourceHistoryIndex >= sourceHistory.length - 1) {
      return false;
    }
    sourceHistoryIndex += 1;
    applySourceHistorySnapshot(sourceHistory[sourceHistoryIndex]);
    return true;
  }

  function handleSourceKeyDown(event: KeyboardEvent) {
    const hasPrimaryModifier = event.ctrlKey || event.metaKey;
    if (!hasPrimaryModifier) {
      return;
    }

    const key = event.key.toLowerCase();
    const wantsUndo = key === 'z' && !event.shiftKey;
    const wantsRedo = (key === 'z' && event.shiftKey) || key === 'y';
    if (!wantsUndo && !wantsRedo) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    wantsUndo ? performSourceUndo() : performSourceRedo();
  }

  function handleToolbarMouseDown(event: MouseEvent) {
    const target = event.target as Element | null;
    if (mode === 'Document' && (target?.closest('.toolbar-menu__item') || target?.closest('.toolbar-menu__trigger'))) {
      event.preventDefault();
      return;
    }
    if (target?.closest('.toolbar-menu__item')) {
      event.preventDefault();
    }
  }

  function handleApplySource() {
    flushDualPreviewUpdate();
    markdown = sourceDraft;
    documentState = {...documentState, title: titleFromMarkdown(sourceDraft)};
    isDirty = sourceDraft !== documentState.markdown;
    statusMessage = 'Source applied to document';
    recordMarkdownHistory(sourceDraft);
    documentHistory = [];
    documentHistoryIndex = -1;
  }

  function handleRevertSource() {
    flushDualPreviewUpdate();
    sourceDraft = markdown;
    statusMessage = 'Source draft reverted';
    recordSourceHistory(sourceDraft);
    documentHistory = [];
    documentHistoryIndex = -1;
  }

  async function handleModeChange(nextMode: EditorMode | string) {
    flushDualPreviewUpdate();
    const normalizedMode = normalizeMode(nextMode);
    if (normalizedMode === 'Document' && hasSourceDraftChanges) {
      markdown = sourceDraft;
      documentState = {...documentState, title: titleFromMarkdown(sourceDraft)};
      isDirty = sourceDraft !== documentState.markdown;
      recordMarkdownHistory(sourceDraft);
    }
    mode = normalizedMode;
    if (normalizedMode !== 'Document') {
      documentHistory = [];
      documentHistoryIndex = -1;
    }
    openMenu = null;
    try {
      const nextConfig = await updatePreferences({
        preferredMode: normalizedMode,
        autosave: workspace.config.autosave,
        theme: workspace.config.theme,
      });
      workspace = {...workspace, config: nextConfig};
    } catch {
      statusMessage = 'Unable to save mode preference';
    }
  }

  async function handleToggleAutosave() {
    const nextAutosave = !workspace.config.autosave;
    try {
      const nextConfig = await updatePreferences({
        preferredMode: mode,
        autosave: nextAutosave,
        theme: workspace.config.theme,
      });
      workspace = {...workspace, config: nextConfig};
      statusMessage = nextAutosave ? 'Autosave enabled' : 'Autosave disabled';
    } catch {
      statusMessage = 'Unable to update autosave';
    }
  }

  async function handleThemeChange(nextTheme: AppTheme) {
    if (nextTheme === workspace.config.theme) {
      return;
    }

    try {
      const nextConfig = await updatePreferences({
        preferredMode: mode,
        autosave: workspace.config.autosave,
        theme: nextTheme,
      });
      workspace = {...workspace, config: nextConfig};
      statusMessage = `Theme: ${nextTheme}`;
    } catch {
      statusMessage = 'Unable to change theme';
    }
  }

  async function handleDeleteActiveNote() {
    await handleDeleteNoteByPath(documentState.path);
  }

  async function handleDeleteNoteByPath(path: string) {
    const targetPath = path?.trim();
    if (!targetPath) {
      statusMessage = 'No saved note to delete';
      return;
    }

    const hasUnsavedActiveChanges = targetPath === documentState.path && (isDirty || hasSourceDraftChanges);
    const message = hasUnsavedActiveChanges
      ? `Delete note with unsaved changes?\n\n${targetPath}\n\nUnsaved changes will be lost.`
      : `Delete note?\n\n${targetPath}`;
    const confirmed = window.confirm(message);
    if (!confirmed) {
      return;
    }

    try {
      const nextWorkspace = await deleteNote(targetPath);
      applyWorkspace(nextWorkspace, 'Note deleted');
    } catch {
      statusMessage = 'Unable to delete note';
    }
  }

  async function handleExportNoteHTML(path: string) {
    const targetPath = path.trim();
    if (!targetPath) {
      return;
    }

    try {
      const doc =
        targetPath === documentState.path
          ? {title: documentState.title, markdown}
          : await getDocument(targetPath);
      const htmlForExport = await buildRenderedHtmlForExport(doc.markdown, targetPath === documentState.path);
      await exportHTML({
        title: doc.title,
        html: htmlForExport,
      });
      statusMessage = 'HTML exported';
    } catch {
      statusMessage = 'Export cancelled or failed';
    }
  }

  async function handleExportNotePDF(path: string) {
    const targetPath = path.trim();
    if (!targetPath) {
      return;
    }

    try {
      const doc =
        targetPath === documentState.path
          ? {title: documentState.title, markdown}
          : await getDocument(targetPath);
      const htmlForExport =
        targetPath === documentState.path && showDocumentPanel && documentPreviewEl
          ? buildPdfHtmlFromPreview()
          : markdownToHtml(doc.markdown);
      await exportPDF({
        title: doc.title,
        markdown: doc.markdown,
        html: htmlForExport,
      });
      statusMessage = 'PDF exported';
    } catch {
      statusMessage = 'PDF export cancelled or failed';
    }
  }

  async function handleRevealNoteInFS(path: string) {
    const targetPath = path.trim();
    if (!targetPath) {
      return;
    }

    try {
      await revealNoteInFS(targetPath);
      statusMessage = 'Opened in file manager';
    } catch {
      statusMessage = 'Unable to reveal file';
    }
  }

  function openNoteContextMenu(note: NoteSummary, x: number, y: number) {
    noteContextMenu = {note, x, y};
  }

  function handleNoteContextMenu(event: MouseEvent, note: NoteSummary) {
    event.preventDefault();
    event.stopPropagation();
    openNoteContextMenu(note, event.clientX, event.clientY);
  }

  function handleMenuToggle(name: string, open: boolean) {
    openMenu = open ? name : null;
  }

  function handleDetailsToggle(name: string, event: Event) {
    const details = event.currentTarget as HTMLDetailsElement;
    handleMenuToggle(name, details.open);
  }

  function buildPdfHtmlFromPreview(): string {
    if (!showDocumentPanel || !documentPreviewEl) {
      return markdownToHtml(markdown);
    }

    const clone = documentPreviewEl.cloneNode(true) as HTMLElement;
    clone.removeAttribute('contenteditable');

    clone.querySelectorAll('[data-marktyp-caret]').forEach((node) => node.remove());
    clone.querySelectorAll('.katex-mathml').forEach((node) => node.remove());
    clone.querySelectorAll('marktyp-math-inline[contenteditable], marktyp-math-block[contenteditable], marktyp-mermaid[contenteditable]').forEach((node) => {
      (node as HTMLElement).removeAttribute('contenteditable');
    });

    return clone.innerHTML;
  }

  async function buildRenderedHtmlForExport(markdownInput: string, preferPreview: boolean): Promise<string> {
    if (preferPreview && showDocumentPanel && documentPreviewEl) {
      return buildPdfHtmlFromPreview();
    }

    const temp = document.createElement('div');
    temp.innerHTML = markdownToHtml(markdownInput);
    renderMathNodes(temp);
    renderCodeNodes(temp);
    mermaidRenderGeneration += 1;
    const generation = mermaidRenderGeneration;
    await renderMermaidNodes(temp, generation);
    return temp.innerHTML;
  }

  function applyNextMarkdown(next: string, nextStatus = 'Source draft changed') {
    sourceDraft = next;
    markdown = next;
    documentState = {...documentState, title: titleFromMarkdown(next)};
    isDirty = next !== documentState.markdown;
    statusMessage = nextStatus;
    recordSourceHistory(next);
    recordMarkdownHistory(next);
  }

  function syncDocumentEditorToMarkdown(nextStatus = 'Document changed') {
    if (!documentPreviewEl) {
      return;
    }

    const next = htmlToMarkdown(documentPreviewEl.innerHTML);
    sourceDraft = next;
    markdown = next;
    documentState = {...documentState, title: titleFromMarkdown(next)};
    isDirty = next !== documentState.markdown;
    statusMessage = nextStatus;
    recordMarkdownHistory(next);
  }

  function focusDocumentEditor(): HTMLElement | null {
    if (mode !== 'Document' || !documentPreviewEl) {
      return null;
    }

    if (document.activeElement !== documentPreviewEl) {
      documentPreviewEl.focus();
    }

    return documentPreviewEl;
  }

  function resolveDocumentRange(root: HTMLElement): Range | null {
    const selection = window.getSelection();
    if (selection && selection.rangeCount > 0) {
      const range = selection.getRangeAt(0);
      if (root.contains(range.commonAncestorContainer)) {
        return range.cloneRange();
      }
    }

    if (
      lastDocumentRange &&
      root.contains(lastDocumentRange.startContainer) &&
      root.contains(lastDocumentRange.endContainer)
    ) {
      return lastDocumentRange.cloneRange();
    }

    const fallback = document.createRange();
    fallback.selectNodeContents(root);
    fallback.collapse(false);
    return fallback;
  }

  function applyDocumentRange(range: Range) {
    const selection = window.getSelection();
    if (!selection) {
      return;
    }
    selection.removeAllRanges();
    selection.addRange(range);
    lastDocumentRange = range.cloneRange();
  }

  function runDocumentCommand(command: string, value?: string) {
    const root = focusDocumentEditor();
    if (!root) {
      return false;
    }
    document.execCommand(command, false, value);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
    openMenu = null;
    return true;
  }

  function insertDocumentHtml(html: string) {
    const root = focusDocumentEditor();
    if (!root) {
      return false;
    }

    const range = resolveDocumentRange(root);
    if (!range) {
      statusMessage = 'Unable to resolve document cursor';
      return false;
    }

    const markerId = `marktyp-caret-${Math.random().toString(36).slice(2)}`;
    const fragment = range.createContextualFragment(`${html}<span data-marktyp-caret="${markerId}">\u200b</span>`);
    range.deleteContents();
    range.insertNode(fragment);

    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    renderPreviewDecorations(documentPreviewEl);
    recordDocumentHistory();

    const marker = root.querySelector(`[data-marktyp-caret="${markerId}"]`) as HTMLElement | null;
    if (marker) {
      const nextRange = document.createRange();
      nextRange.setStartBefore(marker);
      nextRange.setEndBefore(marker);
      applyDocumentRange(nextRange);
      marker.remove();
    }

    openMenu = null;
    return true;
  }

  function handleDocumentInput() {
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown('Document changed');
    recordDocumentHistory();
  }

  function handleDocumentFocus() {
    isEditingDocument = true;
    const selection = window.getSelection();
    if (selection && selection.rangeCount > 0) {
      lastDocumentRange = selection.getRangeAt(0).cloneRange();
    }
    if (documentHistory.length === 0 && documentPreviewEl) {
      resetDocumentHistory();
    }
  }

  function handleDocumentBlur() {
    if (suppressNextDocumentBlur) {
      suppressNextDocumentBlur = false;
      return;
    }
    if (documentEditorChanged) {
      syncDocumentEditorToMarkdown('Document synced');
    }
    documentEditorChanged = false;
    isEditingDocument = false;
  }

  function handleDocumentMouseDown(event: MouseEvent) {
    if (event.button === 2) {
      suppressNextDocumentBlur = true;
    }
  }

  function handleDocumentSelectionEvent() {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || !documentPreviewEl) {
      return;
    }
    const range = selection.getRangeAt(0);
    if (!documentPreviewEl.contains(range.commonAncestorContainer)) {
      return;
    }
    lastDocumentRange = range.cloneRange();
  }

  function openRichEditor(node: HTMLElement, kind: RichEditorKind) {
    richEditorTarget = node;
    richEditorKind = kind;
    richEditorSource = node.getAttribute('source') || '';
    richEditorOpen = true;
  }

  function handleDocumentClick(event: MouseEvent) {
    const target = event.target as HTMLElement | null;
    if (!target) {
      return;
    }

    const inlineMath = target.closest('marktyp-math-inline') as HTMLElement | null;
    if (inlineMath) {
      event.preventDefault();
      openRichEditor(inlineMath, 'math-inline');
      return;
    }

    const blockMath = target.closest('marktyp-math-block') as HTMLElement | null;
    if (blockMath) {
      event.preventDefault();
      openRichEditor(blockMath, 'math-block');
      return;
    }
  }

  function handleRichEditorApply() {
    if (!richEditorTarget) {
      richEditorOpen = false;
      return;
    }

    const source = richEditorSource.trim();
    richEditorTarget.setAttribute('source', source);
    renderPreviewDecorations(documentPreviewEl);
    syncDocumentEditorToMarkdown('Block updated');
    richEditorOpen = false;
    richEditorTarget = null;
  }

  function handleRichEditorCancel() {
    richEditorOpen = false;
    richEditorTarget = null;
  }

  function insertTextAtDocumentCursor(text: string) {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) {
      return false;
    }

    const range = selection.getRangeAt(0);
    const root = documentPreviewEl;
    if (!root || !root.contains(range.commonAncestorContainer)) {
      return false;
    }

    range.deleteContents();
    const textNode = document.createTextNode(text);
    range.insertNode(textNode);
    range.setStartAfter(textNode);
    range.setEndAfter(textNode);
    selection.removeAllRanges();
    selection.addRange(range);
    return true;
  }

  function renderMathNodes(container: HTMLElement | null) {
    if (!container) {
      return;
    }

    const inlineNodes = container.querySelectorAll('marktyp-math-inline');
    inlineNodes.forEach((node) => {
      (node as HTMLElement).setAttribute('contenteditable', 'false');
      const source = node.getAttribute('source') || '';
      const cached = mathInlineCache.get(source);
      if (cached) {
        node.innerHTML = cached;
        return;
      }
      try {
        const rendered = katex.renderToString(source, {
          displayMode: false,
          throwOnError: false,
        });
        mathInlineCache.set(source, rendered);
        node.innerHTML = rendered;
      } catch {
        node.textContent = `$${source}$`;
      }
    });

    const blockNodes = container.querySelectorAll('marktyp-math-block');
    blockNodes.forEach((node) => {
      (node as HTMLElement).setAttribute('contenteditable', 'false');
      const source = node.getAttribute('source') || '';
      const cached = mathBlockCache.get(source);
      if (cached) {
        node.innerHTML = cached;
        return;
      }
      try {
        const rendered = katex.renderToString(source, {
          displayMode: true,
          throwOnError: false,
        });
        mathBlockCache.set(source, rendered);
        node.innerHTML = rendered;
      } catch {
        node.textContent = `$$\n${source}\n$$`;
      }
    });
  }

  function renderCodeNodes(container: HTMLElement | null) {
    if (!container) {
      return;
    }

    const codeNodes = container.querySelectorAll('pre code');
    codeNodes.forEach((node) => {
      hljs.highlightElement(node as HTMLElement);
    });
  }

  async function renderMermaidNodes(container: HTMLElement | null, generation: number) {
    if (!container) {
      return;
    }

    if (!mermaidApi) {
      const mod = await import('mermaid');
      mermaidApi = mod.default as unknown as {initialize: Function; render: Function};
    }

    if (!mermaidInitialized) {
      mermaidApi.initialize({
        startOnLoad: false,
        securityLevel: 'loose',
      });
      mermaidInitialized = true;
    }

    const mermaidNodes = container.querySelectorAll('marktyp-mermaid');
    for (const node of mermaidNodes) {
      (node as HTMLElement).setAttribute('contenteditable', 'false');
      const source = node.getAttribute('source') || '';
      const renderedSource = node.getAttribute('data-rendered-source') || '';
      const cachedSvg = mermaidSvgCache.get(source);
      if (cachedSvg) {
        node.innerHTML = cachedSvg;
        node.setAttribute('data-rendered-source', source);
        continue;
      }
      if (renderedSource === source && node.innerHTML.trim() !== '') {
        continue;
      }

      try {
        if (generation !== mermaidRenderGeneration) {
          return;
        }
        const diagramId = `marktyp-mermaid-${Math.random().toString(36).slice(2)}`;
        const result = await mermaidApi.render(diagramId, source);
        if (generation !== mermaidRenderGeneration) {
          return;
        }
        mermaidSvgCache.set(source, result.svg);
        node.innerHTML = result.svg;
        node.setAttribute('data-rendered-source', source);
      } catch {
        if (generation !== mermaidRenderGeneration) {
          return;
        }
        node.textContent = `\`\`\`mermaid\n${source}\n\`\`\``;
      }
    }
  }

  function hydrateCachedMermaidNodes(container: HTMLElement | null): boolean {
    if (!container) {
      return false;
    }

    let hasPending = false;
    const mermaidNodes = container.querySelectorAll('marktyp-mermaid');
    for (const node of mermaidNodes) {
      const source = node.getAttribute('source') || '';
      const cachedSvg = mermaidSvgCache.get(source);
      if (cachedSvg) {
        node.innerHTML = cachedSvg;
        node.setAttribute('data-rendered-source', source);
      } else if (source.trim() !== '') {
        hasPending = true;
      }
    }

    return hasPending;
  }

  function scheduleMermaidRender(container: HTMLElement | null) {
    if (!container) {
      return;
    }

    const hasPending = hydrateCachedMermaidNodes(container);
    if (!hasPending) {
      return;
    }

    if (mermaidRenderTimer) {
      window.clearTimeout(mermaidRenderTimer);
      mermaidRenderTimer = null;
    }

    mermaidRenderGeneration += 1;
    const generation = mermaidRenderGeneration;
    mermaidRenderTimer = window.setTimeout(() => {
      mermaidRenderTimer = null;
      void renderMermaidNodes(container, generation);
    }, 80);
  }

  function renderPreviewDecorations(container: HTMLElement | null) {
    renderMathNodes(container);
    renderCodeNodes(container);
    scheduleMermaidRender(container);
  }

  async function schedulePreviewDecorations() {
    if (previewRenderScheduled || !documentPreviewEl) {
      return;
    }

    previewRenderScheduled = true;
    await tick();
    previewRenderScheduled = false;
    renderPreviewDecorations(documentPreviewEl);
  }

  async function wrapSelection(prefix: string, suffix: string) {
    if (mode === 'Document' && documentPreviewEl) {
      const active = document.activeElement === documentPreviewEl;
      if (!active) {
        documentPreviewEl.focus();
      }

      if (prefix === '**' && suffix === '**') {
        document.execCommand('bold');
      } else if (prefix === '*' && suffix === '*') {
        document.execCommand('italic');
      } else {
        const selection = window.getSelection();
        const selected = selection?.toString() || '';
        const inserted = insertTextAtDocumentCursor(`${prefix}${selected}${suffix}`);
        if (!inserted) {
          insertTextAtDocumentCursor(`${prefix}${suffix}`);
        }
      }

      syncDocumentEditorToMarkdown();
      openMenu = null;
      return;
    }

    if (!sourceTextarea) {
      statusMessage = 'Source editor unavailable';
      return;
    }

    const {start, end} = resolveSourceSelection();
    const selected = sourceDraft.slice(start, end);
    const next = `${sourceDraft.slice(0, start)}${prefix}${selected}${suffix}${sourceDraft.slice(end)}`;
    applyNextMarkdown(next);
    openMenu = null;

    queueMicrotask(() => {
      sourceTextarea?.focus();
      const cursor = end + prefix.length + suffix.length;
      sourceTextarea?.setSelectionRange(cursor, cursor);
      lastSourceSelectionStart = cursor;
      lastSourceSelectionEnd = cursor;
    });
  }

  async function insertSnippet(snippet: string) {
    if (mode === 'Document' && documentPreviewEl) {
      const active = document.activeElement === documentPreviewEl;
      if (!active) {
        documentPreviewEl.focus();
      }
      insertTextAtDocumentCursor(snippet);
      syncDocumentEditorToMarkdown();
      openMenu = null;
      return;
    }

    if (!sourceTextarea) {
      statusMessage = 'Source editor unavailable';
      return;
    }

    const selection = resolveSourceSelection();
    const start = selection.start ?? sourceDraft.length;
    const end = selection.end ?? sourceDraft.length;
    const next = `${sourceDraft.slice(0, start)}${snippet}${sourceDraft.slice(end)}`;
    applyNextMarkdown(next);
    openMenu = null;

    queueMicrotask(() => {
      sourceTextarea?.focus();
      const cursor = start + snippet.length;
      sourceTextarea?.setSelectionRange(cursor, cursor);
      lastSourceSelectionStart = cursor;
      lastSourceSelectionEnd = cursor;
    });
  }

  async function insertLink() {
    const url = window.prompt('Link URL', 'https://');
    if (!url || url.trim() === '') {
      return;
    }
    if (mode === 'Document' && documentPreviewEl) {
      runDocumentCommand('createLink', url.trim());
      return;
    }
    await insertSnippet(`[link](${url.trim()})`);
  }

  async function insertImage() {
    const src = window.prompt('Image URL or path', '');
    if (!src || src.trim() === '') {
      return;
    }
    if (mode === 'Document' && documentPreviewEl) {
      runDocumentCommand('insertImage', src.trim());
      return;
    }
    await insertSnippet(`![image](${src.trim()})`);
  }

  async function insertMermaid() {
    if (mode === 'Document') {
      insertDocumentHtml(
        `<marktyp-mermaid source="graph TD;&#10;  A[marktyp] --&gt; B[Mermaid]"></marktyp-mermaid><p><br></p>`,
      );
      return;
    }
    await insertSnippet('\n```mermaid\ngraph TD;\n  A[marktyp] --> B[Mermaid]\n```\n');
  }

  async function insertInlineMath() {
    if (mode === 'Document') {
      insertDocumentHtml(`<marktyp-math-inline source="x^2"></marktyp-math-inline>`);
      return;
    }
    await insertSnippet('$x^2$');
  }

  async function insertMathBlock() {
    if (mode === 'Document') {
      insertDocumentHtml(`<marktyp-math-block source="E = mc^2"></marktyp-math-block><p><br></p>`);
      return;
    }
    await insertSnippet('\n$$\nE = mc^2\n$$\n');
  }

  function insertHeading2() {
    if (mode === 'Document') {
      runDocumentCommand('formatBlock', 'h2');
      return;
    }
    void insertSnippet('\n## Heading 2\n');
  }

  function insertBulletList() {
    if (mode === 'Document') {
      runDocumentCommand('insertUnorderedList');
      return;
    }
    void insertSnippet('\n- List item\n');
  }

  function insertTaskList() {
    if (mode === 'Document') {
      insertDocumentHtml('<ul><li><input type="checkbox" /> Task item</li></ul><p><br></p>');
      return;
    }
    void insertSnippet('\n- [ ] Task item\n');
  }

  function insertBlockquote() {
    if (mode === 'Document') {
      runDocumentCommand('formatBlock', 'blockquote');
      return;
    }
    void insertSnippet('\n> Quote\n');
  }

  function insertCodeBlock() {
    if (mode === 'Document') {
      insertDocumentHtml('<pre><code data-language="txt" class="language-txt">code</code></pre><p><br></p>');
      return;
    }
    void insertSnippet('\n```txt\n\n```\n');
  }
</script>

<div class="app-shell">
  <div class="workspace-toolbar">
    <div class="toolbar-titlebar">
      <div class="toolbar-titlebar__meta">
        <p class="toolbar-titlebar__eyebrow">Current document</p>
        <h2 class="toolbar-titlebar__title">{documentState.title || activeNote?.title || 'Untitled'}</h2>
        <p class="toolbar-titlebar__path">{documentState.path || 'Unsaved markdown document'}</p>
      </div>
      <span class={statusLabel === 'Unsaved changes' || statusLabel === 'Source draft pending' ? 'status-pill is-dirty' : 'status-pill'}>
        {statusLabel}
      </span>
    </div>

    <div aria-label="Formatting toolbar" class="toolbar-cluster toolbar-cluster--editing" on:mousedown={handleToolbarMouseDown} role="toolbar" tabindex="-1">
      <details class="toolbar-menu" on:toggle={(event) => handleDetailsToggle('text', event)} open={openMenu === 'text'}>
        <summary class="toolbar-menu__trigger">
          Text
          <span class="toolbar-menu__chevron">▾</span>
        </summary>
        <div class="toolbar-menu__content">
          <button class="toolbar-menu__item" on:click={() => wrapSelection('**', '**')} type="button">Bold</button>
          <button class="toolbar-menu__item" on:click={() => wrapSelection('*', '*')} type="button">Italic</button>
          <button class="toolbar-menu__item" on:click={insertLink} type="button">Link</button>
        </div>
      </details>

      <details class="toolbar-menu" on:toggle={(event) => handleDetailsToggle('blocks', event)} open={openMenu === 'blocks'}>
        <summary class="toolbar-menu__trigger">
          Blocks
          <span class="toolbar-menu__chevron">▾</span>
        </summary>
        <div class="toolbar-menu__content">
          <button class="toolbar-menu__item" on:click={insertHeading2} type="button">Heading 2</button>
          <button class="toolbar-menu__item" on:click={insertBulletList} type="button">Bullet List</button>
          <button class="toolbar-menu__item" on:click={insertTaskList} type="button">Task List</button>
          <button class="toolbar-menu__item" on:click={insertBlockquote} type="button">Blockquote</button>
        </div>
      </details>

      <details class="toolbar-menu" on:toggle={(event) => handleDetailsToggle('code', event)} open={openMenu === 'code'}>
        <summary class="toolbar-menu__trigger">
          Code
          <span class="toolbar-menu__chevron">▾</span>
        </summary>
        <div class="toolbar-menu__content">
          <button class="toolbar-menu__item" on:click={() => wrapSelection('`', '`')} type="button">Inline Code</button>
          <button class="toolbar-menu__item" on:click={insertCodeBlock} type="button">Code Block</button>
        </div>
      </details>

      <details class="toolbar-menu" on:toggle={(event) => handleDetailsToggle('insert', event)} open={openMenu === 'insert'}>
        <summary class="toolbar-menu__trigger">
          Insert
          <span class="toolbar-menu__chevron">▾</span>
        </summary>
        <div class="toolbar-menu__content">
          <button class="toolbar-menu__item" on:click={insertImage} type="button">Image</button>
          <button class="toolbar-menu__item" on:click={insertMermaid} type="button">Mermaid</button>
          <button class="toolbar-menu__item" on:click={insertInlineMath} type="button">Inline Math</button>
          <button class="toolbar-menu__item" on:click={insertMathBlock} type="button">Math Block</button>
        </div>
      </details>
    </div>
  </div>

  <div
    bind:this={appMainEl}
    class="app-main"
    style={`grid-template-columns: ${sidebarColumnWidth}px 8px minmax(0, 1fr);`}
  >
    <aside class={sidebarCollapsed ? 'sidebar is-collapsed' : 'sidebar'}>
      {#if sidebarCollapsed}
        <div class="sidebar-collapsed">
          <button class="sidebar-collapsed__toggle" on:click={toggleSidebar} type="button" title="Show notes">»</button>
        </div>
      {:else}
        <div class="sidebar__collapse-row">
          <button class="sidebar-collapsed__toggle" on:click={toggleSidebar} type="button" title="Hide notes">«</button>
        </div>
        <div class="sidebar__brand">
          <div>
            <p class="sidebar__eyebrow">{workspace.appInfo.version}</p>
            <h1>{workspace.appInfo.name}</h1>
          </div>
          <p class="sidebar__tagline">{workspace.appInfo.tagline}</p>
        </div>

        <div class="sidebar__section">
          <div class="sidebar__section-header">
            <span>Recent notes</span>
            <div class="sidebar__section-actions">
              <button class="ghost-button" on:click={handleNewDocument} type="button">New</button>
              <button class="ghost-button ghost-button--danger" disabled={!documentState.path} on:click={() => handleDeleteActiveNote()} type="button">
                Delete
              </button>
            </div>
          </div>

          <input class="note-search" bind:value={noteQuery} placeholder="Search notes" type="text" />

          <div class="note-list">
            {#each filteredNotes as note (note.id + note.path)}
              <button
                class={note.path === documentState.path ? 'note-card is-active' : 'note-card'}
                on:click={() => handleSelectNote(note.path)}
                on:contextmenu={(event) => handleNoteContextMenu(event, note)}
                type="button"
              >
                <span class="note-card__title">{note.title}</span>
                <span class="note-card__meta">{note.updatedLabel}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </aside>

    <div class={sidebarCollapsed ? 'sidebar-resizer is-hidden' : 'sidebar-resizer'} on:pointerdown={startSidebarResize}></div>

    <main class="workspace">
      <section
        bind:this={dualBodyEl}
        class={mode === 'Dual' ? 'workspace__body is-dual' : 'workspace__body'}
        style={mode === 'Dual'
          ? `grid-template-columns: minmax(320px, ${dualSplit}%) 8px minmax(280px, ${100 - dualSplit}%);`
          : undefined}
      >
        {#if showDocumentPanel}
          <div class="panel">
            <div class="panel__header">
              <span>Document</span>
              <span class="panel__hint">{mode === 'Document' ? 'Visual draft' : 'Markdown preview'}</span>
            </div>
            <div class="panel__body">
              {#if mode === 'Document'}
                <article
                  bind:this={documentPreviewEl}
                  class="document-preview document-editor__content document-editor__content--editable"
                  contenteditable="true"
                  on:blur={handleDocumentBlur}
                  on:click={handleDocumentClick}
                  on:focus={handleDocumentFocus}
                  on:beforeinput={handleDocumentBeforeInput}
                  on:input={handleDocumentInput}
                  on:keydown|capture={handleDocumentKeyDown}
                  on:keyup={handleDocumentSelectionEvent}
                  on:mousedown={handleDocumentMouseDown}
                  on:mouseup={handleDocumentSelectionEvent}
                  spellcheck="false"
                >
                  {@html documentHtmlDraft}
                </article>
              {:else}
                <article bind:this={documentPreviewEl} class="document-preview document-editor__content">
                  {@html renderedDocumentHtml}
                </article>
              {/if}
            </div>
          </div>
        {/if}

        {#if mode === 'Dual'}
          <div class="dual-resizer" on:pointerdown={startDualResize}></div>
        {/if}

        {#if showSourcePanel}
          <div class="panel">
            <div class="panel__header">
              <span>Source</span>
              <div class="panel__actions">
                <span class="panel__hint">Markdown draft</span>
                <button class="ghost-button ghost-button--small" disabled={!hasSourceDraftChanges} on:click={handleRevertSource} type="button">
                  Revert
                </button>
                <button class="ghost-button ghost-button--small" disabled={!hasSourceDraftChanges} on:click={handleApplySource} type="button">
                  Apply
                </button>
              </div>
            </div>
            <div class="panel__body">
              <textarea
                bind:this={sourceTextarea}
                bind:value={sourceDraft}
                class="source-editor"
                on:click={rememberSourceSelection}
                on:focus={rememberSourceSelection}
                on:input={handleSourceDraftInput}
                on:keydown={handleSourceKeyDown}
                on:keyup={rememberSourceSelection}
                on:select={rememberSourceSelection}
                spellcheck={false}
              />
            </div>
          </div>
        {/if}
      </section>
    </main>
  </div>

  {#if richEditorOpen}
    <div class="rich-editor-overlay" on:click={handleRichEditorCancel} role="presentation">
      <div class="rich-editor-modal" on:click|stopPropagation role="dialog" aria-modal="true">
        <div class="rich-editor-modal__header">
          <span>{richEditorKind === 'math-inline' ? 'Inline Math' : richEditorKind === 'math-block' ? 'Math Block' : 'Mermaid'}</span>
        </div>
        <textarea bind:value={richEditorSource} class="rich-editor-modal__input" spellcheck={false}></textarea>
        <div class="rich-editor-modal__actions">
          <button class="ghost-button ghost-button--small" on:click={handleRichEditorCancel} type="button">Cancel</button>
          <button class="ghost-button ghost-button--small" on:click={handleRichEditorApply} type="button">Apply</button>
        </div>
      </div>
    </div>
  {/if}

  {#if noteContextMenu}
    <div
      class="note-context-menu"
      on:click|stopPropagation
      on:keydown|stopPropagation
      role="menu"
      tabindex="0"
      style={`left: ${noteContextMenu.x}px; top: ${noteContextMenu.y}px;`}
    >
      <button
        class="note-context-menu__item"
        on:click={() => {
          const target = noteContextMenu?.note.path;
          noteContextMenu = null;
          if (target) void handleRevealNoteInFS(target);
        }}
        type="button"
      >
        Reveal In Files
      </button>
      <button
        class="note-context-menu__item"
        on:click={() => {
          const target = noteContextMenu?.note.path;
          noteContextMenu = null;
          if (target) void handleExportNoteHTML(target);
        }}
        type="button"
      >
        Export HTML
      </button>
      <button
        class="note-context-menu__item"
        on:click={() => {
          const target = noteContextMenu?.note.path;
          noteContextMenu = null;
          if (target) void handleExportNotePDF(target);
        }}
        type="button"
      >
        Export PDF
      </button>
      <button
        class="note-context-menu__item is-danger"
        on:click={() => {
          const target = noteContextMenu?.note.path;
          noteContextMenu = null;
          if (target) void handleDeleteNoteByPath(target);
        }}
        type="button"
      >
        Delete
      </button>
    </div>
  {/if}
</div>
