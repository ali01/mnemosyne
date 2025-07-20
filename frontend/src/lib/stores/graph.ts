import { writable } from 'svelte/store';
import type { Node, Edge } from '$lib/types/graph';
import { toast } from './toast';

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
			const response = await fetch(`/api/v1/graph/viewport?minX=${viewport.minX}&maxX=${viewport.maxX}&minY=${viewport.minY}&maxY=${viewport.maxY}&level=${level}`);
			const data = await response.json();
			
			update(state => {
				data.nodes.forEach((node: Node) => {
					state.nodes.set(node.id, node);
				});
				return state;
			});
		},
		updateNodePosition: async (nodeId: string, position: { x: number; y: number }) => {
			// Mark node as saving
			update(state => {
				state.savingNodes.add(nodeId);
				return state;
			});
			
			try {
				const response = await fetch(`/api/v1/nodes/${nodeId}/position`, {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(position)
				});
				
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
				
				// Show success feedback briefly
				toast.success('Position saved', 2000);
			} catch (error) {
				// Remove saving state on error
				update(state => {
					state.savingNodes.delete(nodeId);
					return state;
				});
				toast.error('Failed to save node position. Please try again.');
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

// Helper function for retrying API calls
export async function fetchWithRetry(
	url: string,
	options: RequestInit = {},
	maxRetries = 3,
	delay = 1000
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