import '@testing-library/jest-dom';
import { vi } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from './mocks/api';

// Mock SvelteKit modules
vi.mock('$app/environment', () => ({
	browser: false,
	dev: true,
	building: false,
	version: 'test'
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	invalidate: vi.fn(),
	invalidateAll: vi.fn(),
	preloadData: vi.fn(),
	preloadCode: vi.fn(),
	onNavigate: vi.fn(),
	beforeNavigate: vi.fn(),
	afterNavigate: vi.fn()
}));

vi.mock('$app/stores', () => {
	const mockPage = {
		params: {},
		route: { id: null },
		url: new URL('http://localhost:3000'),
		data: {},
		status: 200,
		error: null,
		form: null
	};

	return {
		page: {
			subscribe: (callback: (value: typeof mockPage) => void) => {
				callback(mockPage);
				return () => {};
			}
		},
		navigating: {
			subscribe: (callback: (value: null) => void) => {
				callback(null);
				return () => {};
			}
		},
		updated: {
			subscribe: (callback: (value: boolean) => void) => {
				callback(false);
				return () => {};
			}
		}
	};
});

// Mock fetch globally
global.fetch = vi.fn();

// Mock IntersectionObserver
global.IntersectionObserver = vi.fn(() => ({
	observe: vi.fn(),
	disconnect: vi.fn(),
	unobserve: vi.fn()
}));

// Mock ResizeObserver
global.ResizeObserver = vi.fn(() => ({
	observe: vi.fn(),
	disconnect: vi.fn(),
	unobserve: vi.fn()
}));

// Mock HTMLCanvasElement methods for Sigma.js
HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
	fillRect: vi.fn(),
	clearRect: vi.fn(),
	getImageData: vi.fn(() => ({ data: new Array(4) })),
	putImageData: vi.fn(),
	createImageData: vi.fn(() => ({ data: new Array(4) })),
	setTransform: vi.fn(),
	drawImage: vi.fn(),
	save: vi.fn(),
	fillText: vi.fn(),
	restore: vi.fn(),
	beginPath: vi.fn(),
	moveTo: vi.fn(),
	lineTo: vi.fn(),
	closePath: vi.fn(),
	stroke: vi.fn(),
	translate: vi.fn(),
	scale: vi.fn(),
	rotate: vi.fn(),
	arc: vi.fn(),
	fill: vi.fn(),
	measureText: vi.fn(() => ({ width: 0 })),
	transform: vi.fn(),
	rect: vi.fn(),
	clip: vi.fn()
}));

// Setup MSW server
export const server = setupServer(...handlers);

// Setup global test utilities
beforeAll(() => {
	server.listen();
});

beforeEach(() => {
	// Reset all mocks before each test
	vi.clearAllMocks();
});

afterEach(() => {
	server.resetHandlers();
});

afterAll(() => {
	server.close();
});
