import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import LoadingSpinner from './LoadingSpinner.svelte';

describe('LoadingSpinner', () => {
	afterEach(() => {
		cleanup();
	});

	it('should render with default props', () => {
		render(LoadingSpinner);

		const container = screen.getByRole('status');
		expect(container).toBeInTheDocument();

		const spinnerElement = container.querySelector('.spinner');
		expect(spinnerElement).toBeInTheDocument();

		// Should have screen reader text
		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('should render with small size', () => {
		render(LoadingSpinner, { props: { size: 'small' } });

		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner-small');
	});

	it('should render with medium size (default)', () => {
		render(LoadingSpinner, { props: { size: 'medium' } });

		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner-medium');
	});

	it('should render with large size', () => {
		render(LoadingSpinner, { props: { size: 'large' } });

		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner-large');
	});

	it('should not display message when not provided', () => {
		render(LoadingSpinner);

		// Should only have the default "Loading..." text, no custom message
		const container = screen.getByRole('status');
		expect(container.querySelector('.loading-message')).not.toBeInTheDocument();
		expect(screen.getByText('Loading...')).toBeInTheDocument(); // Default screen reader text
	});

	it('should display message when provided', () => {
		const message = 'Loading data...';
		render(LoadingSpinner, { props: { message } });

		expect(screen.getByText(message)).toBeInTheDocument();
	});

	it('should apply correct CSS classes for small size', () => {
		render(LoadingSpinner, { props: { size: 'small' } });
		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner', 'loading-spinner-small');
	});

	it('should apply correct CSS classes for medium size', () => {
		render(LoadingSpinner, { props: { size: 'medium' } });
		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner', 'loading-spinner-medium');
	});

	it('should apply correct CSS classes for large size', () => {
		render(LoadingSpinner, { props: { size: 'large' } });
		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner', 'loading-spinner-large');
	});

	it('should have spinner animation element', () => {
		render(LoadingSpinner);

		const spinner = document.querySelector('.spinner');
		expect(spinner).toBeInTheDocument();
		expect(spinner).toHaveClass('spinner');
	});

	it('should handle empty message string', () => {
		render(LoadingSpinner, { props: { message: '' } });

		// Empty message should not render loading-message element
		const container = screen.getByRole('status');
		expect(container.querySelector('.loading-message')).not.toBeInTheDocument();

		// Should still have the default screen reader text
		expect(container.querySelector('.sr-only')).toBeInTheDocument();
	});

	it('should handle long message text', () => {
		const longMessage = 'This is a very long loading message that should still display correctly without any issues';
		render(LoadingSpinner, { props: { message: longMessage } });

		expect(screen.getByText(longMessage)).toBeInTheDocument();
	});

	it('should handle special characters in message', () => {
		const specialMessage = 'Loading... 100% çomplete! 🚀';
		render(LoadingSpinner, { props: { message: specialMessage } });

		expect(screen.getByText(specialMessage)).toBeInTheDocument();
	});

	it('should maintain proper structure with both size and message', () => {
		const message = 'Processing...';
		render(LoadingSpinner, { props: { size: 'large', message } });

		const container = screen.getByRole('status');
		expect(container).toHaveClass('loading-spinner-large');

		const spinner = container.querySelector('.spinner');
		expect(spinner).toBeInTheDocument();

		expect(screen.getByText(message)).toBeInTheDocument();
		expect(screen.getByText('Loading...')).toBeInTheDocument(); // Screen reader text
	});
});
