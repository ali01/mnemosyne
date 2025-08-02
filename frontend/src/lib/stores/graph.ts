import { writable } from 'svelte/store';
import type { Node, Edge } from '$lib/types/graph';
import { toast } from './toast';

// Constants
const DEFAULT_MAX_RETRIES = 3;
const DEFAULT_RETRY_DELAY_MS = 1000;
const SUCCESS_TOAST_DURATION_MS = 3000;
const NODE_ID_PATTERN = /^[a-zA-Z0-9_-]+$/;

interface GraphState {
	nodes: Map<string, Node>;
	edges: Map<string, Edge>;
	selectedNode: string | null;
	zoomLevel: number;
	viewport: {
		minX: number;
		maxX: number;
		minY: number;
		maxY: number;
	};
	savingNodes: Set<string>; // Track which nodes are being saved
}

function createGraphStore() {
	const { subscribe, set, update } = writable<GraphState>({
		nodes: new Map(),
		edges: new Map(),
		selectedNode: null,
		zoomLevel: 1,
		viewport: {
			minX: -1000,
			maxX: 1000,
			minY: -1000,
			maxY: 1000
		},
		savingNodes: new Set()
	});

	return {
		subscribe,
		loadViewport: async (viewport: GraphState['viewport'], level: number) => {
			try {
				const response = await fetchWithRetry(
					`/api/v1/graph/viewport?minX=${viewport.minX}&maxX=${viewport.maxX}&minY=${viewport.minY}&maxY=${viewport.maxY}&level=${level}`
				);

				if (!response.ok) {
					throw new Error(`Failed to load viewport: ${response.statusText}`);
				}

				const data = await response.json();

				update(state => {
					data.nodes.forEach((node: Node) => {
						state.nodes.set(node.id, node);
					});
					// Also update edges if provided
					if (data.edges) {
						data.edges.forEach((edge: Edge) => {
							state.edges.set(edge.id, edge);
						});
					}
					return state;
				});
			} catch (error) {
				toast.error('Failed to load graph data. Please refresh the page.');
				console.error('Failed to load viewport:', error);
				throw error;
			}
		},
		updateNodePosition: async (nodeId: string, position: { x: number; y: number }) => {
			// Validate node ID
			if (!NODE_ID_PATTERN.test(nodeId)) {
				toast.error('Invalid node ID');
				console.error('Invalid node ID:', nodeId);
				return;
			}

			// Validate position values
			if (!Number.isFinite(position.x) || !Number.isFinite(position.y)) {
				toast.error('Invalid position values');
				console.error('Invalid position values:', position);
				return;
			}

			// Check if node exists before attempting update
			let nodeExists = false;
			update(state => {
				nodeExists = state.nodes.has(nodeId);
				return state;
			});

			if (!nodeExists) {
				toast.error('Node not found');
				console.error('Node not found:', nodeId);
				return;
			}

			// Mark node as saving
			update(state => {
				state.savingNodes.add(nodeId);
				return state;
			});

			try {
				const response = await fetchWithRetry(
					`/api/v1/nodes/${encodeURIComponent(nodeId)}/position`,
					{
						method: 'PUT',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(position)
					}
				);

				if (!response.ok) {
					throw new Error(`Failed to save position: ${response.statusText}`);
				}

				update(state => {
					const node = state.nodes.get(nodeId);
					if (node) {
						node.position = position;
					}
					state.savingNodes.delete(nodeId);
					return state;
				});

				// Show success feedback with appropriate duration
				toast.success('Position saved', SUCCESS_TOAST_DURATION_MS);
			} catch (error) {
				// Remove saving state on error
				update(state => {
					state.savingNodes.delete(nodeId);
					return state;
				});
				// Provide user-friendly error message based on error type
				let errorMessage = 'Failed to save node position. Please try again.';
				if (error instanceof Error) {
					if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
						errorMessage = 'Network error. Please check your connection and try again.';
					} else if (error.message.includes('500') || error.message.includes('Server')) {
						errorMessage = 'Server error. Please try again later.';
					}
				}
				toast.error(errorMessage);
				console.error('Failed to update node position:', error);
			}
		},
		selectNode: (nodeId: string | null) => {
			update(state => {
				state.selectedNode = nodeId;
				return state;
			});
		}
	};
}

export const graphStore = createGraphStore();

/**
 * Performs a fetch request with automatic retry logic for server errors.
 *
 * @param url - The URL to fetch
 * @param options - Fetch options (method, headers, body, etc.)
 * @param maxRetries - Maximum number of retry attempts (default: 3)
 * @param delay - Initial delay between retries in milliseconds (default: 1000)
 * @returns Promise resolving to the Response object
 * @throws Error if all retry attempts fail
 *
 * @example
 * const response = await fetchWithRetry('/api/data', {
 *   method: 'POST',
 *   body: JSON.stringify({ key: 'value' })
 * });
 */
export async function fetchWithRetry(
	url: string,
	options: RequestInit = {},
	maxRetries = DEFAULT_MAX_RETRIES,
	delay = DEFAULT_RETRY_DELAY_MS
): Promise<Response> {
	for (let i = 0; i < maxRetries; i++) {
		try {
			const response = await fetch(url, options);
			if (!response.ok && i < maxRetries - 1 && response.status >= 500) {
				// Retry on server errors
				await new Promise(resolve => setTimeout(resolve, delay * (i + 1)));
				continue;
			}
			return response;
		} catch (error) {
			if (i === maxRetries - 1) throw error;
			await new Promise(resolve => setTimeout(resolve, delay * (i + 1)));
		}
	}
	throw new Error('Failed to fetch after retries');
}
