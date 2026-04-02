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
			file_path: 'memex/search-result-1.md',
			metadata: { type: 'core' }
		},
		{
			id: 'search2',
			title: 'Search Result 2',
			file_path: 'memex/search-result-2.md',
			metadata: { type: 'detail' }
		}
	]
};

// API request handlers
export const handlers = [
	// Graph data endpoint
	http.get('/api/v1/graphs/:id', () => {
		return HttpResponse.json(mockGraphData);
	}),

	// Search endpoint
	http.get('/api/v1/graphs/:id/search', ({ request }) => {
		const url = new URL(request.url);
		const query = url.searchParams.get('q');

		if (!query || query.length < TEST_CONSTANTS.MIN_SEARCH_LENGTH) {
			return HttpResponse.json({ nodes: [] });
		}

		const filteredNodes = mockSearchResults.nodes.filter(node =>
			node.title.toLowerCase().includes(query.toLowerCase())
		);

		return HttpResponse.json({ nodes: filteredNodes });
	}),

	// Batch position update endpoint
	http.put('/api/v1/graphs/:id/positions', async ({ request }) => {
		const positions = await request.json() as any;

		if (!Array.isArray(positions) || positions.length === 0) {
			return new HttpResponse(
				JSON.stringify({ error: 'No positions provided' }),
				{ status: 400, headers: { 'Content-Type': 'application/json' } }
			);
		}

		return HttpResponse.json({ message: 'Positions updated', count: positions.length });
	}),

	// Single position update endpoint
	http.put('/api/v1/graphs/:id/positions/:nodeId', async ({ request }) => {
		const position = await request.json() as any;

		if (!position || typeof position.x !== 'number' || typeof position.y !== 'number') {
			return new HttpResponse(
				JSON.stringify({ error: 'Invalid position data' }),
				{ status: 400, headers: { 'Content-Type': 'application/json' } }
			);
		}

		return HttpResponse.json({ message: 'Position updated' });
	}),

	// Graph list endpoint
	http.get('/api/v1/graphs', () => {
		return HttpResponse.json({ graphs: [] });
	}),
];
