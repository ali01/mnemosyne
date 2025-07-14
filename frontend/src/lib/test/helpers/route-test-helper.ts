import { vi } from 'vitest';

// Helper to create a minimal Svelte component mock
export function createComponentMock(name: string, renderOutput: string) {
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

// Helper to mock SvelteKit stores
export function mockSvelteKitStores() {
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
			return () => {};
		})
	};

	const navigating = {
		subscribe: vi.fn((callback) => {
			callback(null);
			return () => {};
		})
	};

	const updated = {
		subscribe: vi.fn((callback) => {
			callback(false);
			return () => {};
		}),
		check: vi.fn(() => Promise.resolve(false))
	};

	return { page, navigating, updated };
}

// Helper to create a test harness for route components
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
