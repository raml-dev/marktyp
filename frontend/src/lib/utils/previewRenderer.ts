import { BoundedCache } from './boundedCache';

// Each editor/export owns cancellation state; exports cannot cancel the live preview.
export function createPreviewRenderer(resolveImage?: (source: string) => Promise<string>) {
  let disposed = false;
  let mermaidInitialized = false;
  let mermaidApi: typeof import('mermaid').default | null = null;
  let mermaidRenderTimer: ReturnType<typeof setTimeout> | null = null;
  let mermaidRenderGeneration = 0;
  const mathInlineCache = new BoundedCache<string, string>();
  const mathBlockCache = new BoundedCache<string, string>();
  const mermaidSvgCache = new BoundedCache<string, string>();
  async function renderMathNodes(container: HTMLElement | null) {
    if (!container?.querySelector('marktyp-math-inline, marktyp-math-block')) return;
    const { default: katex } = await import('katex');
    if (disposed) return;

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

  async function renderCodeNodes(container: HTMLElement | null) {
    if (container?.isContentEditable || !container?.querySelector('pre code')) return;
    const { default: hljs } = await import('highlight.js');
    if (disposed) return;

    const codeNodes = container.querySelectorAll('pre code');
    codeNodes.forEach((node) => {
      if (!(node as HTMLElement).dataset.highlighted) hljs.highlightElement(node as HTMLElement);
    });
  }

  async function renderMermaidNodes(container: HTMLElement | null, generation: number) {
    if (!container) {
      return;
    }

    if (!container.querySelector('marktyp-mermaid')) return;
    if (!mermaidApi) {
      const mod = await import('mermaid');
      mermaidApi = mod.default;
    }

    if (!mermaidInitialized) {
      mermaidApi.initialize({
        startOnLoad: false,
        securityLevel: 'strict',
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

  async function renderLocalImages(container: HTMLElement | null) {
    if (!container || !resolveImage) return;
    const images = Array.from(container.querySelectorAll('img'));
    await Promise.all(
      images.map(async (image) => {
        const source = image.getAttribute('data-marktyp-source') || image.getAttribute('src') || '';
        if (!source || /^(https?:|data:|blob:)/i.test(source)) return;
        image.setAttribute('data-marktyp-source', source);
        try {
          const preview = await resolveImage(source);
          if (!disposed && container.contains(image)) image.src = preview;
        } catch (error) {
          console.error('Local image preview failed', error);
        }
      }),
    );
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
      void renderMermaidNodes(container, generation).catch((error) =>
        console.error('Diagram rendering failed', error),
      );
    }, 80);
  }

  function renderPreviewDecorations(container: HTMLElement | null) {
    void renderMathNodes(container).catch((error) => console.error('Math rendering failed', error));
    void renderCodeNodes(container).catch((error) => console.error('Code rendering failed', error));
    scheduleMermaidRender(container);
    void renderLocalImages(container);
  }
  return {
    renderPreviewDecorations,
    async renderAll(container: HTMLElement) {
      await Promise.all([
        renderMathNodes(container),
        renderCodeNodes(container),
        renderLocalImages(container),
      ]);
      await renderMermaidNodes(container, ++mermaidRenderGeneration);
    },
    dispose() {
      disposed = true;
      if (mermaidRenderTimer) clearTimeout(mermaidRenderTimer);
      mermaidRenderGeneration++;
      mathInlineCache.clear();
      mathBlockCache.clear();
      mermaidSvgCache.clear();
    },
  };
}
