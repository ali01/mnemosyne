import type { MarkedExtension, Tokens } from 'marked';

// WikiLink patterns: [[Note]], [[Note|Alias]], [[Note#Section]]
const wikilinkRegex = /\[\[([^\]|#]+)(?:#([^\]|]+))?(?:\|([^\]]+))?\]\]/;

export const wikilinkExtension: MarkedExtension = {
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
        // We need to find the node ID for this note name
        // For now, we'll use the note name as the ID (this should be improved with a lookup)
        const href = `/notes/${encodeURIComponent(token.noteName)}${token.section ? '#' + encodeURIComponent(token.section) : ''}`;
        return `<a href="${href}" class="wikilink" data-note="${token.noteName}">${token.alias}</a>`;
      }
    }
  ]
};

// Helper function to extract all WikiLinks from markdown content
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