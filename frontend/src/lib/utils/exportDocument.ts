import { markdownToHtml } from './markdown';
import { createPreviewRenderer } from './previewRenderer';

// Render an immutable source snapshot. Never clone a possibly stale live preview.
export async function renderExportHtml(
  markdown: string,
  resolveImage?: (source: string) => Promise<string>,
): Promise<string> {
  const container = document.createElement('div');
  container.innerHTML = markdownToHtml(markdown);
  const renderer = createPreviewRenderer(resolveImage);
  try {
    await renderer.renderAll(container);
    container
      .querySelectorAll('[contenteditable]')
      .forEach((node) => node.removeAttribute('contenteditable'));
    container
      .querySelectorAll('[data-marktyp-source]')
      .forEach((node) => node.removeAttribute('data-marktyp-source'));
    return container.innerHTML;
  } finally {
    renderer.dispose();
  }
}
