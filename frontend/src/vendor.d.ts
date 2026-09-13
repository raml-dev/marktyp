// Minimal typed contracts for the APIs used from packages that ship no declarations.
declare module 'markdown-it' {
  export default class MarkdownIt {
    constructor(options?: { breaks?: boolean; html?: boolean; linkify?: boolean });
    use(plugin: unknown, options?: Record<string, unknown>): this;
    renderer: {
      rules: Record<
        string,
        | ((
            tokens: Array<{
              attrGet(name: string): string | null;
              attrSet(name: string, value: string): void;
            }>,
            index: number,
            options: unknown,
            environment: unknown,
            renderer: { renderToken(tokens: unknown[], index: number, options: unknown): string },
          ) => string)
        | undefined
      >;
    };
    render(markdown: string): string;
  }
}
declare module 'markdown-it-footnote' {
  const plugin: unknown;
  export default plugin;
}
declare module 'markdown-it-task-lists' {
  const plugin: unknown;
  export default plugin;
}
declare module 'turndown' {
  export default class TurndownService {
    constructor(options?: { bulletListMarker?: string; codeBlockStyle?: string; headingStyle?: string });
    use(plugin: unknown): this;
    addRule(
      name: string,
      rule: {
        filter: string | string[] | ((node: HTMLElement) => boolean);
        replacement: (content: string, node: HTMLElement) => string;
      },
    ): this;
    turndown(html: string | Node): string;
  }
}
declare module 'turndown-plugin-gfm' {
  export const gfm: unknown;
}
