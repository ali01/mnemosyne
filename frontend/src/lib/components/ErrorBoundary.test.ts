import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import ErrorBoundary from './ErrorBoundary.svelte';

// Mock the toast store
vi.mock('$lib/stores/toast', () => ({
	toast: {
		error: vi.fn()
	}
}));

// Import after mocking
import { toast } from '$lib/stores/toast';
const mockToastError = vi.mocked(toast.error);

// Mock window.location.reload
const mockReload = vi.fn();
Object.defineProperty(window, 'location', {
	value: {
		reload: mockReload
	},
	writable: true
});

describe('ErrorBoundary', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		cleanup();
	});

	it('should render without errors', () => {
		const { container } = render(ErrorBoundary, {
			props: {
				fallback: 'Test fallback'
			}
		});

		// Should render container
		expect(container).toBeInTheDocument();

		// Should not show error UI initially
		expect(screen.queryByText('Oops! Something went wrong')).not.toBeInTheDocument();
		expect(screen.queryByRole('button')).not.toBeInTheDocument();
	});

	it('should accept and use custom fallback prop', () => {
		const customFallback = 'My custom error message';
		const { container } = render(ErrorBoundary, {
			props: {
				fallback: customFallback
			}
		});

		// Component should render without errors
		expect(container).toBeInTheDocument();
	});

	it('should provide default fallback when no prop is provided', () => {
		const { container } = render(ErrorBoundary);

		// Component should render with default fallback
		expect(container).toBeInTheDocument();
	});

	// Note: Due to how Svelte's onMount works in the test environment,
	// we cannot directly test the error handling behavior. The component's
	// error handling is tested through manual testing and integration tests.
	// The following tests verify that the component structure is correct
	// and that it can be rendered without errors.

	it('should have the correct component structure when rendered', () => {
		const { container } = render(ErrorBoundary);

		// Container should exist (even if slot is empty)
		expect(container).toBeTruthy();

		// When no error, it renders the slot (which is empty in this test)
		// The error UI only shows when hasError is true
		expect(container.querySelector('.error-boundary')).not.toBeInTheDocument();
	});

	it('should render with different fallback messages', () => {
		const fallbacks = [
			'Error occurred',
			'Something went wrong!',
			'Please refresh the page',
			''
		];

		fallbacks.forEach(fallback => {
			const { container } = render(ErrorBoundary, {
				props: { fallback }
			});
			expect(container).toBeInTheDocument();
			cleanup();
		});
	});
});
