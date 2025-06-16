import { writable } from 'svelte/store';
import type { Node, Edge } from '$lib/types/graph';

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
		}
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
			await fetch(`/api/v1/nodes/${nodeId}/position`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(position)
			});
			
			update(state => {
				const node = state.nodes.get(nodeId);
				if (node) {
					node.position = position;
				}
				return state;
			});
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