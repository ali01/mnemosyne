<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { graphStore } from '$lib/stores/graph';
	
	let container;
	let sigma = null;
	let Graph;
	let Sigma;
	
	onMount(async () => {
		// Import graph libraries only on client side
		const graphologyModule = await import('graphology');
		const sigmaModule = await import('sigma');
		Graph = graphologyModule.default;
		Sigma = sigmaModule.default;
		
		const graph = new Graph();
		
		// Load graph data from API
		try {
			const response = await fetch('/api/v1/graph?level=0');
			const data = await response.json();
			
			// Add nodes
			data.nodes.forEach((node) => {
				graph.addNode(node.id, {
					x: node.position.x,
					y: node.position.y,
					size: 10,
					label: node.title,
					color: getNodeColor(node.metadata?.type),
				});
			});
			
			// Add edges
			data.edges.forEach((edge) => {
				try {
					graph.addEdge(edge.source, edge.target, {
						weight: edge.weight,
						// Remove type for now - Sigma needs special configuration for edge types
					});
				} catch (e) {
					// Skip if nodes don't exist
				}
			});
		} catch (error) {
			console.error('Failed to load graph:', error);
		}
		
		const settings = {
			renderLabels: true,
			renderEdgeLabels: false,
			defaultNodeColor: '#3a7bd5',
			defaultEdgeColor: '#666',
			labelColor: { color: '#ffffff' },
			minZoomRatio: 0.1,
			maxZoomRatio: 10,
		};
		
		sigma = new Sigma(graph, container, settings);
		
		// Handle node clicks
		sigma.on('clickNode', ({ node }) => {
			graphStore.selectNode(node);
		});
		
		// Handle node dragging
		let draggedNode = null;
		let isDragging = false;
		
		sigma.on('downNode', (e) => {
			draggedNode = e.node;
			isDragging = true;
			sigma.getGraph().setNodeAttribute(draggedNode, 'highlighted', true);
		});
		
		sigma.getMouseCaptor().on('mousemovebody', (e) => {
			if (isDragging && draggedNode) {
				// Get the pointer position relative to the sigma container
				const pos = sigma.viewportToGraph(e);
				sigma.getGraph().setNodeAttribute(draggedNode, 'x', pos.x);
				sigma.getGraph().setNodeAttribute(draggedNode, 'y', pos.y);
				// Prevent the default camera movement
				e.preventSigmaDefault();
				e.original.preventDefault();
				e.original.stopPropagation();
			}
		});
		
		sigma.getMouseCaptor().on('mouseup', () => {
			if (draggedNode) {
				sigma.getGraph().setNodeAttribute(draggedNode, 'highlighted', false);
			}
			draggedNode = null;
			isDragging = false;
		});
		
		sigma.on('clickStage', () => {
			draggedNode = null;
			isDragging = false;
		});
	});
	
	function getNodeColor(type) {
		switch (type) {
			case 'core':
				return '#e74c3c';
			case 'sub':
				return '#3498db';
			case 'detail':
				return '#2ecc71';
			default:
				return '#3a7bd5';
		}
	}
	
	onDestroy(() => {
		if (sigma) {
			sigma.kill();
		}
	});
	
	function handleZoomIn() {
		if (sigma) {
			const camera = sigma.getCamera();
			camera.animatedZoom({ duration: 300 });
		}
	}
	
	function handleZoomOut() {
		if (sigma) {
			const camera = sigma.getCamera();
			camera.animatedUnzoom({ duration: 300 });
		}
	}
	
	function handleReset() {
		if (sigma) {
			const camera = sigma.getCamera();
			camera.animatedReset({ duration: 300 });
		}
	}
</script>

<div class="graph-container">
	<div class="graph-canvas" bind:this={container}></div>
	
	<div class="controls">
		<button on:click={handleZoomIn} title="Zoom In">+</button>
		<button on:click={handleZoomOut} title="Zoom Out">-</button>
		<button on:click={handleReset} title="Reset View">⟲</button>
	</div>
</div>

<style>
	.graph-container {
		width: 100%;
		height: 100%;
		position: relative;
	}
	
	.graph-canvas {
		width: 100%;
		height: 100%;
	}
	
	.controls {
		position: absolute;
		top: 20px;
		right: 20px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		background: var(--color-surface);
		padding: 10px;
		border-radius: 8px;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
	}
	
	button {
		width: 40px;
		height: 40px;
		border: none;
		background: var(--color-primary);
		color: white;
		border-radius: 4px;
		cursor: pointer;
		font-size: 18px;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: opacity 0.2s;
	}
	
	button:hover {
		opacity: 0.8;
	}
</style>