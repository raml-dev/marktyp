<script lang="ts">
  import { onDestroy } from 'svelte';
  import { editorState, workspaceStore } from '../stores/workspaceStore';
  let sourceTextarea: HTMLTextAreaElement | null = null;

  type SourceSnapshot = { value: string; start: number; end: number };
  let sourceHistory: SourceSnapshot[] = [{ value: $editorState.sourceDraft, start: 0, end: 0 }];

  let sourceHistoryIndex = 0;

  let applyingSourceHistory = false;

  let dualPreviewTimer: number | null = null;

  let lastSourceSelectionStart = 0;

  let lastSourceSelectionEnd = 0;

  function handleSourceDraftInput(event: Event) {
    const target = event.currentTarget as HTMLTextAreaElement;
    const value = target.value;
    workspaceStore.editSource(value);
    if ($editorState.mode === 'Dual') scheduleDualPreviewUpdate(value);
    lastSourceSelectionStart = target.selectionStart ?? value.length;
    lastSourceSelectionEnd = target.selectionEnd ?? value.length;
    recordSourceHistory(value);
  }

  function scheduleDualPreviewUpdate(value: string) {
    if (dualPreviewTimer) {
      window.clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
    }

    dualPreviewTimer = window.setTimeout(() => {
      dualPreviewTimer = null;
      workspaceStore.updatePreview(value);
    }, 140);
  }

  export function flushDualPreviewUpdate() {
    if (dualPreviewTimer) {
      window.clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
      workspaceStore.updatePreview($editorState.sourceDraft);
    }
  }

  function rememberSourceSelection(event: Event) {
    const target = event.currentTarget as HTMLTextAreaElement;
    lastSourceSelectionStart = target.selectionStart ?? 0;
    lastSourceSelectionEnd = target.selectionEnd ?? 0;
    const current = sourceHistory[sourceHistoryIndex];
    if (current) {
      sourceHistory[sourceHistoryIndex] = {
        ...current,
        start: lastSourceSelectionStart,
        end: lastSourceSelectionEnd,
      };
    }
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
    sourceHistory = [{ value: seed, start: 0, end: 0 }];
    sourceHistoryIndex = 0;
  }

  function recordSourceHistory(next: string) {
    if (applyingSourceHistory) return;
    const snapshot = {
      value: next,
      start: sourceTextarea?.selectionStart ?? lastSourceSelectionStart,
      end: sourceTextarea?.selectionEnd ?? lastSourceSelectionEnd,
    };
    const snapshots = sourceHistory.slice(0, sourceHistoryIndex + 1);
    if (snapshots[snapshots.length - 1]?.value === next) snapshots[snapshots.length - 1] = snapshot;
    else snapshots.push(snapshot);
    let size = snapshots.reduce((total, item) => total + item.value.length, 0);
    while (snapshots.length > 1 && (snapshots.length > 100 || size > 4_000_000)) {
      size -= snapshots.shift()!.value.length;
    }
    sourceHistory = snapshots;
    sourceHistoryIndex = sourceHistory.length - 1;
  }

  function applySourceHistorySnapshot(snapshot: SourceSnapshot) {
    applyingSourceHistory = true;
    workspaceStore.editSource(snapshot.value);
    if ($editorState.mode === 'Dual') scheduleDualPreviewUpdate(snapshot.value);
    applyingSourceHistory = false;
    queueMicrotask(() => {
      sourceTextarea?.focus({ preventScroll: true });
      sourceTextarea?.setSelectionRange(snapshot.start, snapshot.end);
      lastSourceSelectionStart = snapshot.start;
      lastSourceSelectionEnd = snapshot.end;
    });
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
    if (wantsUndo) performSourceUndo();
    else performSourceRedo();
  }

  function handleApplySource() {
    flushDualPreviewUpdate();
    workspaceStore.applySource();
  }

  function handleRevertSource() {
    if (dualPreviewTimer) {
      clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
    }
    workspaceStore.revertSource();
    recordSourceHistory($editorState.sourceDraft);
  }

  function applyNextMarkdown(next: string, nextStatus = 'Source draft changed') {
    workspaceStore.editSource(next, true, nextStatus);
    recordSourceHistory(next);
  }

  export async function wrapSelection(prefix: string, suffix: string) {
    if (!sourceTextarea) {
      workspaceStore.message('Source editor unavailable');
      return;
    }

    const { start, end } = resolveSourceSelection();
    const selected = $editorState.sourceDraft.slice(start, end);
    const next = `${$editorState.sourceDraft.slice(0, start)}${prefix}${selected}${suffix}${$editorState.sourceDraft.slice(end)}`;
    applyNextMarkdown(next);

    queueMicrotask(() => {
      sourceTextarea?.focus();
      const cursor = end + prefix.length + suffix.length;
      sourceTextarea?.setSelectionRange(cursor, cursor);
      lastSourceSelectionStart = cursor;
      lastSourceSelectionEnd = cursor;
    });
  }

  export async function insertSnippet(snippet: string) {
    if (!sourceTextarea) {
      workspaceStore.message('Source editor unavailable');
      return;
    }

    const selection = resolveSourceSelection();
    const start = selection.start ?? $editorState.sourceDraft.length;
    const end = selection.end ?? $editorState.sourceDraft.length;
    const next = `${$editorState.sourceDraft.slice(0, start)}${snippet}${$editorState.sourceDraft.slice(end)}`;
    applyNextMarkdown(next);

    queueMicrotask(() => {
      sourceTextarea?.focus();
      const cursor = start + snippet.length;
      sourceTextarea?.setSelectionRange(cursor, cursor);
      lastSourceSelectionStart = cursor;
      lastSourceSelectionEnd = cursor;
    });
  }
  $: hasSourceDraftChanges = $editorState.sourceDraft !== $editorState.markdown;
  export function resetSession() {
    if (dualPreviewTimer) {
      clearTimeout(dualPreviewTimer);
      dualPreviewTimer = null;
    }
    resetSourceHistory($editorState.sourceDraft);
    lastSourceSelectionStart = lastSourceSelectionEnd = 0;
  }
  onDestroy(() => {
    if (dualPreviewTimer) clearTimeout(dualPreviewTimer);
  });
</script>

<div class="panel">
  <div class="panel__header">
    <span>Source</span>
    <div class="panel__actions">
      <span class="panel__hint">Markdown draft</span>
      <button
        class="ghost-button ghost-button--small"
        disabled={!hasSourceDraftChanges}
        on:click={handleRevertSource}
        type="button"
      >
        Revert
      </button>
      <button
        class="ghost-button ghost-button--small"
        disabled={!hasSourceDraftChanges}
        on:click={handleApplySource}
        type="button"
      >
        Apply
      </button>
    </div>
  </div>
  <div class="panel__body">
    <textarea
      bind:this={sourceTextarea}
      value={$editorState.sourceDraft}
      class="source-editor"
      aria-label="Markdown source"
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
