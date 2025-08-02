import { http, HttpResponse } from 'msw';

// Test constants
export const TEST_CONSTANTS = {
	NODE1_ID: 'node1',
	NODE2_ID: 'node2',
	NODE1_POSITION: { x: 100, y: 100 },
	NODE2_POSITION: { x: 200, y: 200 },
	DEFAULT_WEIGHT: 1,
	MIN_SEARCH_LENGTH: 2
};

// Test data factory functions
export function createMockNode(overrides?: Partial<any>) {
	return {
		id: TEST_CONSTANTS.NODE1_ID,
		title: 'Test Node',
		position: TEST_CONSTANTS.NODE1_POSITION,
		metadata: { type: 'core' },
		...overrides
	};
}

export function createMockEdge(overrides?: Partial<any>) {
	return {
		id: 'edge1',
		source: TEST_CONSTANTS.NODE1_ID,
		target: TEST_CONSTANTS.NODE2_ID,
		weight: TEST_CONSTANTS.DEFAULT_WEIGHT,
		...overrides
	};
}

// Sample test data
export const mockGraphData = {
	nodes: [
		createMockNode({
			id: TEST_CONSTANTS.NODE1_ID,
			title: 'Test Node 1',
			position: TEST_CONSTANTS.NODE1_POSITION,
			metadata: { type: 'core' }
		}),
		createMockNode({
			id: TEST_CONSTANTS.NODE2_ID,
			title: 'Test Node 2',
			position: TEST_CONSTANTS.NODE2_POSITION,
			metadata: { type: 'sub' }
		})
	],
	edges: [
		createMockEdge()
	]
};

export const mockSearchResults = {
	nodes: [
		{
			id: 'search1',
			title: 'Search Result 1',
			metadata: { type: 'core' }
		},
		{
			id: 'search2',
			title: 'Search Result 2',
			metadata: { type: 'detail' }
		}
	]
};

export const mockNodeContent = {
	id: TEST_CONSTANTS.NODE1_ID,
	title: 'Test Node Content',
	content: '# Test Content\n\nThis is a test note with [[Test Link]] and regular content.'
};

// API request handlers
export const handlers = [
	// Graph data endpoint
	http.get('/api/v1/graph', ({ request }) => {
		const url = new URL(request.url);
		const level = url.searchParams.get('level');

		return HttpResponse.json(mockGraphData);
	}),

	// Search endpoint (corrected to match actual API)
	http.get('/api/v1/nodes/search', ({ request }) => {
		const url = new URL(request.url);
		const query = url.searchParams.get('q');

		if (!query || query.length < TEST_CONSTANTS.MIN_SEARCH_LENGTH) {
			return HttpResponse.json({ nodes: [] });
		}

		// Filter results based on query
		const filteredNodes = mockSearchResults.nodes.filter(node =>
			node.title.toLowerCase().includes(query.toLowerCase())
		);

		return HttpResponse.json({ nodes: filteredNodes });
	}),

	// Node content endpoint
	http.get('/api/v1/nodes/:id/content', ({ params }) => {
		const { id } = params;

		if (id === TEST_CONSTANTS.NODE1_ID) {
			return HttpResponse.json(mockNodeContent);
		}

		if (id === 'nonexistent') {
			return new HttpResponse(null, { status: 404 });
		}

		return HttpResponse.json({
			id,
			title: `Node ${id}`,
			content: `# Node ${id}\n\nContent for node ${id}`
		});
	}),

	// Node position update endpoint with validation
	http.put('/api/v1/nodes/:id/position', async ({ params, request }) => {
		const { id } = params;
		const position = await request.json() as any;

		// Validate position data
		if (!position || typeof position.x !== 'number' || typeof position.y !== 'number') {
			return new HttpResponse(
				JSON.stringify({ error: 'Invalid position data' }),
				{ status: 400, headers: { 'Content-Type': 'application/json' } }
			);
		}

		// Simulate success
		return new HttpResponse(null, { status: 200 });
	}),

	// Error endpoints for testing error handling
	http.get('/api/v1/graph/error', () => {
		return new HttpResponse(null, { status: 500 });
	}),

	http.get('/api/v1/nodes/search/error', () => {
		return new HttpResponse(null, { status: 500 });
	}),

	// Network error simulation
	http.get('/api/v1/graph/network-error', () => {
		return HttpResponse.error();
	}),

	// Timeout simulation
	http.get('/api/v1/graph/timeout', () => {
		return new Promise(() => {}); // Never resolves
	}),

	// Validation error
	http.get('/api/v1/graph/validation-error', () => {
		return new HttpResponse(
			JSON.stringify({ error: 'Invalid request parameters' }),
			{ status: 400, headers: { 'Content-Type': 'application/json' } }
		);
	})
];
