import { http, HttpResponse } from 'msw';

// Sample test data
export const mockGraphData = {
	nodes: [
		{
			id: 'node1',
			title: 'Test Node 1',
			position: { x: 100, y: 100 },
			metadata: { type: 'core' }
		},
		{
			id: 'node2',
			title: 'Test Node 2',
			position: { x: 200, y: 200 },
			metadata: { type: 'sub' }
		}
	],
	edges: [
		{
			id: 'edge1',
			source: 'node1',
			target: 'node2',
			weight: 1
		}
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
	id: 'node1',
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

	// Search endpoint
	http.get('/api/v1/search', ({ request }) => {
		const url = new URL(request.url);
		const query = url.searchParams.get('q');

		if (!query || query.length < 2) {
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

		if (id === 'node1') {
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

	// Node position update endpoint
	http.put('/api/v1/nodes/:id/position', async ({ params, request }) => {
		const { id } = params;
		const position = await request.json();

		// Simulate success
		return new HttpResponse(null, { status: 200 });
	}),

	// Error endpoints for testing error handling
	http.get('/api/v1/graph/error', () => {
		return new HttpResponse(null, { status: 500 });
	}),

	http.get('/api/v1/search/error', () => {
		return new HttpResponse(null, { status: 500 });
	})
];
