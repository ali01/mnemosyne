import { describe, it, expect } from 'vitest';
import { marked } from 'marked';
import { wikilinkExtension, extractWikiLinks } from './wikilink-renderer';

describe('WikiLink renderer', () => {
	beforeEach(() => {
		// Configure marked with WikiLink extension
		marked.setOptions(marked.getDefaults());
		marked.use(wikilinkExtension);
	});

	describe('wikilinkExtension', () => {
		it('should render simple WikiLinks', async () => {
			const markdown = 'This is a [[Simple Link]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Simple%20Link" class="wikilink" data-note="Simple Link">Simple Link</a>');
		});

		it('should render WikiLinks with aliases', async () => {
			const markdown = 'Check out [[Target Note|Display Text]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Target%20Note" class="wikilink" data-note="Target Note">Display Text</a>');
		});

		it('should render WikiLinks with sections', async () => {
			const markdown = 'See [[Note Name#Section]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Note%20Name#Section" class="wikilink" data-note="Note Name">Note Name</a>');
		});

		it('should render WikiLinks with both section and alias', async () => {
			const markdown = 'Look at [[Note#Section|Custom Text]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Note#Section" class="wikilink" data-note="Note">Custom Text</a>');
		});

		it('should handle multiple WikiLinks in same text', async () => {
			const markdown = 'Links: [[First]] and [[Second|Second Link]] and [[Third#section]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/First" class="wikilink" data-note="First">First</a>');
			expect(html).toContain('<a href="/notes/Second" class="wikilink" data-note="Second">Second Link</a>');
			expect(html).toContain('<a href="/notes/Third#section" class="wikilink" data-note="Third">Third</a>');
		});

		it('should properly encode URLs', async () => {
			const markdown = 'Link to [[Note with spaces and (parentheses)]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Note%20with%20spaces%20and%20(parentheses)" class="wikilink"');
		});

		it('should handle special characters in note names', async () => {
			const markdown = 'Special: [[Note-with_underscores & symbols!]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Note-with_underscores%20%26%20symbols!" class="wikilink"');
			expect(html).toContain('data-note="Note-with_underscores & symbols!"');
		});

		it('should handle Unicode characters', async () => {
			const markdown = 'Unicode: [[测试笔记]] and [[Café Notes]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/%E6%B5%8B%E8%AF%95%E7%AC%94%E8%AE%B0" class="wikilink" data-note="测试笔记">测试笔记</a>');
			expect(html).toContain('<a href="/notes/Caf%C3%A9%20Notes" class="wikilink" data-note="Café Notes">Café Notes</a>');
		});

		it('should not render broken WikiLinks', async () => {
			const markdown = 'Broken: [[Incomplete and [Normal] brackets.';
			const html = await marked(markdown);

			// Should not contain wikilink class
			expect(html).not.toContain('class="wikilink"');
			// Should preserve original text
			expect(html).toContain('[[Incomplete');
		});

		it('should focus on WikiLink processing without interference', async () => {
			const markdown = 'Text with [[Wiki Link]] only.';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Wiki%20Link" class="wikilink" data-note="Wiki Link">Wiki Link</a>');
		});

		it('should handle nested brackets with current regex limitations', async () => {
			const markdown = 'Text with [[Note [with brackets]]].';
			const html = await marked(markdown);

			// Current regex doesn't fully support nested brackets
			// It will match up to the first closing bracket
			expect(html).toContain('class="wikilink"');
			expect(html).toContain('Note [with brackets');
		});

		it('should trim whitespace from note names and aliases', async () => {
			const markdown = 'Whitespace: [[  Note Name  |  Display Text  ]].';
			const html = await marked(markdown);

			expect(html).toContain('<a href="/notes/Note%20Name" class="wikilink" data-note="Note Name">Display Text</a>');
		});
	});

	describe('extractWikiLinks', () => {
		it('should extract simple WikiLinks', () => {
			const content = 'Text with [[Simple Link]] and more text.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Simple Link', alias: 'Simple Link' }
			]);
		});

		it('should extract WikiLinks with aliases', () => {
			const content = 'Check [[Target|Alias]] out.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Target', alias: 'Alias' }
			]);
		});

		it('should extract WikiLinks with sections', () => {
			const content = 'See [[Note#Section]] for details.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Note', section: 'Section', alias: 'Note' }
			]);
		});

		it('should extract WikiLinks with both section and alias', () => {
			const content = 'Reference [[Note#Section|Custom]].';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Note', section: 'Section', alias: 'Custom' }
			]);
		});

		it('should extract multiple WikiLinks', () => {
			const content = 'Links: [[First]], [[Second|Alias]], and [[Third#section]].';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'First', alias: 'First' },
				{ noteName: 'Second', alias: 'Alias' },
				{ noteName: 'Third', section: 'section', alias: 'Third' }
			]);
		});

		it('should handle duplicate links', () => {
			const content = 'Same link: [[Note]] and [[Note]] again.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Note', alias: 'Note' },
				{ noteName: 'Note', alias: 'Note' }
			]);
		});

		it('should return empty array when no links found', () => {
			const content = 'No links here, just regular text.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([]);
		});

		it('should ignore malformed links', () => {
			const content = 'Malformed: [[Incomplete and [Normal] brackets.';
			const links = extractWikiLinks(content);

			expect(links).toEqual([]);
		});

		it('should handle empty content', () => {
			const links = extractWikiLinks('');

			expect(links).toEqual([]);
		});

		it('should trim whitespace in extracted links', () => {
			const content = 'Whitespace: [[  Note  |  Alias  ]] and [[  Another#section  ]].';
			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Note', alias: 'Alias' },
				{ noteName: 'Another', section: 'section', alias: 'Another' }
			]);
		});

		it('should handle complex content with mixed formatting', () => {
			const content = `
				# Header with [[Header Link]]

				Regular paragraph with [[Basic Link]] and [[Link|With Alias]].

				- List item with [[List Link#section]]
				- Another item

				Code block should not affect: [[Code Link]]

				> Quote with [[Quote Link]]
			`;

			const links = extractWikiLinks(content);

			expect(links).toEqual([
				{ noteName: 'Header Link', alias: 'Header Link' },
				{ noteName: 'Basic Link', alias: 'Basic Link' },
				{ noteName: 'Link', alias: 'With Alias' },
				{ noteName: 'List Link', section: 'section', alias: 'List Link' },
				{ noteName: 'Code Link', alias: 'Code Link' },
				{ noteName: 'Quote Link', alias: 'Quote Link' }
			]);
		});
	});
});
