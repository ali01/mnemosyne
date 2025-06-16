<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Graph from 'graphology';
	import Sigma from 'sigma';
	import type { Settings } from 'sigma/settings';
	import { graphStore } from '$lib/stores/graph';
	
	let container: HTMLDivElement;
	let sigma: Sigma | null = null;
	
	onMount(async () => {
		const graph = new Graph();
		
		// TODO: Load graph data from API
		// For now, add some sample nodes
		graph.addNode('1', { x: 0, y: 0, size: 10, label: 'Sample Node 1', color: '#3a7bd5' });
		graph.addNode('2', { x: 100, y: 100, size: 10, label: 'Sample Node 2', color: '#3a7bd5' });
		graph.addEdge('1', '2');
		
		const settings: Partial<Settings> = {
			renderLabels: true,
			renderEdgeLabels: false,
			defaultNodeColor: '#3a7bd5',
			defaultEdgeColor: '#666',
			minZoomRatio: 0.1,
			maxZoomRatio: 10,
		};
		
		sigma = new Sigma(graph, container, settings);
		
		// Handle node clicks
		sigma.on('clickNode', ({ node }) => {
			console.log('Clicked node:', node);
			// TODO: Open node content viewer
		});
		
		// Handle viewport changes for dynamic loading
		sigma.getCamera().on('updated', () => {
			const viewport = sigma!.getCamera().getViewport();
			// TODO: Load nodes within viewport
		});
	});
	
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