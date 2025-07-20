import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { graphStore, fetchWithRetry } from './graph';
import { server } from '../test/setup';
import { http, HttpResponse } from 'msw';

// Mock the toast store
vi.mock('./toast', () => ({
	toast: {
		success: vi.fn(),
		error: vi.fn(),
		info: vi.fn(),
		warning: vi.fn()
	}
}));

describe('graph store', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		// Clear store state between tests
		const state = get(graphStore);
		state.nodes.clear();
		state.edges.clear();
		state.selectedNode = null;
		state.savingNodes.clear();
	});

	it('should have correct initial state', () => {
		const state = get(graphStore);

		expect(state.nodes).toBeInstanceOf(Map);
		expect(state.edges).toBeInstanceOf(Map);
		expect(state.selectedNode).toBeNull();
		expect(state.zoomLevel).toBe(1);
		expect(state.viewport).toEqual({
			minX: -1000,
			maxX: 1000,
			minY: -1000,
			maxY: 1000
		});
		expect(state.savingNodes).toBeInstanceOf(Set);
		expect(state.savingNodes.size).toBe(0);
	});

	describe('selectNode', () => {
		it('should set selected node', () => {
			graphStore.selectNode('node1');

			const state = get(graphStore);
			expect(state.selectedNode).toBe('node1');
		});

		it('should clear selected node when null is passed', () => {
			graphStore.selectNode('node1');
			graphStore.selectNode(null);

			const state = get(graphStore);
			expect(state.selectedNode).toBeNull();
		});
	});

	describe('updateNodePosition', () => {
		beforeEach(() => {
			// Reset MSW handlers
			server.resetHandlers();
		});

		it('should successfully update node position', async () => {
			// Mock successful API response
			server.use(
				http.put('/api/v1/nodes/node1/position', () => {
					return new HttpResponse(null, { status: 200 });
				})
			);

			const position = { x: 100, y: 200 };

			// Add a node to the store first
			const state = get(graphStore);
			state.nodes.set('node1', {
				id: 'node1',
				title: 'Test Node',
				position: { x: 0, y: 0 },
				level: 1,
				metadata: { type: 'core' }
			});

			await graphStore.updateNodePosition('node1', position);

			const updatedState = get(graphStore);
			expect(updatedState.savingNodes.has('node1')).toBe(false);

			const node = updatedState.nodes.get('node1');
			expect(node?.position).toEqual(position);
		});

		it('should track saving state during position update', async () => {
			// Mock slow API response
			server.use(
				http.put('/api/v1/nodes/node1/position', async () => {
					await new Promise(resolve => setTimeout(resolve, 100));
					return new HttpResponse(null, { status: 200 });
				})
			);

			const position = { x: 100, y: 200 };

			// Start the update
			const updatePromise = graphStore.updateNodePosition('node1', position);

			// Check that node is marked as saving
			const savingState = get(graphStore);
			expect(savingState.savingNodes.has('node1')).toBe(true);

			// Wait for completion
			await updatePromise;

			// Check that saving state is cleared
			const finalState = get(graphStore);
			expect(finalState.savingNodes.has('node1')).toBe(false);
		});

		it('should handle API errors gracefully', async () => {
			const { toast } = await import('./toast');

			// Mock API error
			server.use(
				http.put('/api/v1/nodes/node1/position', () => {
					return new HttpResponse(null, { status: 500 });
				})
			);

			const position = { x: 100, y: 200 };

			await graphStore.updateNodePosition('node1', position);

			// Check that saving state is cleared
			const state = get(graphStore);
			expect(state.savingNodes.has('node1')).toBe(false);

			// Check that error toast was shown
			expect(toast.error).toHaveBeenCalledWith('Failed to save node position. Please try again.');
		});

		it('should handle network errors', async () => {
			const { toast } = await import('./toast');

			// Mock network error
			server.use(
				http.put('/api/v1/nodes/node1/position', () => {
					return HttpResponse.error();
				})
			);

			const position = { x: 100, y: 200 };

			await graphStore.updateNodePosition('node1', position);

			// Check that saving state is cleared
			const state = get(graphStore);
			expect(state.savingNodes.has('node1')).toBe(false);

			// Check that error toast was shown
			expect(toast.error).toHaveBeenCalledWith('Failed to save node position. Please try again.');
		});

		it('should handle multiple concurrent position updates', async () => {
			// Mock successful API responses
			server.use(
				http.put('/api/v1/nodes/:id/position', () => {
					return new HttpResponse(null, { status: 200 });
				})
			);

			const updates = [
				{ nodeId: 'node1', position: { x: 100, y: 200 } },
				{ nodeId: 'node2', position: { x: 300, y: 400 } },
				{ nodeId: 'node3', position: { x: 500, y: 600 } }
			];

			// Start all updates concurrently
			const promises = updates.map(update =>
				graphStore.updateNodePosition(update.nodeId, update.position)
			);

			// Check that all nodes are marked as saving
			const savingState = get(graphStore);
			updates.forEach(update => {
				expect(savingState.savingNodes.has(update.nodeId)).toBe(true);
			});

			// Wait for all to complete
			await Promise.all(promises);

			// Check that all saving states are cleared
			const finalState = get(graphStore);
			updates.forEach(update => {
				expect(finalState.savingNodes.has(update.nodeId)).toBe(false);
			});
		});
	});

	describe('loadViewport', () => {
		beforeEach(() => {
			server.resetHandlers();
		});

		it('should load nodes from viewport API', async () => {
			const mockNodes = [
				{
					id: 'node1',
					title: 'Node 1',
					position: { x: 100, y: 100 },
					metadata: { type: 'core' }
				},
				{
					id: 'node2',
					title: 'Node 2',
					position: { x: 200, y: 200 },
					metadata: { type: 'sub' }
				}
			];

			// Mock viewport API response
			server.use(
				http.get('/api/v1/graph/viewport', () => {
					return HttpResponse.json({ nodes: mockNodes });
				})
			);

			const viewport = { minX: 0, maxX: 500, minY: 0, maxY: 500 };

			await graphStore.loadViewport(viewport, 1);

			const state = get(graphStore);
			expect(state.nodes.size).toBe(2);
			expect(state.nodes.get('node1')).toEqual(mockNodes[0]);
			expect(state.nodes.get('node2')).toEqual(mockNodes[1]);
		});

		it('should append new nodes to existing ones', async () => {
			// Add initial node
			const state = get(graphStore);
			state.nodes.set('existing', {
				id: 'existing',
				title: 'Existing Node',
				position: { x: 0, y: 0 },
				level: 1,
				metadata: { type: 'detail' }
			});

			const newNodes = [
				{
					id: 'new1',
					title: 'New Node 1',
					position: { x: 100, y: 100 },
					metadata: { type: 'core' }
				}
			];

			server.use(
				http.get('/api/v1/graph/viewport', () => {
					return HttpResponse.json({ nodes: newNodes });
				})
			);

			const viewport = { minX: 0, maxX: 500, minY: 0, maxY: 500 };

			await graphStore.loadViewport(viewport, 1);

			const updatedState = get(graphStore);
			expect(updatedState.nodes.size).toBe(2);
			expect(updatedState.nodes.has('existing')).toBe(true);
			expect(updatedState.nodes.has('new1')).toBe(true);
		});

		it('should update existing nodes with new data', async () => {
			// Add initial node
			const state = get(graphStore);
			state.nodes.set('node1', {
				id: 'node1',
				title: 'Old Title',
				position: { x: 0, y: 0 },
				level: 1,
				metadata: { type: 'detail' }
			});

			const updatedNodes = [
				{
					id: 'node1',
					title: 'New Title',
					position: { x: 100, y: 100 },
					metadata: { type: 'core' }
				}
			];

			server.use(
				http.get('/api/v1/graph/viewport', () => {
					return HttpResponse.json({ nodes: updatedNodes });
				})
			);

			const viewport = { minX: 0, maxX: 500, minY: 0, maxY: 500 };

			await graphStore.loadViewport(viewport, 1);

			const updatedState = get(graphStore);
			expect(updatedState.nodes.size).toBe(1);

			const node = updatedState.nodes.get('node1');
			expect(node?.title).toBe('New Title');
			expect(node?.position).toEqual({ x: 100, y: 100 });
			expect(node?.metadata?.type).toBe('core');
		});

		it('should include viewport parameters in API call', async () => {
			let capturedUrl = '';

			server.use(
				http.get('/api/v1/graph/viewport', ({ request }) => {
					capturedUrl = request.url;
					return HttpResponse.json({ nodes: [] });
				})
			);

			const viewport = { minX: -100, maxX: 300, minY: -200, maxY: 400 };

			await graphStore.loadViewport(viewport, 2);

			const url = new URL(capturedUrl);
			expect(url.searchParams.get('minX')).toBe('-100');
			expect(url.searchParams.get('maxX')).toBe('300');
			expect(url.searchParams.get('minY')).toBe('-200');
			expect(url.searchParams.get('maxY')).toBe('400');
			expect(url.searchParams.get('level')).toBe('2');
		});
	});
});

describe('fetchWithRetry utility', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		server.resetHandlers();
	});

	it('should return successful response on first try', async () => {
		server.use(
			http.get('/test-endpoint', () => {
				return HttpResponse.json({ success: true });
			})
		);

		const response = await fetchWithRetry('/test-endpoint');
		expect(response.ok).toBe(true);

		const data = await response.json();
		expect(data).toEqual({ success: true });
	});

	it('should retry on server errors (5xx)', async () => {
		let attemptCount = 0;

		server.use(
			http.get('/test-endpoint', () => {
				attemptCount++;
				if (attemptCount < 3) {
					return new HttpResponse(null, { status: 500 });
				}
				return HttpResponse.json({ success: true });
			})
		);

		const response = await fetchWithRetry('/test-endpoint', {}, 3, 10);
		expect(response.ok).toBe(true);
		expect(attemptCount).toBe(3);
	});

	it('should not retry on client errors (4xx)', async () => {
		let attemptCount = 0;

		server.use(
			http.get('/test-endpoint', () => {
				attemptCount++;
				return new HttpResponse(null, { status: 404 });
			})
		);

		const response = await fetchWithRetry('/test-endpoint', {}, 3);
		expect(response.status).toBe(404);
		expect(attemptCount).toBe(1);
	});

	it('should retry on network errors', async () => {
		let attemptCount = 0;

		server.use(
			http.get('/test-endpoint', () => {
				attemptCount++;
				if (attemptCount < 2) {
					return HttpResponse.error();
				}
				return HttpResponse.json({ success: true });
			})
		);

		const response = await fetchWithRetry('/test-endpoint', {}, 3, 10);
		expect(response.ok).toBe(true);
		expect(attemptCount).toBe(2);
	});

	it('should return failing response after max retries exceeded', async () => {
		let attemptCount = 0;

		server.use(
			http.get('/test-endpoint', () => {
				attemptCount++;
				return new HttpResponse(null, { status: 500 });
			})
		);

		const response = await fetchWithRetry('/test-endpoint', {}, 2, 10);
		expect(response.status).toBe(500);
		expect(attemptCount).toBe(2);
	});

	it('should use exponential backoff delay', async () => {
		const startTime = Date.now();
		let attempts = 0;

		server.use(
			http.get('/test-endpoint', () => {
				attempts++;
				return new HttpResponse(null, { status: 500 });
			})
		);

		try {
			await fetchWithRetry('/test-endpoint', {}, 3, 50);
		} catch (error) {
			// Expected to fail
		}

		const elapsed = Date.now() - startTime;
		// Should have waited: 50ms + 100ms = 150ms minimum
		// Allow some tolerance for test execution time
		expect(elapsed).toBeGreaterThan(100);
		expect(attempts).toBe(3);
	});

	it('should pass through request options', async () => {
		let capturedMethod: string | undefined;
		let capturedContentType: string | null | undefined;

		server.use(
			http.post('/test-endpoint', ({ request }) => {
				capturedMethod = request.method;
				capturedContentType = request.headers.get('Content-Type');
				return HttpResponse.json({ success: true });
			})
		);

		const options = {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ test: 'data' })
		};

		await fetchWithRetry('/test-endpoint', options);

		expect(capturedMethod).toBe('POST');
		expect(capturedContentType).toBe('application/json');
	});
});
