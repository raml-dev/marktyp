<script lang="ts">
  import FormattingToolbar from './FormattingToolbar.svelte';
  import NotesSidebar from './NotesSidebar.svelte';
  import NoteContextMenu from './NoteContextMenu.svelte';
  import DocumentEditor from './DocumentEditor.svelte';
  import ResourceDialog from './ResourceDialog.svelte';
  import TableDialog from './TableDialog.svelte';
  import AboutDialog from './AboutDialog.svelte';
  import DeleteNoteDialog from './DeleteNoteDialog.svelte';
  let documentEditor: DocumentEditor | null = null;
  import { editorState, workspaceStore, normalizeMode } from '../stores/workspaceStore';

  import { onMount } from 'svelte';

  import type { AppTheme, EditorMode, NoteSummary } from '../../types';
  import SourceEditor from './SourceEditor.svelte';
  let sourceEditor: SourceEditor | null = null;
  import { isSafeUrl } from '../utils/sanitizeHtml';
  import { createHTMLTable, createMarkdownTable, markdownDestination } from '../utils/insertions';

  type NoteContextMenuState = {
    note: NoteSummary;
    x: number;
    y: number;
  };

  let noteContextMenu: NoteContextMenuState | null = null;

  let openMenu: string | null = null;

  let appMainEl: HTMLElement | null = null;
  let dualBodyEl: HTMLElement | null = null;
  let sidebarCollapsed = false;
  let sidebarWidth = 320;
  let dualSplit = 58;
  let resizingSidebar = false;
  let resizingDual = false;
  let resourceDialog: 'link' | null = null;
  let resourceValue = '';
  let tableDialogOpen = false;
  let tableRows = 3;
  let tableColumns = 3;
  let aboutOpen = false;
  let pendingDeletePath: string | null = null;

  $: hasSourceDraftChanges = $editorState.sourceDraft !== $editorState.markdown;
  $: activeNote =
    $editorState.workspace.notes.find((note) => note.path === $editorState.documentState.path) ?? null;
  $: filteredNotes = (() => {
    const query = $editorState.noteQuery.trim().toLowerCase();
    if (!query) {
      return $editorState.workspace.notes;
    }

    return $editorState.workspace.notes.filter((note) => {
      return note.title.toLowerCase().includes(query) || note.path.toLowerCase().includes(query);
    });
  })();
  $: statusLabel = hasSourceDraftChanges
    ? 'Source draft pending'
    : $editorState.isDirty
      ? 'Unsaved changes'
      : $editorState.statusMessage;
  $: showDocumentPanel = $editorState.mode === 'Document' || $editorState.mode === 'Dual';
  $: showSourcePanel = $editorState.mode === 'Source' || $editorState.mode === 'Dual';

  $: {
    document.documentElement.dataset.theme = $editorState.workspace.config.theme || 'marktyp';
  }

  $: workspaceStore.scheduleDraft(
    $editorState.documentState.path,
    $editorState.sourceDraft,
    $editorState.sourceDraft !== $editorState.documentState.markdown,
  );
  $: workspaceStore.scheduleAutosave(
    $editorState.workspace.config.autosave,
    $editorState.sourceDraft !== $editorState.documentState.markdown,
    $editorState.sourceDraft,
    $editorState.saving,
    $editorState.loading,
  );

  onMount(() => {
    void workspaceStore.load().then(resetEditorSession);

    const unsubscribe = workspaceStore.listen((action: string) => {
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
        case 'about:open':
          aboutOpen = true;
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

      if (resizingDual && dualBodyEl && $editorState.mode === 'Dual') {
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
      if (document.querySelector('dialog[open]')) return;
      if (event.key === 'Escape') {
        noteContextMenu = null;
        openMenu = null;
      }

      const hasPrimaryModifier = event.ctrlKey || event.metaKey;
      const key = event.key.toLowerCase();
      if (!hasPrimaryModifier) return;
      if (key === 's') {
        event.preventDefault();
        void (event.shiftKey ? handleSaveDocumentAs() : handleSaveFromShortcut());
      } else if (key === 'n') {
        event.preventDefault();
        void handleNewDocument();
      } else if (key === 'o') {
        event.preventDefault();
        void handleOpenDocument();
      }
    }

    window.addEventListener('pointerdown', onPointerDown);
    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', onPointerUp);
    window.addEventListener('resize', onResizeOrScroll);
    window.addEventListener('scroll', onResizeOrScroll, true);
    window.addEventListener('keydown', onKeyDown, true);

    return () => {
      unsubscribe();
      workspaceStore.dispose();
      document.body.classList.remove('is-resizing');
      window.removeEventListener('pointerdown', onPointerDown);
      window.removeEventListener('pointermove', onPointerMove);
      window.removeEventListener('pointerup', onPointerUp);
      window.removeEventListener('pointercancel', onPointerUp);
      window.removeEventListener('resize', onResizeOrScroll);
      window.removeEventListener('scroll', onResizeOrScroll, true);
      window.removeEventListener('keydown', onKeyDown, true);
    };
  });

  function clamp(value: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, value));
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
    if ($editorState.mode !== 'Dual') return;
    event.preventDefault();
    resizingDual = true;
    document.body.classList.add('is-resizing');
  }

  function resizeSidebarWithKeyboard(event: KeyboardEvent) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    sidebarWidth = clamp(sidebarWidth + (event.key === 'ArrowRight' ? 16 : -16), 220, 520);
  }

  function resizeDualWithKeyboard(event: KeyboardEvent) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    dualSplit = clamp(dualSplit + (event.key === 'ArrowRight' ? 2 : -2), 32, 78);
  }

  function handleExportHTML() {
    return workspaceStore.exportDocument('html');
  }

  function handleExportPDF() {
    return workspaceStore.exportDocument('pdf');
  }

  function handleToolbarMouseDown(event: MouseEvent) {
    const target = event.target as Element | null;
    if (
      $editorState.mode === 'Document' &&
      (target?.closest('.toolbar-menu__item') || target?.closest('.toolbar-menu__trigger'))
    ) {
      event.preventDefault();
      return;
    }
    if (target?.closest('.toolbar-menu__item')) {
      event.preventDefault();
    }
  }

  async function handleModeChange(nextMode: EditorMode | string) {
    flushDualPreviewUpdate();
    const normalizedMode = normalizeMode(nextMode);
    openMenu = null;
    await workspaceStore.changeMode(normalizedMode);
  }

  function handleDeleteActiveNote() {
    requestDeleteNote($editorState.documentState.path);
  }

  function handleExportNoteHTML(path: string) {
    return workspaceStore.exportDocument('html', path);
  }

  function handleExportNotePDF(path: string) {
    return workspaceStore.exportDocument('pdf', path);
  }

  function handleRevealNoteInFS(path: string) {
    return workspaceStore.reveal(path);
  }

  function openNoteContextMenu(note: NoteSummary, x: number, y: number) {
    noteContextMenu = {
      note,
      x: clamp(x, 8, window.innerWidth - 200),
      y: clamp(y, 8, window.innerHeight - 180),
    };
  }

  function handleNoteContextMenu(event: MouseEvent, note: NoteSummary) {
    event.preventDefault();
    event.stopPropagation();
    openNoteContextMenu(note, event.clientX, event.clientY);
  }

  function openResourceDialog(kind: 'link') {
    if ($editorState.mode === 'Document') documentEditor?.preserveInsertionPoint();
    resourceDialog = kind;
    resourceValue = 'https://';
    openMenu = null;
  }

  function closeResourceDialog() {
    resourceDialog = null;
    resourceValue = '';
  }

  async function applyResourceDialog() {
    const kind = resourceDialog;
    const value = resourceValue.trim();
    if (!kind || !value || !isSafeUrl(value)) {
      workspaceStore.message('Enter a valid URL or file path');
      return;
    }
    closeResourceDialog();
    if ($editorState.mode === 'Document' && documentEditor) {
      documentEditor.insertLink(value);
      return;
    }
    await insertSnippet(`[link](${markdownDestination(value)})`);
  }

  function insertLink() {
    openResourceDialog('link');
  }

  async function insertImage() {
    openMenu = null;
    if ($editorState.mode === 'Document') documentEditor?.preserveInsertionPoint();
    const source = await workspaceStore.selectImage();
    if (!source) return;
    if ($editorState.mode === 'Document') documentEditor?.insertImage(source);
    else await insertSnippet(`![image](${markdownDestination(source)})`);
  }

  function openTableDialog() {
    if ($editorState.mode === 'Document') documentEditor?.preserveInsertionPoint();
    tableRows = 3;
    tableColumns = 3;
    tableDialogOpen = true;
    openMenu = null;
  }

  function insertTable() {
    if ($editorState.mode === 'Document') insertDocumentHtml(createHTMLTable(tableRows, tableColumns));
    else void insertSnippet(createMarkdownTable(tableRows, tableColumns));
    tableDialogOpen = false;
  }

  async function insertMermaid() {
    if ($editorState.mode === 'Document') {
      insertDocumentHtml(
        `<marktyp-mermaid source="graph TD;&#10;  A[marktyp] --&gt; B[Mermaid]"></marktyp-mermaid><p><br></p>`,
      );
      return;
    }
    await insertSnippet('\n```mermaid\ngraph TD;\n  A[marktyp] --> B[Mermaid]\n```\n');
  }

  async function insertInlineMath() {
    if ($editorState.mode === 'Document') {
      insertDocumentHtml(`<marktyp-math-inline source="x^2"></marktyp-math-inline>`);
      return;
    }
    await insertSnippet('$x^2$');
  }

  async function insertMathBlock() {
    if ($editorState.mode === 'Document') {
      insertDocumentHtml(`<marktyp-math-block source="E = mc^2"></marktyp-math-block><p><br></p>`);
      return;
    }
    await insertSnippet('\n$$\nE = mc^2\n$$\n');
  }

  function insertHeading2() {
    if ($editorState.mode === 'Document') {
      runDocumentCommand('formatBlock', 'h2');
      return;
    }
    void insertSnippet('\n## Heading 2\n');
  }

  function insertBulletList() {
    if ($editorState.mode === 'Document') {
      runDocumentCommand('insertUnorderedList');
      return;
    }
    void insertSnippet('\n- List item\n');
  }

  function insertTaskList() {
    if ($editorState.mode === 'Document') {
      insertDocumentHtml('<ul><li><input type="checkbox" /> Task item</li></ul><p><br></p>');
      return;
    }
    void insertSnippet('\n- [ ] Task item\n');
  }

  function insertBlockquote() {
    if ($editorState.mode === 'Document') {
      runDocumentCommand('formatBlock', 'blockquote');
      return;
    }
    void insertSnippet('\n> Quote\n');
  }

  function insertInlineCode() {
    if ($editorState.mode === 'Document') {
      documentEditor?.insertInlineCode();
      openMenu = null;
      return;
    }
    void wrapSelection('`', '`');
  }

  function insertCodeBlock() {
    if ($editorState.mode === 'Document') {
      insertDocumentHtml(
        '<pre><code data-marktyp-edit-target data-language="txt" class="language-txt">code</code></pre><p><br></p>',
      );
      return;
    }
    void insertSnippet('\n```txt\n\n```\n');
  }

  function resetEditorSession() {
    documentEditor?.resetSession();
    sourceEditor?.resetSession();
    resourceDialog = null;
    tableDialogOpen = false;
    noteContextMenu = null;
    pendingDeletePath = null;
  }
  async function handleNewDocument() {
    if (await workspaceStore.create()) resetEditorSession();
  }
  async function handleOpenDocument() {
    if (await workspaceStore.open()) resetEditorSession();
  }
  async function handleSelectNote(path: string) {
    if (path !== $editorState.documentState.path && (await workspaceStore.select(path))) resetEditorSession();
  }
  function requestDeleteNote(path: string) {
    if (!path) return;
    noteContextMenu = null;
    pendingDeletePath = path;
  }
  async function confirmDeleteNote() {
    const path = pendingDeletePath;
    pendingDeletePath = null;
    if (path && (await workspaceStore.remove(path))) resetEditorSession();
  }
  function handleSaveDocument() {
    flushDualPreviewUpdate();
    return workspaceStore.save();
  }
  const handleSaveFromShortcut = handleSaveDocument;
  function handleSaveDocumentAs() {
    flushDualPreviewUpdate();
    return workspaceStore.save(true);
  }
  function handleToggleAutosave() {
    return workspaceStore.preferences({ autosave: !$editorState.workspace.config.autosave });
  }
  function handleThemeChange(theme: AppTheme) {
    return workspaceStore.theme(theme);
  }

  function runDocumentCommand(command: string, value?: string) {
    openMenu = null;
    return documentEditor?.runDocumentCommand(command, value);
  }
  function insertDocumentHtml(html: string) {
    openMenu = null;
    return documentEditor?.insertDocumentHtml(html);
  }

  function flushDualPreviewUpdate() {
    sourceEditor?.flushDualPreviewUpdate();
  }
  function wrapSelection(prefix: string, suffix: string) {
    openMenu = null;
    if ($editorState.mode === 'Document') documentEditor?.wrapSelection(prefix, suffix);
    else void sourceEditor?.wrapSelection(prefix, suffix);
  }
  function insertSnippet(text: string) {
    openMenu = null;
    if ($editorState.mode === 'Document') documentEditor?.insertText(text);
    else return sourceEditor?.insertSnippet(text);
  }
</script>

<div
  class="app-shell"
  inert={$editorState.loading}
  aria-busy={$editorState.loading || $editorState.saving || $editorState.exporting}
>
  <div class="workspace-toolbar">
    <div class="toolbar-titlebar">
      <div class="toolbar-titlebar__meta">
        <p class="toolbar-titlebar__eyebrow">Current document</p>
        <h2 class="toolbar-titlebar__title">
          {$editorState.documentState.title || activeNote?.title || 'Untitled'}
        </h2>
        <p class="toolbar-titlebar__path">{$editorState.documentState.path || 'Unsaved markdown document'}</p>
      </div>
      <span
        role="status"
        aria-live="polite"
        class={statusLabel === 'Unsaved changes' || statusLabel === 'Source draft pending'
          ? 'status-pill is-dirty'
          : 'status-pill'}
      >
        {$editorState.loading
          ? 'Loading…'
          : $editorState.saving
            ? 'Saving…'
            : $editorState.exporting
              ? 'Exporting…'
              : statusLabel}
      </span>
    </div>

    <FormattingToolbar
      bind:openMenu
      onmousedown={handleToolbarMouseDown}
      actions={{
        Bold: () => wrapSelection('**', '**'),
        Italic: () => wrapSelection('*', '*'),
        Link: insertLink,
        'Heading 2': insertHeading2,
        'Bullet List': insertBulletList,
        'Task List': insertTaskList,
        Blockquote: insertBlockquote,
        'Inline Code': insertInlineCode,
        'Code Block': insertCodeBlock,
        Image: insertImage,
        Table: openTableDialog,
        Mermaid: insertMermaid,
        'Inline Math': insertInlineMath,
        'Math Block': insertMathBlock,
      }}
    />
  </div>

  <div
    bind:this={appMainEl}
    class="app-main"
    style={`grid-template-columns: ${sidebarColumnWidth}px 8px minmax(0, 1fr);`}
  >
    <NotesSidebar
      collapsed={sidebarCollapsed}
      bind:query={$editorState.noteQuery}
      notes={filteredNotes}
      activePath={$editorState.documentState.path}
      ontoggle={toggleSidebar}
      oncreate={handleNewDocument}
      ondelete={handleDeleteActiveNote}
      onselect={handleSelectNote}
      oncontextmenu={handleNoteContextMenu}
    />

    <button
      type="button"
      class={sidebarCollapsed ? 'sidebar-resizer is-hidden' : 'sidebar-resizer'}
      aria-label="Resize notes sidebar with the left and right arrow keys"
      data-value={Math.round(sidebarWidth)}
      tabindex="0"
      on:keydown={resizeSidebarWithKeyboard}
      on:pointerdown={startSidebarResize}
    ></button>

    <main class="workspace">
      <section
        bind:this={dualBodyEl}
        class={$editorState.mode === 'Dual' ? 'workspace__body is-dual' : 'workspace__body'}
        style={$editorState.mode === 'Dual'
          ? `grid-template-columns: minmax(0, ${dualSplit}fr) 8px minmax(0, ${100 - dualSplit}fr);`
          : undefined}
      >
        {#if showDocumentPanel}
          <div class="panel">
            <div class="panel__header">
              <span>Document</span>
              <span class="panel__hint"
                >{$editorState.mode === 'Document' ? 'Visual draft' : 'Markdown preview'}</span
              >
            </div>
            <div class="panel__body">
              <DocumentEditor bind:this={documentEditor} />
            </div>
          </div>
        {/if}

        {#if $editorState.mode === 'Dual'}
          <button
            type="button"
            class="dual-resizer"
            aria-label="Resize editor panes with the left and right arrow keys"
            data-value={Math.round(dualSplit)}
            tabindex="0"
            on:keydown={resizeDualWithKeyboard}
            on:pointerdown={startDualResize}
          ></button>
        {/if}

        {#if showSourcePanel}
          <SourceEditor bind:this={sourceEditor} />
        {/if}
      </section>
    </main>
  </div>

  {#if aboutOpen}
    <AboutDialog info={$editorState.workspace.appInfo} onclose={() => (aboutOpen = false)} />
  {/if}

  {#if pendingDeletePath}
    <DeleteNoteDialog
      path={pendingDeletePath}
      losesChanges={pendingDeletePath === $editorState.documentState.path && $editorState.isDirty}
      onconfirm={confirmDeleteNote}
      oncancel={() => (pendingDeletePath = null)}
    />
  {/if}

  {#if resourceDialog}
    <ResourceDialog
      kind={resourceDialog}
      bind:value={resourceValue}
      onapply={applyResourceDialog}
      oncancel={closeResourceDialog}
    />
  {/if}

  {#if tableDialogOpen}
    <TableDialog
      bind:rows={tableRows}
      bind:columns={tableColumns}
      onapply={insertTable}
      oncancel={() => (tableDialogOpen = false)}
    />
  {/if}

  {#if noteContextMenu}
    {@const path = noteContextMenu.note.path}
    <NoteContextMenu
      x={noteContextMenu.x}
      y={noteContextMenu.y}
      onclose={() => (noteContextMenu = null)}
      onreveal={() => handleRevealNoteInFS(path)}
      onhtml={() => handleExportNoteHTML(path)}
      onpdf={() => handleExportNotePDF(path)}
      ondelete={() => requestDeleteNote(path)}
    />
  {/if}
</div>
