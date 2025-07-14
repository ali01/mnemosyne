import { vi } from 'vitest';

// Mock graph implementation
export const createMockGraph = () => ({
	addNode: vi.fn(),
	addEdge: vi.fn(),
	hasNode: vi.fn(() => true),
	getNodeAttributes: vi.fn(() => ({ x: 100, y: 100, color: '#3a7bd5' })),
	setNodeAttribute: vi.fn(),
	forEachNode: vi.fn(),
	forEachEdge: vi.fn(),
	nodes: vi.fn(() => []),
	edges: vi.fn(() => []),
	order: 0,
	size: 0
});

// Mock camera implementation
export const createMockCamera = () => ({
	animatedZoom: vi.fn(),
	animatedUnzoom: vi.fn(),
	animatedReset: vi.fn(),
	getState: vi.fn(() => ({ ratio: 1, x: 0, y: 0 })),
	setState: vi.fn()
});

// Mock mouse captor
export const createMockMouseCaptor = () => ({
	on: vi.fn()
});

// Mock Sigma instance
export const createMockSigma = () => {
	const mockGraph = createMockGraph();
	const mockCamera = createMockCamera();
	const mockMouseCaptor = createMockMouseCaptor();

	return {
		getGraph: vi.fn(() => mockGraph),
		getCamera: vi.fn(() => mockCamera),
		getMouseCaptor: vi.fn(() => mockMouseCaptor),
		on: vi.fn(),
		off: vi.fn(),
		kill: vi.fn(),
		refresh: vi.fn(),
		render: vi.fn(),
		viewportToGraph: vi.fn((coords: { x: number; y: number }) => coords),
		graphToViewport: vi.fn((coords: { x: number; y: number }) => coords)
	};
};

// Mock Sigma constructor
export const mockSigmaConstructor = vi.fn().mockImplementation(() => createMockSigma());

// Mock the entire sigma module
vi.mock('sigma', () => ({
	default: mockSigmaConstructor,
	Sigma: mockSigmaConstructor
}));

// Mock graphology
vi.mock('graphology', () => ({
	default: vi.fn().mockImplementation(() => createMockGraph())
}));
