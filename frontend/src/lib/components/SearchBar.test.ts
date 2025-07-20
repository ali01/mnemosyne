import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import SearchBar from './SearchBar.svelte';

// Mock the debounce utility
vi.mock('$lib/utils/debounce', () => ({
	debounce: vi.fn((fn: Function) => fn)
}));

// Mock fetch
globalThis.fetch = vi.fn();

describe('SearchBar', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		cleanup();
	});

	it('should render with default placeholder', () => {
		render(SearchBar);

		const searchInput = screen.getByPlaceholderText('Search nodes...');
		expect(searchInput).toBeInTheDocument();
		expect(searchInput).toHaveAttribute('type', 'text');
		expect(searchInput).toHaveAttribute('autocomplete', 'off');
	});

	it('should render with custom placeholder', () => {
		const customPlaceholder = 'Search for items...';
		render(SearchBar, {
			props: { placeholder: customPlaceholder }
		});

		expect(screen.getByPlaceholderText(customPlaceholder)).toBeInTheDocument();
	});

	it('should have search icon', () => {
		render(SearchBar);

		const searchIcon = document.querySelector('.search-icon');
		expect(searchIcon).toBeInTheDocument();
		expect(searchIcon?.tagName).toBe('svg');
	});

	it('should not show dropdown initially', () => {
		render(SearchBar);

		expect(screen.queryByRole('button')).not.toBeInTheDocument();
		expect(document.querySelector('.search-dropdown')).not.toBeInTheDocument();
	});

	it('should have correct component structure', () => {
		render(SearchBar);

		// Should have search container
		expect(document.querySelector('.search-container')).toBeInTheDocument();
		expect(document.querySelector('.search-input-wrapper')).toBeInTheDocument();

		// Should have search input with correct attributes
		const searchInput = screen.getByPlaceholderText('Search nodes...');
		expect(searchInput).toHaveClass('search-input');
	});

	it('should handle user input', async () => {
		render(SearchBar);

		const searchInput = screen.getByPlaceholderText('Search nodes...');

		await userEvent.type(searchInput, 'test');

		expect(searchInput).toHaveValue('test');
	});

	it('should render search icon SVG correctly', () => {
		render(SearchBar);

		const searchIcon = document.querySelector('.search-icon');
		expect(searchIcon).toBeInTheDocument();
		expect(searchIcon).toHaveAttribute('width', '16');
		expect(searchIcon).toHaveAttribute('height', '16');
		expect(searchIcon).toHaveAttribute('viewBox', '0 0 24 24');
		expect(searchIcon).toHaveAttribute('fill', 'none');
		expect(searchIcon).toHaveAttribute('stroke', 'currentColor');
	});

	it('should have correct CSS classes', () => {
		render(SearchBar);

		expect(document.querySelector('.search-container')).toBeInTheDocument();
		expect(document.querySelector('.search-input-wrapper')).toBeInTheDocument();
		expect(document.querySelector('.search-icon')).toBeInTheDocument();
		expect(document.querySelector('.search-input')).toBeInTheDocument();
	});


	it('should maintain focus capabilities', async () => {
		render(SearchBar);

		const searchInput = screen.getByPlaceholderText('Search nodes...');

		searchInput.focus();
		expect(searchInput).toHaveFocus();
	});

	it('should handle empty placeholder', () => {
		render(SearchBar, {
			props: { placeholder: '' }
		});

		const searchInput = document.querySelector('.search-input');
		expect(searchInput).toBeInTheDocument();
		expect(searchInput).toHaveAttribute('placeholder', '');
	});

	it('should test component props interface', () => {
		// Test that component accepts expected props
		const props = { placeholder: 'Test placeholder' };

		expect(() => {
			render(SearchBar, { props });
		}).not.toThrow();
	});

	it('should render input with correct attributes', () => {
		render(SearchBar);

		const searchInput = screen.getByPlaceholderText('Search nodes...');

		expect(searchInput).toHaveAttribute('type', 'text');
		expect(searchInput).toHaveAttribute('autocomplete', 'off');
		expect(searchInput).toHaveClass('search-input');
	});

	it('should handle long placeholder text', () => {
		const longPlaceholder = 'This is a very long placeholder text that should still work correctly even when it contains many characters';

		render(SearchBar, {
			props: { placeholder: longPlaceholder }
		});

		expect(screen.getByPlaceholderText(longPlaceholder)).toBeInTheDocument();
	});

	it('should handle special characters in placeholder', () => {
		const specialPlaceholder = 'Search with special chars: 🔍 100% & more!';

		render(SearchBar, {
			props: { placeholder: specialPlaceholder }
		});

		expect(screen.getByPlaceholderText(specialPlaceholder)).toBeInTheDocument();
	});

	it('should render component successfully', () => {
		// Basic test to ensure component renders
		expect(() => {
			render(SearchBar);
		}).not.toThrow();
	});
});
