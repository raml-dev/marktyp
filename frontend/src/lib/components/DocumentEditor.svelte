<script lang="ts">
  import { onMount, tick } from 'svelte';
  import RichEditorModal from './RichEditorModal.svelte';
  import { editorState, workspaceStore } from '../stores/workspaceStore';
  import { markdownToHtml, htmlToMarkdown } from '../utils/markdown';
  import { createPreviewRenderer } from '../utils/previewRenderer';
  import { sanitizeHtml } from '../utils/sanitizeHtml';
  const previewRenderer = createPreviewRenderer((source) => workspaceStore.imagePreview(source));
  const { renderPreviewDecorations } = previewRenderer;
  type RichEditorKind = 'math-inline' | 'math-block' | 'mermaid';

  let documentPreviewEl: HTMLElement | null = null;

  let previewRenderScheduled = false;

  let isEditingDocument = false;

  let documentHtmlDraft = markdownToHtml($editorState.markdown);

  let previewDecorationsHtml: string;

  let lastDocumentRange: Range | null = null;
  let pendingInsertionBookmark: Omit<DocumentSnapshot, 'html' | 'scrollTop'> | null = null;
  let pendingCodeExit: { code: Element; at: number } | null = null;

  let richEditorOpen = false;

  let richEditorKind: RichEditorKind = 'math-inline';

  let richEditorSource = '';

  let richEditorTarget: HTMLElement | null = null;

  type DocumentSnapshot = {
    html: string;
    startPath: number[];
    startOffset: number;
    endPath: number[];
    endOffset: number;
    scrollTop: number;
  };
  let documentHistory: DocumentSnapshot[] = [];

  let documentHistoryIndex = -1;

  let applyingDocumentHistory = false;

  let documentEditorChanged = false;

  function pathFromRoot(root: Node, node: Node): number[] {
    const path: number[] = [];
    for (let current: Node | null = node; current && current !== root; current = current.parentNode) {
      if (!current.parentNode) return [];
      path.unshift(Array.prototype.indexOf.call(current.parentNode.childNodes, current));
    }
    return path;
  }

  function nodeFromPath(root: Node, path: number[]): Node {
    let node = root;
    for (const index of path) node = node.childNodes[index] ?? node;
    return node;
  }

  function captureDocumentSnapshot(): DocumentSnapshot | null {
    if (!documentPreviewEl) return null;
    const selection = window.getSelection();
    const range = selection?.rangeCount ? selection.getRangeAt(0) : null;
    const validRange = range && documentPreviewEl.contains(range.commonAncestorContainer) ? range : null;
    return {
      html: documentPreviewEl.innerHTML,
      startPath: validRange ? pathFromRoot(documentPreviewEl, validRange.startContainer) : [],
      startOffset: validRange?.startOffset ?? 0,
      endPath: validRange ? pathFromRoot(documentPreviewEl, validRange.endContainer) : [],
      endOffset: validRange?.endOffset ?? 0,
      scrollTop: documentPreviewEl.parentElement?.scrollTop ?? 0,
    };
  }

  function restoreDocumentSelection(snapshot: DocumentSnapshot) {
    if (!documentPreviewEl) return;
    const start = nodeFromPath(documentPreviewEl, snapshot.startPath);
    const end = nodeFromPath(documentPreviewEl, snapshot.endPath);
    const range = document.createRange();
    try {
      range.setStart(
        start,
        Math.min(
          snapshot.startOffset,
          start.nodeType === Node.TEXT_NODE ? (start.textContent?.length ?? 0) : start.childNodes.length,
        ),
      );
      range.setEnd(
        end,
        Math.min(
          snapshot.endOffset,
          end.nodeType === Node.TEXT_NODE ? (end.textContent?.length ?? 0) : end.childNodes.length,
        ),
      );
    } catch {
      range.selectNodeContents(documentPreviewEl);
      range.collapse(false);
    }
    documentPreviewEl.focus({ preventScroll: true });
    applyDocumentRange(range);
    if (documentPreviewEl.parentElement) documentPreviewEl.parentElement.scrollTop = snapshot.scrollTop;
  }

  function refreshCurrentSnapshot() {
    const snapshot = captureDocumentSnapshot();
    if (snapshot && documentHistory[documentHistoryIndex]?.html === snapshot.html) {
      documentHistory[documentHistoryIndex] = snapshot;
    }
  }

  function resetDocumentHistory() {
    const snapshot = captureDocumentSnapshot();
    documentHistory = snapshot ? [snapshot] : [];
    documentHistoryIndex = snapshot ? 0 : -1;
  }

  function recordDocumentHistory() {
    if (!documentPreviewEl || applyingDocumentHistory || $editorState.mode !== 'Document') return;
    const snapshot = captureDocumentSnapshot();
    if (!snapshot) return;
    const nextHistory = documentHistory.slice(0, documentHistoryIndex + 1);
    if (nextHistory[nextHistory.length - 1]?.html === snapshot.html)
      nextHistory[nextHistory.length - 1] = snapshot;
    else nextHistory.push(snapshot);
    let size = nextHistory.reduce((total, item) => total + item.html.length, 0);
    while (nextHistory.length > 1 && (nextHistory.length > 100 || size > 4_000_000)) {
      size -= nextHistory.shift()!.html.length;
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
    const snapshot = documentHistory[index];
    // The contenteditable subtree is explicitly owned by the editor, not Svelte.
    // eslint-disable-next-line svelte/no-dom-manipulating
    documentPreviewEl.innerHTML = snapshot.html;
    syncDocumentEditorToMarkdown('Document changed');
    renderPreviewDecorations(documentPreviewEl);
    restoreDocumentSelection(snapshot);
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

  function handleDocumentKeyDown(event: KeyboardEvent) {
    if ($editorState.mode !== 'Document') return;
    const hasPrimaryModifier = event.ctrlKey || event.metaKey;
    if (hasPrimaryModifier && event.key.toLowerCase() === 'a' && documentPreviewEl) {
      event.preventDefault();
      const all = document.createRange();
      all.selectNodeContents(documentPreviewEl);
      applyDocumentRange(all);
      return;
    }
    if (event.key !== 'Enter') pendingCodeExit = null;
    if (event.key === 'Enter' && exitCodeBlockAtCaret(event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      return;
    }
    if (event.key === 'Enter' && convertFenceAtCaret()) {
      event.preventDefault();
      return;
    }

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

  function textPointAtOffset(root: Node, offset: number): { node: Node; offset: number } {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let remaining = offset;
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const length = node.textContent?.length ?? 0;
      if (remaining <= length) return { node, offset: remaining };
      remaining -= length;
    }
    return { node: root, offset: root.childNodes.length };
  }

  function editableBlockAtCaret(caret: Range, selector: string): HTMLElement | null {
    if (!documentPreviewEl) return null;
    let candidate: Node | null = caret.startContainer;
    let offset = caret.startOffset;
    if (candidate === documentPreviewEl && offset > 0) {
      candidate = documentPreviewEl.childNodes[offset - 1] ?? null;
      offset = candidate?.textContent?.length ?? 0;
    }
    if (candidate?.nodeType === Node.TEXT_NODE && candidate.parentNode === documentPreviewEl) {
      const text = candidate;
      const paragraph = document.createElement('p');
      // This contenteditable subtree is explicitly owned by the editor rather than Svelte.
      // eslint-disable-next-line svelte/no-dom-manipulating
      documentPreviewEl.insertBefore(paragraph, text);
      paragraph.append(text);
      caret.setStart(text, Math.min(offset, text.textContent?.length ?? 0));
      caret.collapse(true);
      applyDocumentRange(caret);
      return paragraph;
    }
    const element = candidate instanceof Element ? candidate : candidate?.parentElement;
    const block = element?.closest(selector);
    if (!block || block === documentPreviewEl || !documentPreviewEl.contains(block)) return null;
    if (caret.startContainer === documentPreviewEl) {
      caret.selectNodeContents(block);
      caret.collapse(false);
      applyDocumentRange(caret);
    }
    return block as HTMLElement;
  }

  function placeCaretAfterInline(element: HTMLElement) {
    const boundary = document.createElement('span');
    boundary.dataset.marktypInlineBoundary = '';
    boundary.textContent = '\u200b';
    element.after(boundary);
    const next = document.createRange();
    next.selectNodeContents(boundary);
    next.collapse(false);
    applyDocumentRange(next);
  }

  function convertInlineCode(event: InputEvent): boolean {
    if (event.data !== '`' || !documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const caret = selection.getRangeAt(0);
    const block = editableBlockAtCaret(caret, 'p, div, li, h1, h2, h3, h4, h5, h6');
    if (!block) return false;
    const beforeRange = document.createRange();
    beforeRange.selectNodeContents(block);
    beforeRange.setEnd(caret.startContainer, caret.startOffset);
    const before = beforeRange.toString();
    if (before.endsWith('``')) return false;
    const opening = before.lastIndexOf('`');
    if (opening < 0 || before[opening - 1] === '\\' || opening === before.length - 1) return false;
    event.preventDefault();
    const start = textPointAtOffset(block, opening);
    const replacement = document.createRange();
    replacement.setStart(start.node, start.offset);
    replacement.setEnd(caret.startContainer, caret.startOffset);
    replacement.deleteContents();
    const code = document.createElement('code');
    code.textContent = before.slice(opening + 1);
    replacement.insertNode(code);
    placeCaretAfterInline(code);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
    return true;
  }

  type CodeBlockCaret = {
    caret: Range;
    code: HTMLElement;
    pre: HTMLPreElement;
  };

  function resolveCodeBlockCaret(caret: Range): CodeBlockCaret | null {
    if (!documentPreviewEl) return null;
    const element =
      caret.startContainer instanceof Element ? caret.startContainer : caret.startContainer.parentElement;
    const pre = element?.closest('pre');
    if (!pre || !documentPreviewEl.contains(pre)) return null;
    const code = Array.from(pre.children).find(
      (child): child is HTMLElement => child instanceof HTMLElement && child.tagName === 'CODE',
    );
    if (!code) return null;
    if (code.contains(caret.startContainer)) return { caret, code, pre };
    if (!pre.contains(caret.startContainer)) return null;

    const beforeCaret = document.createRange();
    beforeCaret.selectNodeContents(pre);
    beforeCaret.setEnd(caret.startContainer, caret.startOffset);
    const caretTextOffset = beforeCaret.toString().length;
    const codeIndex = Array.prototype.indexOf.call(pre.childNodes, code);
    const children = Array.from(pre.childNodes);
    const leading = children.slice(0, codeIndex);
    const trailing = children.slice(codeIndex + 1);
    code.prepend(...leading);
    code.append(...trailing);

    const point = textPointAtOffset(code, caretTextOffset);
    const normalizedCaret = document.createRange();
    normalizedCaret.setStart(point.node, point.offset);
    normalizedCaret.collapse(true);
    applyDocumentRange(normalizedCaret);
    return { caret: normalizedCaret, code, pre };
  }

  function normalizeCodeBlockSelection() {
    const selection = window.getSelection();
    if (selection?.rangeCount && selection.isCollapsed) resolveCodeBlockCaret(selection.getRangeAt(0));
  }

  function exitCodeBlockAtCaret(force: boolean): boolean {
    if (!documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const codeBlock = resolveCodeBlockCaret(selection.getRangeAt(0));
    if (!codeBlock) return false;
    const { caret, code, pre } = codeBlock;
    const beforeRange = document.createRange();
    beforeRange.selectNodeContents(code);
    beforeRange.setEnd(caret.startContainer, caret.startOffset);
    const afterRange = document.createRange();
    afterRange.selectNodeContents(code);
    afterRange.setStart(caret.startContainer, caret.startOffset);
    const afterText = afterRange.toString();
    const atEnd = afterText === '' || afterText === '\n';
    const beforeText = beforeRange.toString();
    const closingFence = atEnd ? beforeText.match(/(?:^|\n)[ \t]*(?:`{3,}|~{3,})[ \t]*$/) : null;
    if (closingFence) {
      const fenceStart = textPointAtOffset(code, beforeText.length - closingFence[0].length);
      const fenceRange = document.createRange();
      fenceRange.setStart(fenceStart.node, fenceStart.offset);
      fenceRange.setEnd(caret.startContainer, caret.startOffset);
      fenceRange.deleteContents();
    }
    const now = Date.now();
    const repeatedEnter = pendingCodeExit?.code === code && now - pendingCodeExit.at < 1500;
    const shouldExit =
      force || Boolean(closingFence) || (atEnd && (beforeText.endsWith('\n') || repeatedEnter));
    if (!shouldExit) {
      // Keep one trailing newline as a caret anchor. Without it, WebKit can move
      // subsequent input out of an otherwise empty <code> and into its <pre>.
      const newline = document.createTextNode(atEnd && afterText === '' ? '\n\n' : '\n');
      caret.insertNode(newline);
      const next = document.createRange();
      next.setStart(newline, 1);
      next.collapse(true);
      applyDocumentRange(next);
      pendingCodeExit = atEnd ? { code, at: now } : null;
      documentEditorChanged = true;
      syncDocumentEditorToMarkdown();
      recordDocumentHistory();
      return true;
    }
    pendingCodeExit = null;
    const paragraph = document.createElement('p');
    paragraph.append(document.createElement('br'));
    pre.after(paragraph);
    const focusParagraph = () => {
      if (!paragraph.isConnected) return;
      const next = document.createRange();
      next.setStart(paragraph, 0);
      next.collapse(true);
      applyDocumentRange(next);
    };
    focusParagraph();
    requestAnimationFrame(focusParagraph);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
    return true;
  }

  function convertFenceAtCaret(): boolean {
    if (!documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const caret = selection.getRangeAt(0);
    const element =
      caret.startContainer instanceof Element ? caret.startContainer : caret.startContainer.parentElement;
    const block = element?.closest('p, div');
    if (!block || block === documentPreviewEl || !documentPreviewEl.contains(block)) return false;
    const match = block.textContent?.trim().match(/^```([\w-]*)$/);
    if (!match) return false;
    const pre = document.createElement('pre');
    const code = document.createElement('code');
    const language = match[1] || 'txt';
    code.dataset.language = language;
    code.className = `language-${language}`;
    pre.append(code);
    // This contenteditable subtree is owned by the editor rather than Svelte.
    block.replaceWith(pre);
    const next = document.createRange();
    next.selectNodeContents(code);
    next.collapse(true);
    applyDocumentRange(next);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
    return true;
  }

  function convertCompletedInlineCodeAtCaret(): boolean {
    if (!documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const caret = selection.getRangeAt(0);
    const element =
      caret.startContainer instanceof Element ? caret.startContainer : caret.startContainer.parentElement;
    if (element?.closest('code, pre')) return false;
    const block = editableBlockAtCaret(caret, 'p, div, li, h1, h2, h3, h4, h5, h6');
    if (!block) return false;
    const beforeRange = document.createRange();
    beforeRange.selectNodeContents(block);
    beforeRange.setEnd(caret.startContainer, caret.startOffset);
    const before = beforeRange.toString();
    const match = before.match(/(^|[^\\`])`([^`\n]+)`$/);
    if (!match) return false;
    const opening = before.length - match[2].length - 2;
    const start = textPointAtOffset(block, opening);
    const replacement = document.createRange();
    replacement.setStart(start.node, start.offset);
    replacement.setEnd(caret.startContainer, caret.startOffset);
    replacement.deleteContents();
    const code = document.createElement('code');
    code.textContent = match[2];
    replacement.insertNode(code);
    placeCaretAfterInline(code);
    return true;
  }

  function convertCompletedInlineMarkupAtCaret(): boolean {
    if (!documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const caret = selection.getRangeAt(0);
    const element =
      caret.startContainer instanceof Element ? caret.startContainer : caret.startContainer.parentElement;
    if (element?.closest('code, pre')) return false;
    const block = editableBlockAtCaret(caret, 'p, div, li, h1, h2, h3, h4, h5, h6, blockquote');
    if (!block) return false;
    const beforeRange = document.createRange();
    beforeRange.selectNodeContents(block);
    beforeRange.setEnd(caret.startContainer, caret.startOffset);
    const before = beforeRange.toString();
    const patterns: Array<{ expression: RegExp; tag: 'strong' | 'em' | 'del'; content: number }> = [
      { expression: /(^|[^\\*])\*\*([^*\n]+)\*\*$/, tag: 'strong', content: 2 },
      { expression: /(^|[^\\~])~~([^~\n]+)~~$/, tag: 'del', content: 2 },
      { expression: /(^|[^\\*])\*([^*\n]+)\*$/, tag: 'em', content: 2 },
      { expression: /(^|[^\\_])_([^_\n]+)_$/, tag: 'em', content: 2 },
    ];
    const matched = patterns
      .map((pattern) => ({ pattern, match: before.match(pattern.expression) }))
      .find((item) => item.match);
    if (!matched?.match || matched.match.index === undefined) return false;
    const prefixLength = matched.match[1]?.length ?? 0;
    const opening = matched.match.index + prefixLength;
    const start = textPointAtOffset(block, opening);
    const replacement = document.createRange();
    replacement.setStart(start.node, start.offset);
    replacement.setEnd(caret.startContainer, caret.startOffset);
    replacement.deleteContents();
    const formatted = document.createElement(matched.pattern.tag);
    formatted.textContent = matched.match[matched.pattern.content];
    replacement.insertNode(formatted);
    placeCaretAfterInline(formatted);
    return true;
  }

  function convertMarkdownBlockAtCaret(): boolean {
    if (!documentPreviewEl) return false;
    const selection = window.getSelection();
    if (!selection?.rangeCount || !selection.isCollapsed) return false;
    const caret = selection.getRangeAt(0);
    const paragraph = editableBlockAtCaret(caret, 'p, div');
    if (!paragraph || paragraph.querySelector('code, img')) return false;
    const text = paragraph.textContent || '';
    let replacement: HTMLElement | null = null;
    let editable: HTMLElement | null = null;
    const heading = text.match(/^(#{1,6})\s(.+)$/);
    const quote = text.match(/^>\s(.+)$/);
    const unordered = text.match(/^[-*+]\s(.+)$/);
    const ordered = text.match(/^\d+\.\s(.+)$/);
    if (heading) {
      replacement = document.createElement(`h${heading[1].length}`);
      replacement.textContent = heading[2];
      editable = replacement;
    } else if (quote) {
      replacement = document.createElement('blockquote');
      replacement.textContent = quote[1];
      editable = replacement;
    } else if (unordered || ordered) {
      replacement = document.createElement(unordered ? 'ul' : 'ol');
      editable = document.createElement('li');
      editable.textContent = (unordered || ordered)![1];
      replacement.append(editable);
    } else if (text === '---') {
      replacement = document.createElement('hr');
      editable = document.createElement('p');
      editable.append(document.createElement('br'));
      paragraph.replaceWith(replacement, editable);
    }
    if (!replacement || !editable) return false;
    if (text !== '---') paragraph.replaceWith(replacement);
    const next = document.createRange();
    next.selectNodeContents(editable);
    next.collapse(false);
    applyDocumentRange(next);
    return true;
  }

  function handleDocumentBeforeInput(event: InputEvent) {
    if ($editorState.mode !== 'Document') return;
    if (event.inputType === 'historyUndo' || event.inputType === 'historyRedo') event.preventDefault();
    else if (event.data === '`') convertInlineCode(event);
  }

  function syncDocumentEditorToMarkdown(nextStatus = 'Document changed') {
    if (!documentPreviewEl) {
      return;
    }

    const next = htmlToMarkdown(documentPreviewEl.innerHTML);
    workspaceStore.editDocument(next, nextStatus);
  }

  function focusDocumentEditor(): HTMLElement | null {
    if ($editorState.mode !== 'Document' || !documentPreviewEl) return null;
    if (document.activeElement !== documentPreviewEl) {
      const savedRange = lastDocumentRange?.cloneRange() ?? null;
      documentPreviewEl.focus({ preventScroll: true });
      if (
        savedRange &&
        documentPreviewEl.contains(savedRange.startContainer) &&
        documentPreviewEl.contains(savedRange.endContainer)
      )
        applyDocumentRange(savedRange);
    }
    return documentPreviewEl;
  }

  function resolveDocumentRange(root: HTMLElement): Range | null {
    if (pendingInsertionBookmark) {
      const bookmark = pendingInsertionBookmark;
      pendingInsertionBookmark = null;
      const start = nodeFromPath(root, bookmark.startPath);
      const end = nodeFromPath(root, bookmark.endPath);
      const preserved = document.createRange();
      try {
        preserved.setStart(
          start,
          Math.min(
            bookmark.startOffset,
            start.nodeType === Node.TEXT_NODE ? (start.textContent?.length ?? 0) : start.childNodes.length,
          ),
        );
        preserved.setEnd(
          end,
          Math.min(
            bookmark.endOffset,
            end.nodeType === Node.TEXT_NODE ? (end.textContent?.length ?? 0) : end.childNodes.length,
          ),
        );
        return preserved;
      } catch {
        // Continue with the live or last known range when the document changed externally.
      }
    }
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
    isEditingDocument = true;
    documentPreviewEl?.focus({ preventScroll: true });
    const selection = window.getSelection();
    if (!selection) {
      return;
    }
    selection.removeAllRanges();
    selection.addRange(range);
    lastDocumentRange = range.cloneRange();
  }

  export function runDocumentCommand(command: string, value?: string) {
    const root = focusDocumentEditor();
    if (!root) {
      return false;
    }
    document.execCommand(command, false, value);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();

    return true;
  }

  export function insertDocumentHtml(html: string) {
    const root = focusDocumentEditor();
    if (!root) {
      return false;
    }

    const range = resolveDocumentRange(root);
    if (!range) {
      workspaceStore.message('Unable to resolve document cursor');
      return false;
    }

    const markerId = `marktyp-caret-${Math.random().toString(36).slice(2)}`;
    const fragment = range.createContextualFragment(
      `${html}<span data-marktyp-caret="${markerId}">\u200b</span>`,
    );
    range.deleteContents();
    const isTable = /^\s*<table[\s>]/i.test(html);
    const rangeElement =
      range.startContainer instanceof Element ? range.startContainer : range.startContainer.parentElement;
    const paragraph = isTable ? rangeElement?.closest('p, h1, h2, h3, h4, h5, h6, blockquote') : null;
    if (paragraph && root.contains(paragraph)) {
      const tailRange = document.createRange();
      tailRange.setStart(range.startContainer, range.startOffset);
      tailRange.selectNodeContents(paragraph);
      tailRange.setStart(range.startContainer, range.startOffset);
      const tail = tailRange.extractContents();
      const trailing = paragraph.cloneNode(false) as HTMLElement;
      trailing.append(tail);
      paragraph.after(fragment);
      if (trailing.textContent || trailing.querySelector('img, code, br')) {
        const insertedTable = paragraph.nextElementSibling;
        const insertionEnd = insertedTable?.nextElementSibling;
        insertionEnd?.after(trailing);
      }
    } else {
      range.insertNode(fragment);
    }

    const editTarget = root.querySelector('[data-marktyp-edit-target]') as HTMLElement | null;
    const marker = root.querySelector(`[data-marktyp-caret="${markerId}"]`) as HTMLElement | null;
    const nextRange = document.createRange();
    if (editTarget) {
      editTarget.removeAttribute('data-marktyp-edit-target');
      nextRange.selectNodeContents(editTarget);
    } else if (marker) {
      nextRange.setStartBefore(marker);
      nextRange.collapse(true);
    }
    if (editTarget || marker) applyDocumentRange(nextRange);
    marker?.remove();

    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    renderPreviewDecorations(documentPreviewEl);
    recordDocumentHistory();

    return true;
  }

  function handleDocumentPaste(event: ClipboardEvent) {
    event.preventDefault();
    refreshCurrentSnapshot();
    const html = event.clipboardData?.getData('text/html');
    const text = event.clipboardData?.getData('text/plain') ?? '';
    if (html) insertDocumentHtml(sanitizeHtml(html));
    else insertText(text);
  }

  function handleDocumentDrop(event: DragEvent) {
    event.preventDefault();
    refreshCurrentSnapshot();
    const text = event.dataTransfer?.getData('text/plain') ?? '';
    if (text) insertText(text);
  }

  function handleDocumentInput() {
    normalizeCodeBlockSelection();
    if (!convertCompletedInlineCodeAtCaret() && !convertCompletedInlineMarkupAtCaret()) {
      convertMarkdownBlockAtCaret();
    }
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
    if (documentEditorChanged) {
      syncDocumentEditorToMarkdown('Document synced');
    }
    documentEditorChanged = false;
    isEditingDocument = false;
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
    refreshCurrentSnapshot();
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

    const link = target.closest('a');
    if (link) {
      event.preventDefault();
      if ($editorState.mode !== 'Document' || event.ctrlKey || event.metaKey)
        workspaceStore.openLink(link.href);
      return;
    }
    if ($editorState.mode !== 'Document') return;
    const diagram = target.closest('marktyp-mermaid') as HTMLElement | null;
    if (diagram) {
      event.preventDefault();
      openRichEditor(diagram, 'mermaid');
      return;
    }
    if (target instanceof HTMLInputElement && target.type === 'checkbox') {
      target.toggleAttribute('checked', target.checked);
      handleDocumentInput();
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
    documentEditorChanged = true;
    recordDocumentHistory();
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

  async function schedulePreviewDecorations() {
    if (previewRenderScheduled || !documentPreviewEl) {
      return;
    }

    previewRenderScheduled = true;
    await tick();
    previewRenderScheduled = false;
    renderPreviewDecorations(documentPreviewEl);
  }
  $: markdownSnapshot = $editorState.markdown;
  $: renderedDocumentHtml = markdownToHtml(markdownSnapshot);
  $: if (!isEditingDocument) documentHtmlDraft = renderedDocumentHtml;
  $: previewDecorationsHtml = $editorState.mode === 'Document' ? documentHtmlDraft : renderedDocumentHtml;
  $: if (
    documentPreviewEl &&
    previewDecorationsHtml !== undefined &&
    !($editorState.mode === 'Document' && isEditingDocument)
  )
    schedulePreviewDecorations();
  onMount(() => {
    document.addEventListener('selectionchange', handleDocumentSelectionEvent);
    return () => {
      previewRenderer.dispose();
      document.removeEventListener('selectionchange', handleDocumentSelectionEvent);
    };
  });
  export function resetSession() {
    documentHistory = [];
    documentHistoryIndex = -1;
    documentEditorChanged = false;
    isEditingDocument = false;
    lastDocumentRange = null;
    pendingInsertionBookmark = null;
    richEditorTarget = null;
    richEditorOpen = false;
  }
  export function wrapSelection(prefix: string, suffix: string) {
    const root = focusDocumentEditor();
    if (!root) return;
    const range = resolveDocumentRange(root);
    if (range) applyDocumentRange(range);
    if (prefix === '**' && suffix === '**') document.execCommand('bold');
    else if (prefix === '*' && suffix === '*') document.execCommand('italic');
    else {
      const selected = window.getSelection()?.toString() || '';
      insertTextAtDocumentCursor(prefix + selected + suffix);
    }
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
  }
  export function preserveInsertionPoint() {
    if (!documentPreviewEl) return;
    const selection = window.getSelection();
    const liveRange =
      selection?.rangeCount && documentPreviewEl.contains(selection.getRangeAt(0).commonAncestorContainer)
        ? selection.getRangeAt(0)
        : null;
    const range = liveRange ?? lastDocumentRange;
    if (!range) return;
    pendingInsertionBookmark = {
      startPath: pathFromRoot(documentPreviewEl, range.startContainer),
      startOffset: range.startOffset,
      endPath: pathFromRoot(documentPreviewEl, range.endContainer),
      endOffset: range.endOffset,
    };
  }

  export function insertImage(source: string) {
    const image = document.createElement('img');
    image.src = source;
    image.alt = 'image';
    image.dataset.marktypSource = source;
    return insertDocumentHtml(image.outerHTML);
  }

  export function insertLink(url: string) {
    const selection = window.getSelection();
    if (selection?.toString()) return runDocumentCommand('createLink', url);
    const link = document.createElement('a');
    link.href = url;
    link.textContent = 'link';
    link.setAttribute('data-marktyp-edit-target', '');
    return insertDocumentHtml(link.outerHTML);
  }

  export function insertInlineCode() {
    const selected = window.getSelection()?.toString() || 'code';
    const escaped = selected.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return insertDocumentHtml(`<code data-marktyp-edit-target>${escaped}</code>`);
  }

  export function insertText(text: string) {
    const root = focusDocumentEditor();
    if (!root) return;
    const range = resolveDocumentRange(root);
    if (range) applyDocumentRange(range);
    insertTextAtDocumentCursor(text);
    documentEditorChanged = true;
    syncDocumentEditorToMarkdown();
    recordDocumentHistory();
  }
</script>

{#if $editorState.mode === 'Document'}
  <div
    bind:this={documentPreviewEl}
    class="document-preview document-editor__content document-editor__content--editable"
    contenteditable="true"
    role="textbox"
    tabindex="0"
    aria-label="Document editor"
    aria-multiline="true"
    on:blur={handleDocumentBlur}
    on:click={handleDocumentClick}
    on:focus={handleDocumentFocus}
    on:beforeinput={handleDocumentBeforeInput}
    on:input={handleDocumentInput}
    on:paste={handleDocumentPaste}
    on:dragover|preventDefault
    on:drop={handleDocumentDrop}
    on:keydown|capture={handleDocumentKeyDown}
    on:keyup={handleDocumentSelectionEvent}
    on:mouseup={handleDocumentSelectionEvent}
    spellcheck="false"
  >
    <!-- HTML passes through the Markdown sanitizer before reaching the editor. -->
    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html documentHtmlDraft}
  </div>
{:else}
  <!-- Preview links are handled centrally so they never navigate the desktop webview. -->
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-noninteractive-element-interactions -->
  <article
    bind:this={documentPreviewEl}
    class="document-preview document-editor__content"
    on:click={handleDocumentClick}
  >
    <!-- Sanitized in markdownToHtml. -->
    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html renderedDocumentHtml}
  </article>
{/if}
{#if richEditorOpen}
  <RichEditorModal
    bind:source={richEditorSource}
    kind={richEditorKind}
    onapply={handleRichEditorApply}
    oncancel={handleRichEditorCancel}
  />
{/if}
