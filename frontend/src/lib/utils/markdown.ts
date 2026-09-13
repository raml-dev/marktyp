import MarkdownIt from 'markdown-it';
import markdownItFootnote from 'markdown-it-footnote';
import markdownItTaskLists from 'markdown-it-task-lists';
import TurndownService from 'turndown';
import { gfm } from 'turndown-plugin-gfm';
import { sanitizeHtml } from './sanitizeHtml';
import { markdownDestination } from './insertions';

const markdownRenderer = new MarkdownIt({
  breaks: true,
  html: true,
  linkify: true,
});

markdownRenderer.use(markdownItFootnote);
markdownRenderer.use(markdownItTaskLists as never, { enabled: true, label: true });
const defaultImageRenderer = markdownRenderer.renderer.rules.image;
markdownRenderer.renderer.rules.image = (tokens, index, options, environment, renderer) => {
  const token = tokens[index];
  const source = token.attrGet('src') || '';
  if (source && !/^(https?:|data:|blob:)/i.test(source)) {
    try {
      token.attrSet('data-marktyp-source', decodeURI(source));
    } catch {
      token.attrSet('data-marktyp-source', source);
    }
  }
  return defaultImageRenderer
    ? defaultImageRenderer(tokens, index, options, environment, renderer)
    : renderer.renderToken(tokens, index, options);
};

const turndown = new TurndownService({
  bulletListMarker: '-',
  codeBlockStyle: 'fenced',
  headingStyle: 'atx',
});

turndown.use(gfm);

turndown.addRule('marktypMermaid', {
  filter: 'marktyp-mermaid',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `\n\n\`\`\`mermaid\n${source}\n\`\`\`\n\n`;
  },
});

turndown.addRule('marktypMathBlock', {
  filter: 'marktyp-math-block',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `\n\n$$\n${source}\n$$\n\n`;
  },
});

turndown.addRule('marktypMathInline', {
  filter: 'marktyp-math-inline',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `$${source}$`;
  },
});

turndown.addRule('fencedCodeBlock', {
  filter: (node: Node) => {
    return node.nodeName === 'PRE' && Boolean((node as Element).querySelector('code'));
  },
  replacement: (_content: string, node: Element) => {
    const code = node.querySelector('code');
    const language = code?.getAttribute('data-language') || '';
    const className = code?.getAttribute('class') || '';
    const classMatch = className.match(/language-([\w-]+)/);
    const resolvedLanguage = language || classMatch?.[1] || '';
    const content = code?.textContent?.replace(/\n$/, '') || '';
    return `\n\n\`\`\`${resolvedLanguage}\n${content}\n\`\`\`\n\n`;
  },
});

turndown.addRule('marktypInlineBoundary', {
  filter: (node: Node) => node instanceof Element && node.hasAttribute('data-marktyp-inline-boundary'),
  replacement: (content: string) => content.replace(/\u200b/g, ''),
});

turndown.addRule('marktypImageSource', {
  filter: (node: Node) =>
    node.nodeName === 'IMG' && Boolean((node as Element).getAttribute('data-marktyp-source')),
  replacement: (_content: string, node: Element) => {
    const source = markdownDestination(node.getAttribute('data-marktyp-source') || '');
    const alt = (node.getAttribute('alt') || 'image').replace(/]/g, '\\]');
    return `![${alt}](${source})`;
  },
});

turndown.addRule('taskItemCheckbox', {
  filter: (node: Node) => {
    const element = node as HTMLInputElement;
    return node.nodeName === 'INPUT' && element.type === 'checkbox';
  },
  replacement: (_content: string, node: Element) =>
    (node as HTMLInputElement).checked || node.hasAttribute('checked') ? '[x]' : '[ ]',
});

export function titleFromMarkdown(markdown: string, fallback = 'Untitled'): string {
  const lines = markdown.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('# ')) {
      return trimmed.slice(2).trim();
    }
  }

  return fallback;
}

export function markdownToHtml(markdown: string): string {
  const normalized = normalizeSpecialMarkdown(markdown);
  return sanitizeHtml(markdownRenderer.render(normalized));
}

export function htmlToMarkdown(html: string): string {
  const markdown = turndown.turndown(html);
  return cleanupMarkdown(markdown);
}

function normalizeSpecialMarkdown(markdown: string): string {
  // Consume code and math in one pass: dollar signs inside code/diagram sources
  // must never be reinterpreted as math, nor escaped attributes processed twice.
  return markdown.replace(
    /(^[ \t]*(`{3,}|~{3,})[^\n]*\n[\s\S]*?^[ \t]*\2[ \t]*(?=\n|$))|(`+)([^\n]*?)\3|(\$\$[\s\S]*?\$\$)|(?<![\\\w])\$([^\n$]+)\$/gm,
    (
      match,
      fence: string,
      _delimiter: string,
      inline: string,
      _code: string,
      block: string,
      math: string,
    ) => {
      if (fence) {
        const mermaid = fence.match(
          /^[ \t]*(?:`{3,}|~{3,})mermaid\s*\n([\s\S]*?)\n[ \t]*(?:`{3,}|~{3,})[ \t]*$/,
        );
        return mermaid
          ? `<marktyp-mermaid source="${escapeAttribute(mermaid[1].trim())}"></marktyp-mermaid>\n`
          : fence;
      }
      if (inline) return match;
      if (block)
        return `<marktyp-math-block source="${escapeAttribute(block.slice(2, -2).trim())}"></marktyp-math-block>`;
      return `<marktyp-math-inline source="${escapeAttribute(math.trim())}"></marktyp-math-inline>`;
    },
  );
}

function cleanupMarkdown(markdown: string): string {
  return markdown
    .replace(/\n{3,}/g, '\n\n')
    .replace(/^\s+/, '')
    .trim();
}

function escapeAttribute(value: string): string {
  return escapeHtml(value).replace(/"/g, '&quot;').replace(/\n/g, '&#10;');
}

function unescapeAttribute(value: string): string {
  return value
    .replace(/&#10;/g, '\n')
    .replace(/&quot;/g, '"')
    .replace(/&amp;/g, '&');
}

function escapeHtml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
