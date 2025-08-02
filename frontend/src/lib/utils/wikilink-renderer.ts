import type { MarkedExtension, Tokens } from 'marked';

/**
 * WikiLink Extension for Marked.js
 *
 * Supports WikiLink patterns:
 * - [[Note]] - Simple link to a note
 * - [[Note|Alias]] - Link with custom display text
 * - [[Note#Section]] - Link to a specific section
 * - [[Note#Section|Alias]] - Section link with custom display text
 *
 * Limitations:
 * - Does not support nested brackets in note names (e.g., [[Note [with brackets]]])
 * - Note names are used as IDs; requires backend support for name-based lookups
 */

// WikiLink regex pattern
const wikilinkRegex = /\[\[([^\]|#]+)(?:#([^\]|]+))?(?:\|([^\]]+))?\]\]/;

// Interface for wikilink options
export interface WikilinkOptions {
  baseUrl?: string;
  nodeIdResolver?: (noteName: string) => string | Promise<string>;
}

// HTML escape function for security
function escapeHtml(text: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  };
  return text.replace(/[&<>"']/g, (m) => map[m]);
}

/**
 * Creates a WikiLink extension for Marked.js with configurable options
 *
 * @param options - Configuration options for the wikilink extension
 * @returns MarkedExtension configured with the provided options
 */
export function createWikilinkExtension(options: WikilinkOptions = {}): MarkedExtension {
  const { baseUrl = '/notes', nodeIdResolver } = options;

  return {
  extensions: [
    {
      name: 'wikilink',
      level: 'inline',
      start(src: string) {
        return src.indexOf('[[');
      },
      tokenizer(src: string) {
        const match = wikilinkRegex.exec(src);
        if (match) {
          const [fullMatch, noteName, section, alias] = match;
          return {
            type: 'wikilink',
            raw: fullMatch,
            noteName: noteName.trim(),
            section: section?.trim(),
            alias: alias?.trim() || noteName.trim(),
          };
        }
        return undefined;
      },
      renderer(token: any) {
        // Use nodeIdResolver if provided, otherwise use note name as ID
        const nodeId = nodeIdResolver ? nodeIdResolver(token.noteName) : token.noteName;

        // Handle async resolver if needed
        if (nodeId instanceof Promise) {
          console.warn('Async nodeIdResolver not supported in synchronous renderer');
          return `<span class="wikilink-pending" data-note="${escapeHtml(token.noteName)}">${escapeHtml(token.alias)}</span>`;
        }

        const href = `${baseUrl}/${encodeURIComponent(nodeId)}${token.section ? '#' + encodeURIComponent(token.section) : ''}`;
        const escapedNoteName = escapeHtml(token.noteName);
        const escapedAlias = escapeHtml(token.alias);

        return `<a href="${href}" class="wikilink" data-note="${escapedNoteName}">${escapedAlias}</a>`;
      }
    }
  ]
  };
}

// Default extension for backward compatibility
export const wikilinkExtension = createWikilinkExtension();

/**
 * Extracts all WikiLinks from markdown content
 *
 * @param content - The markdown content to parse
 * @returns Array of wikilink objects containing noteName, section, and alias
 */
export function extractWikiLinks(content: string): Array<{ noteName: string; section?: string; alias?: string }> {
  const links: Array<{ noteName: string; section?: string; alias?: string }> = [];
  const regex = new RegExp(wikilinkRegex, 'g');
  let match;

  while ((match = regex.exec(content)) !== null) {
    links.push({
      noteName: match[1].trim(),
      section: match[2]?.trim(),
      alias: match[3]?.trim() || match[1].trim()
    });
  }

  return links;
}
