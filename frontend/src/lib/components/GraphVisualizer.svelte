<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { graphStore, fetchWithRetry } from "$lib/stores/graph";
    import { debounce } from "$lib/utils/debounce";
    import { goto } from "$app/navigation";
    import { toast } from "$lib/stores/toast";
    import LoadingSpinner from "./LoadingSpinner.svelte";
    import type { Sigma as SigmaType } from "sigma";
    import type Graph from "graphology";

    let container: HTMLDivElement;
    let sigma: SigmaType;
    let GraphConstructor: typeof Graph;
    let SigmaConstructor: typeof SigmaType;
    let savingNodes = new Set<string>();
    let loading = true;
    let error = "";
    let unsubscribe: (() => void) | null = null;

    const debouncedUpdatePosition = debounce(
        (nodeId: string, position: { x: number; y: number }) => {
            graphStore.updateNodePosition(nodeId, position);
        },
        300, // 300ms debounce
    );

    onMount(async () => {
        const graphologyModule = await import("graphology");
        const sigmaModule = await import("sigma");
        GraphConstructor = graphologyModule.default;
        SigmaConstructor = sigmaModule.default;

        const graph = new GraphConstructor();

        // Load graph data from API
        try {
            const response = await fetchWithRetry("/api/v1/graph?level=0");

            if (!response.ok) {
                throw new Error(`Failed to load graph: ${response.statusText}`);
            }

            const data = await response.json();

            // Add nodes
            data.nodes.forEach((node: any) => {
                graph.addNode(node.id, {
                    x: node.position.x,
                    y: node.position.y,
                    size: 10,
                    label: node.title,
                    color: getNodeColor(node.metadata?.type),
                    metadata: node.metadata, // Store metadata for later use
                });
            });

            // Add edges
            data.edges.forEach((edge: any) => {
                try {
                    graph.addEdge(edge.source, edge.target, {
                        weight: edge.weight,
                        // Remove type for now - Sigma needs special configuration for edge types
                    });
                } catch (e) {
                    // Skip if nodes don't exist
                }
            });

            loading = false;
        } catch (err) {
            console.error("Failed to load graph:", err);
            error =
                err instanceof Error
                    ? err.message
                    : "Failed to load graph data";
            toast.error(
                "Failed to load graph. Please refresh the page to try again.",
            );
            loading = false;
            return;
        }

        const settings = {
            renderLabels: true,
            renderEdgeLabels: false,
            defaultNodeColor: "#3a7bd5",
            defaultEdgeColor: "#666",
            labelColor: { color: "#ffffff" },
            minZoomRatio: 0.1,
            maxZoomRatio: 10,
        };

        sigma = new SigmaConstructor(graph, container, settings);

        // Subscribe to graph store for saving state after sigma is initialized
        unsubscribe = graphStore.subscribe((state) => {
            savingNodes = state.savingNodes;

            // Update node visual state based on saving status
            // Check that sigma still exists and has a graph
            if (sigma && sigma.getGraph) {
                const graph = sigma.getGraph();
                if (graph) {
                    savingNodes.forEach((nodeId) => {
                        if (graph.hasNode(nodeId)) {
                            graph.setNodeAttribute(nodeId, "color", "#ffa500"); // Orange for saving
                        }
                    });

                    graph.forEachNode((node, attributes) => {
                        if (!savingNodes.has(node) && attributes.color === "#ffa500") {
                            graph.setNodeAttribute(
                                node,
                                "color",
                                getNodeColor(attributes.metadata?.type),
                            );
                        }
                    });
                }
            }
        });

        // Handle node clicks
        sigma.on("clickNode", ({ node }) => {
            // Validate and encode the node ID to prevent path traversal attacks
            // Node IDs should be alphanumeric with hyphens/underscores
            if (/^[a-zA-Z0-9_-]+$/.test(node)) {
                goto(`/notes/${encodeURIComponent(node)}`);
            } else {
                console.error("Invalid node ID:", node);
                toast.error("Unable to navigate to this node");
            }
        });

        // Handle node dragging
        let draggedNode: string | null = null;
        let isDragging = false;

        sigma.on("downNode", (e) => {
            draggedNode = e.node;
            isDragging = true;
            if (sigma) {
                sigma
                    .getGraph()
                    .setNodeAttribute(draggedNode, "highlighted", true);
            }
        });

        sigma.getMouseCaptor().on("mousemovebody", (e: any) => {
            if (isDragging && draggedNode && sigma) {
                // Get the pointer position relative to the sigma container
                const pos = sigma.viewportToGraph(e);
                sigma.getGraph().setNodeAttribute(draggedNode, "x", pos.x);
                sigma.getGraph().setNodeAttribute(draggedNode, "y", pos.y);
                // Prevent the default camera movement
                e.preventSigmaDefault();
                e.original.preventDefault();
                e.original.stopPropagation();
            }
        });

        sigma.getMouseCaptor().on("mouseup", () => {
            if (draggedNode && sigma) {
                const graph = sigma.getGraph();
                // Check if the node still exists before accessing its attributes
                if (graph.hasNode(draggedNode)) {
                    graph.setNodeAttribute(draggedNode, "highlighted", false);

                    // Save the new position to the backend (debounced)
                    const nodeData = graph.getNodeAttributes(draggedNode);
                    if (nodeData.x !== undefined && nodeData.y !== undefined) {
                        debouncedUpdatePosition(draggedNode, {
                            x: nodeData.x,
                            y: nodeData.y,
                        });
                    }
                }
            }
            draggedNode = null;
            isDragging = false;
        });

        sigma.on("clickStage", () => {
            draggedNode = null;
            isDragging = false;
        });
    });

    function getNodeColor(type: string) {
        switch (type) {
            case "core":
                return "#e74c3c";
            case "sub":
                return "#3498db";
            case "detail":
                return "#2ecc71";
            default:
                return "#3a7bd5";
        }
    }

    onDestroy(() => {
        if (sigma) {
            sigma.kill();
        }
        if (unsubscribe) {
            unsubscribe();
        }
    });

    function handleZoomIn() {
        if (!sigma) return;
        const camera = sigma.getCamera();
        camera.animatedZoom({ duration: 300 });
    }

    function handleZoomOut() {
        if (!sigma) return;
        const camera = sigma.getCamera();
        camera.animatedUnzoom({ duration: 300 });
    }

    function handleReset() {
        if (!sigma) return;
        const camera = sigma.getCamera();
        camera.animatedReset({ duration: 300 });
    }
</script>

<div class="graph-container">
    {#if loading}
        <div class="loading-container">
            <LoadingSpinner size="large" message="Loading graph data..." />
        </div>
    {:else if error}
        <div class="error-container">
            <p class="error-message">{error}</p>
            <button
                class="retry-button"
                on:click={() => window.location.reload()}
            >
                Reload Page
            </button>
        </div>
    {:else}
        <div class="graph-canvas" bind:this={container}></div>

        <div class="controls">
            <button on:click={handleZoomIn} title="Zoom In">+</button>
            <button on:click={handleZoomOut} title="Zoom Out">-</button>
            <button on:click={handleReset} title="Reset View">⟲</button>
        </div>
    {/if}
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

    .loading-container,
    .error-container {
        width: 100%;
        height: 100%;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 1.5rem;
    }

    .error-message {
        color: #e74c3c;
        font-size: 1.2rem;
        text-align: center;
        max-width: 400px;
    }

    .retry-button {
        background-color: var(--color-primary);
        color: white;
        border: none;
        padding: 0.75rem 2rem;
        border-radius: 4px;
        cursor: pointer;
        font-size: 1rem;
        transition: opacity 0.2s;
        width: auto;
        height: auto;
    }

    .retry-button:hover {
        opacity: 0.8;
    }
</style>
