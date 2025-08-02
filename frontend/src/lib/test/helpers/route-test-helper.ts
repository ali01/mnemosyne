import { vi } from 'vitest';

// Type definitions for Svelte component mocks
interface SvelteComponentMock {
	name: string;
	$$: {
		on_mount: any[];
		on_destroy: any[];
		before_update: any[];
		after_update: any[];
		context: Map<any, any>;
		callbacks: Record<string, any>;
		dirty: any[];
		skip_bound: boolean;
		root: any;
		ctx: any;
		fragment: any;
		update: () => void;
		not_equal: () => boolean;
		bound: Record<string, any>;
	};
	$$render: () => string;
	$$set: () => void;
	$on: () => void;
	$destroy: () => void;
	$capture_state: () => Record<string, any>;
	$inject_state: () => void;
}

/**
 * Creates a minimal Svelte component mock for testing
 * @param name - The name of the component
 * @param renderOutput - The HTML output to render
 * @returns A mock Svelte component object
 */
export function createComponentMock(name: string, renderOutput: string): SvelteComponentMock {
	return {
		name,
		$$: {
			on_mount: [],
			on_destroy: [],
			before_update: [],
			after_update: [],
			context: new Map(),
			callbacks: {},
			dirty: [],
			skip_bound: false,
			root: null,
			ctx: null,
			fragment: null,
			update: () => {},
			not_equal: () => true,
			bound: {}
		},
		$$render: () => renderOutput,
		$$set: () => {},
		$on: () => {},
		$destroy: () => {},
		$capture_state: () => ({}),
		$inject_state: () => {}
	};
}

/**
 * Mock SvelteKit stores with proper cleanup
 * @returns Mock store objects with subscribe methods
 */
export function mockSvelteKitStores() {
	const unsubscribers: (() => void)[] = [];
	const page = {
		subscribe: vi.fn((callback) => {
			callback({
				url: new URL('http://localhost'),
				params: {},
				route: { id: '/' },
				status: 200,
				error: null,
				data: {},
				form: undefined
			});
			const unsubscribe = () => {};
			unsubscribers.push(unsubscribe);
			return unsubscribe;
		})
	};

	const navigating = {
		subscribe: vi.fn((callback) => {
			callback(null);
			const unsubscribe = () => {};
			unsubscribers.push(unsubscribe);
			return unsubscribe;
		})
	};

	const updated = {
		subscribe: vi.fn((callback) => {
			callback(false);
			const unsubscribe = () => {};
			unsubscribers.push(unsubscribe);
			return unsubscribe;
		}),
		check: vi.fn(() => Promise.resolve(false))
	};

	// Cleanup function to call all unsubscribers
	const cleanup = () => {
		unsubscribers.forEach(fn => fn());
		unsubscribers.length = 0;
	};

	return { page, navigating, updated, cleanup };
}

/**
 * Creates a test harness for route components with navigation functions
 * @returns Mock navigation functions for testing
 */
export function createRouteTestHarness() {
	return {
		goto: vi.fn(),
		invalidate: vi.fn(),
		invalidateAll: vi.fn(),
		preloadCode: vi.fn(),
		preloadData: vi.fn(),
		beforeNavigate: vi.fn(),
		afterNavigate: vi.fn()
	};
}
