import { describe, it, expect, vi, beforeEach } from 'vitest';
import { openInObsidian } from './obsidian';

describe('openInObsidian', () => {
  let clickedHref: string;

  beforeEach(() => {
    clickedHref = '';
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = { href: '', click: () => { clickedHref = el.href; } } as any;
      return el;
    });
  });

  it('constructs correct URI for a simple path', () => {
    openInObsidian('walros', 'memex/concepts/network.md');
    expect(clickedHref).toBe(
      'obsidian://open?vault=walros&file=memex%2Fconcepts%2Fnetwork'
    );
  });

  it('handles paths without .md extension', () => {
    openInObsidian('walros', 'memex/concepts/network');
    expect(clickedHref).toBe(
      'obsidian://open?vault=walros&file=memex%2Fconcepts%2Fnetwork'
    );
  });

  it('handles paths with spaces', () => {
    openInObsidian('walros', 'memex/my notes/hello world.md');
    expect(clickedHref).toBe(
      'obsidian://open?vault=walros&file=memex%2Fmy%20notes%2Fhello%20world'
    );
  });

  it('handles root-level files', () => {
    openInObsidian('research', 'index.md');
    expect(clickedHref).toBe(
      'obsidian://open?vault=research&file=index'
    );
  });

  it('handles vault names with special characters', () => {
    openInObsidian('my vault', 'note.md');
    expect(clickedHref).toBe(
      'obsidian://open?vault=my%20vault&file=note'
    );
  });
});
