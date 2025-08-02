/// <reference types="vitest/globals" />
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
		url: new URL('http://localhost'),
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
globalThis.fetch = vi.fn();

// Mock IntersectionObserver with complete interface
function createMockIntersectionObserver(): IntersectionObserver {
	return {
		observe: vi.fn(),
		disconnect: vi.fn(),
		unobserve: vi.fn(),
		root: null,
		rootMargin: '',
		thresholds: [],
		takeRecords: vi.fn(() => [])
	} as unknown as IntersectionObserver;
}

globalThis.IntersectionObserver = vi.fn(createMockIntersectionObserver) as any;

// Mock ResizeObserver with complete interface
function createMockResizeObserver(): ResizeObserver {
	return {
		observe: vi.fn(),
		disconnect: vi.fn(),
		unobserve: vi.fn()
	} as unknown as ResizeObserver;
}

globalThis.ResizeObserver = vi.fn(createMockResizeObserver) as any;

// Mock HTMLCanvasElement methods for Sigma.js
function create2DContext(): CanvasRenderingContext2D {
	return {
		fillRect: vi.fn(),
		clearRect: vi.fn(),
		getImageData: vi.fn(() => ({ data: new Array(4), width: 1, height: 1, colorSpace: 'srgb' })),
		putImageData: vi.fn(),
		createImageData: vi.fn(() => ({ data: new Array(4), width: 1, height: 1, colorSpace: 'srgb' })),
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
		measureText: vi.fn(() => ({ width: 0, actualBoundingBoxAscent: 0, actualBoundingBoxDescent: 0, actualBoundingBoxLeft: 0, actualBoundingBoxRight: 0, fontBoundingBoxAscent: 0, fontBoundingBoxDescent: 0 })),
		transform: vi.fn(),
		rect: vi.fn(),
		clip: vi.fn(),
		canvas: {} as HTMLCanvasElement
	} as unknown as CanvasRenderingContext2D;
}

function createWebGLContext(): WebGLRenderingContext {
	return {
		clear: vi.fn(),
		clearColor: vi.fn(),
		enable: vi.fn(),
		disable: vi.fn(),
		viewport: vi.fn(),
		getExtension: vi.fn(),
		getParameter: vi.fn(),
		createShader: vi.fn(),
		shaderSource: vi.fn(),
		compileShader: vi.fn(),
		getShaderParameter: vi.fn(() => true),
		createProgram: vi.fn(() => ({})),
		attachShader: vi.fn(),
		linkProgram: vi.fn(),
		getProgramParameter: vi.fn(() => true),
		useProgram: vi.fn(),
		createBuffer: vi.fn(() => ({})),
		bindBuffer: vi.fn(),
		bufferData: vi.fn(),
		getAttribLocation: vi.fn(() => 0),
		enableVertexAttribArray: vi.fn(),
		vertexAttribPointer: vi.fn(),
		drawArrays: vi.fn(),
		drawElements: vi.fn(),
		canvas: {} as HTMLCanvasElement
	} as unknown as WebGLRenderingContext;
}

HTMLCanvasElement.prototype.getContext = vi.fn((contextId: string, options?: any) => {
	if (contextId === '2d') {
		return create2DContext();
	} else if (contextId === 'webgl' || contextId === 'webgl2' || contextId === 'experimental-webgl') {
		return createWebGLContext();
	}
	return null;
}) as any;

// Setup MSW server
export const server = setupServer(...handlers);

// Setup environment variables for tests
vi.stubEnv('PUBLIC_API_URL', 'http://localhost:8080');

// Setup global test utilities
beforeAll(async () => {
	await server.listen();
});

beforeEach(() => {
	// Reset all mocks before each test
	vi.clearAllMocks();
	// Reset environment variables
	vi.unstubAllEnvs();
	vi.stubEnv('PUBLIC_API_URL', 'http://localhost:8080');
});

afterEach(() => {
	server.resetHandlers();
});

afterAll(() => {
	server.close();
});
