<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { navigate } from "$lib/router";
    import { toast } from "$lib/stores/toast";
    import LoadingSpinner from "./LoadingSpinner.svelte";
    import type { Sigma as SigmaType } from "sigma";
    import type Graph from "graphology";

    export let graphId: string = '';

    let container: HTMLDivElement;
    let sigma: SigmaType;
    let GraphConstructor: typeof Graph;
    let SigmaConstructor: typeof SigmaType;
    let loading = true;
    let error = "";
    let nodeCount = 0;
    let edgeCount = 0;
    let hoveredNode: string | null = null;

    function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
        const words = text.split(" ");
        const lines: string[] = [];
        let line = "";
        for (const word of words) {
            const testLine = line ? line + " " + word : word;
            if (ctx.measureText(testLine).width > maxWidth && line) {
                lines.push(line);
                line = word;
            } else {
                line = testLine;
            }
        }
        lines.push(line);
        return lines;
    }

    function saveAllPositions(graph: any) {
        const positions: { node_id: string; x: number; y: number }[] = [];
        graph.forEachNode((node: string, attrs: any) => {
            positions.push({ node_id: node, x: attrs.x, y: attrs.y });
        });
        const posUrl = graphId ? `/api/v1/graphs/${graphId}/positions` : '/api/v1/graphs/0/positions';
        fetch(posUrl, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(positions),
        }).catch((err) => console.error("Failed to save positions:", err));
    }

    onMount(async () => {
        const [graphologyModule, sigmaModule, forceAtlas2Module, louvainModule, noverlapModule] =
            await Promise.all([
                import("graphology"),
                import("sigma"),
                import("graphology-layout-forceatlas2"),
                import("graphology-communities-louvain"),
                import("graphology-layout-noverlap"),
            ]);
        GraphConstructor = graphologyModule.default;
        SigmaConstructor = sigmaModule.default;
        const forceAtlas2 = forceAtlas2Module.default;
        const louvain = louvainModule.default;
        const noverlap = noverlapModule.default;

        const graph = new GraphConstructor({ type: "undirected" });

        try {
            const graphUrl = graphId ? `/api/v1/graphs/${graphId}` : '/api/v1/graphs/0';
            const response = await fetch(graphUrl);

            if (!response.ok) {
                throw new Error(`Failed to load graph: ${response.statusText}`);
            }

            const data = await response.json();

            data.nodes.forEach((node: any) => {
                const hasPosition = node.position.x !== 0 || node.position.y !== 0;
                graph.addNode(node.id, {
                    x: hasPosition ? node.position.x : (Math.random() - 0.5) * 200,
                    y: hasPosition ? node.position.y : (Math.random() - 0.5) * 200,
                    size: 3,
                    label: node.title,
                    color: "#7b8cff",
                    metadata: node.metadata,
                });
            });

            data.edges.forEach((edge: any) => {
                try {
                    graph.addEdge(edge.source, edge.target, {
                        weight: edge.weight,
                    });
                } catch (e) {
                    // Skip duplicate edges or missing nodes
                }
            });

            // Remove orphan nodes (no connections)
            const orphans: string[] = [];
            graph.forEachNode((node) => {
                if (graph.degree(node) === 0) orphans.push(node);
            });
            orphans.forEach((node) => graph.dropNode(node));

            // Detect communities with Louvain and assign colors
            const communityPalette = [
                "#7b8cff", "#ff6b6b", "#6bffb8", "#ffb86b", "#6bc5ff",
                "#d46bff", "#ff6bb5", "#6bfff0", "#c8ff6b", "#ff916b",
                "#8b6bff", "#6bff8b", "#ff6b6b", "#6bd4ff", "#ffe06b",
                "#a06bff", "#ff6be0", "#6bffe0", "#ffaa6b", "#6b9fff",
            ];
            louvain.assign(graph, { resolution: 1.0 });
            const communityColors = new Map<string, string>();
            let colorIdx = 0;
            graph.forEachNode((node, attrs) => {
                const community = String(attrs.community);
                if (!communityColors.has(community)) {
                    communityColors.set(community, communityPalette[colorIdx % communityPalette.length]);
                    colorIdx++;
                }
                graph.setNodeAttribute(node, "color", communityColors.get(community));
                graph.setNodeAttribute(node, "communityColor", communityColors.get(community));
            });

            // Run layout if no saved positions
            const needsLayout = data.nodes.every(
                (n: any) => n.position.x === 0 && n.position.y === 0,
            );
            if (needsLayout) {
                // Two-level layout:
                // 1. Build a meta-graph of communities and lay it out
                // 2. Position each node near its community centroid
                // 3. Run ForceAtlas2 locally within the seeded positions

                // Group nodes by community
                const communities = new Map<string, string[]>();
                graph.forEachNode((node, attrs) => {
                    const c = String(attrs.community);
                    if (!communities.has(c)) communities.set(c, []);
                    communities.get(c)!.push(node);
                });

                // Build a meta-graph: one node per community, edges weighted by inter-community connections
                const metaGraph = new GraphConstructor();
                for (const [cid, members] of communities) {
                    metaGraph.addNode(cid, {
                        x: (Math.random() - 0.5) * 100,
                        y: (Math.random() - 0.5) * 100,
                        size: Math.sqrt(members.length),
                    });
                }
                graph.forEachEdge((_edge, _attrs, source, target) => {
                    const sc = String(graph.getNodeAttribute(source, "community"));
                    const tc = String(graph.getNodeAttribute(target, "community"));
                    if (sc !== tc) {
                        if (!metaGraph.hasEdge(sc, tc) && !metaGraph.hasEdge(tc, sc)) {
                            metaGraph.addEdge(sc, tc, { weight: 1 });
                        } else {
                            const edgeKey = metaGraph.hasEdge(sc, tc)
                                ? metaGraph.edge(sc, tc)
                                : metaGraph.edge(tc, sc);
                            if (edgeKey) {
                                metaGraph.setEdgeAttribute(edgeKey, "weight",
                                    metaGraph.getEdgeAttribute(edgeKey, "weight") + 1);
                            }
                        }
                    }
                });

                // Lay out the meta-graph
                forceAtlas2.assign(metaGraph, {
                    iterations: 300,
                    settings: {
                        gravity: 0.3,
                        scalingRatio: 500,
                        barnesHutOptimize: false,
                        strongGravityMode: true,
                        slowDown: 5,
                    },
                });

                // Seed each node near its community centroid
                // High-degree nodes go closer to center, low-degree at the edges
                for (const [cid, members] of communities) {
                    const cx = metaGraph.getNodeAttribute(cid, "x");
                    const cy = metaGraph.getNodeAttribute(cid, "y");
                    const baseSpread = Math.sqrt(members.length) * 16;
                    const maxDeg = Math.max(...members.map((n) => graph.degree(n)), 1);
                    for (const node of members) {
                        const degree = graph.degree(node);
                        // base distance from center: 0 for highest degree, 1 for lowest
                        const dist = 1 - degree / maxDeg;
                        // add randomness so same-degree nodes don't form rings
                        const jitter = 0.3 + Math.random() * 0.7;
                        const r = dist * baseSpread * jitter + Math.random() * baseSpread * 0.3;
                        const angle = Math.random() * Math.PI * 2;
                        graph.setNodeAttribute(node, "x", cx + Math.cos(angle) * r);
                        graph.setNodeAttribute(node, "y", cy + Math.sin(angle) * r);
                    }
                }

                // Set node sizes before ForceAtlas2 so adjustSizes repulsion works
                graph.forEachNode((node) => {
                    const degree = graph.degree(node);
                    graph.setNodeAttribute(node, "size", 3 + Math.sqrt(degree) * 2);
                });

                // Run ForceAtlas2 on the full graph to refine within-community positions
                // adjustSizes creates repulsion proportional to node size
                forceAtlas2.assign(graph, {
                    iterations: 600,
                    settings: {
                        gravity: 0.05,
                        scalingRatio: 50,
                        barnesHutOptimize: true,
                        barnesHutTheta: 0.5,
                        strongGravityMode: false,
                        slowDown: 10,
                        adjustSizes: true,
                        linLogMode: true,
                    },
                });


                saveAllPositions(graph);
            } else {
                // Still need to set sizes when loading saved positions
                graph.forEachNode((node) => {
                    const degree = graph.degree(node);
                    const size = 3 + Math.sqrt(degree) * 2;
                    graph.setNodeAttribute(node, "size", size);
                });
            }

            nodeCount = graph.order;
            edgeCount = graph.size;

            loading = false;
            await tick();
        } catch (err) {
            console.error("Failed to load graph:", err);
            error =
                err instanceof Error
                    ? err.message
                    : "Failed to load graph data";
            toast.error("Failed to load graph data.");
            loading = false;
            return;
        }

        if (!container) return;

        sigma = new SigmaConstructor(graph, container, {
            renderLabels: true,
            renderEdgeLabels: false,
            defaultNodeColor: "#7b8cff",
            defaultEdgeColor: "rgba(255, 255, 255, 0.035)",
            defaultEdgeType: "line",
            labelColor: { color: "rgba(200, 202, 208, 0.85)" },
            labelFont: "DM Sans, sans-serif",
            labelWeight: "300",
            labelSize: 10,
            labelRenderedSizeThreshold: 4,
            labelDensity: 0.8,
            labelGridCellSize: 120,
            minZoomRatio: 0.005,
            maxZoomRatio: 30,
            minEdgeThickness: 0.4,
            stagePadding: 40,
            defaultDrawNodeLabel: (context: CanvasRenderingContext2D, data: any, settings: any) => {
                if (!data.label) return;
                const size = settings.labelSize;
                const font = settings.labelFont;
                const weight = settings.labelWeight;
                context.fillStyle = "rgba(200, 202, 208, 0.85)";
                context.font = `${weight} ${size}px ${font}`;

                const maxWidth = 100;
                const lineHeight = size * 1.2;
                const x = data.x + data.size + 3;
                const lines = wrapText(context, data.label, maxWidth);
                const startY = data.y + size / 3;
                for (let i = 0; i < lines.length; i++) {
                    context.fillText(lines[i], x, startY + i * lineHeight);
                }
            },
            defaultDrawNodeHover: (context: CanvasRenderingContext2D, data: any, settings: any) => {
                const size = settings.labelSize;
                const font = settings.labelFont;
                const weight = settings.labelWeight;
                context.font = `${weight} ${size}px ${font}`;

                const PADDING = 4;
                const bgColor = "rgba(14, 14, 22, 0.92)";
                const borderColor = "rgba(123, 140, 255, 0.25)";
                const labelColor = "#e0e2e8";
                const maxWidth = 120;
                const lineHeight = size * 1.2;

                if (typeof data.label === "string") {
                    const lines = wrapText(context, data.label, maxWidth);

                    const textWidth = Math.min(maxWidth, Math.max(...lines.map((l) => context.measureText(l).width)));
                    const boxWidth = Math.round(textWidth + 8);
                    const boxHeight = Math.round(lines.length * lineHeight + 2 * PADDING);
                    const radius = Math.max(data.size, size / 2) + PADDING;
                    const xStart = data.x + radius;

                    // Background pill
                    context.fillStyle = bgColor;
                    context.shadowOffsetX = 0;
                    context.shadowOffsetY = 2;
                    context.shadowBlur = 12;
                    context.shadowColor = "rgba(0, 0, 0, 0.5)";
                    context.beginPath();
                    context.roundRect(
                        xStart - 2,
                        data.y - boxHeight / 2,
                        boxWidth + 4,
                        boxHeight,
                        4
                    );
                    context.fill();

                    // Border
                    context.shadowBlur = 0;
                    context.strokeStyle = borderColor;
                    context.lineWidth = 1;
                    context.stroke();

                    // Node circle with glow
                    context.beginPath();
                    context.arc(data.x, data.y, data.size + PADDING, 0, Math.PI * 2);
                    context.fillStyle = bgColor;
                    context.fill();

                    // Node dot
                    context.beginPath();
                    context.arc(data.x, data.y, data.size, 0, Math.PI * 2);
                    context.fillStyle = data.color;
                    context.fill();

                    // Label text (wrapped)
                    context.fillStyle = labelColor;
                    context.shadowBlur = 0;
                    const textX = data.x + data.size + PADDING + 3;
                    const textStartY = data.y - (lines.length - 1) * lineHeight / 2 + size / 3;
                    for (let i = 0; i < lines.length; i++) {
                        context.fillText(lines[i], textX, textStartY + i * lineHeight);
                    }
                } else {
                    context.beginPath();
                    context.arc(data.x, data.y, data.size + PADDING, 0, Math.PI * 2);
                    context.fillStyle = bgColor;
                    context.shadowBlur = 12;
                    context.shadowColor = "rgba(0, 0, 0, 0.5)";
                    context.fill();
                    context.shadowBlur = 0;

                    // Node dot
                    context.beginPath();
                    context.arc(data.x, data.y, data.size, 0, Math.PI * 2);
                    context.fillStyle = data.color;
                    context.fill();
                }
            },
            nodeReducer: (node, data) => {
                const res = { ...data };
                if (hoveredNode) {
                    if (node === hoveredNode) {
                        res.highlighted = true;
                        res.zIndex = 1;
                    } else if (graph.hasEdge(node, hoveredNode) || graph.hasEdge(hoveredNode, node)) {
                        res.highlighted = true;
                        res.zIndex = 1;
                    } else {
                        res.color = "rgba(60, 60, 80, 0.3)";
                        res.label = "";
                        res.zIndex = 0;
                    }
                }
                return res;
            },
            edgeReducer: (edge, data) => {
                const res = { ...data };
                if (hoveredNode) {
                    const src = graph.source(edge);
                    const tgt = graph.target(edge);
                    if (src === hoveredNode || tgt === hoveredNode) {
                        res.color = "rgba(123, 140, 255, 0.25)";
                        res.size = 1;
                        res.zIndex = 1;
                    } else {
                        res.color = "rgba(255, 255, 255, 0.008)";
                        res.zIndex = 0;
                    }
                }
                return res;
            },
        });

        // Hover interactions
        sigma.on("enterNode", ({ node }) => {
            hoveredNode = node;
            container.style.cursor = "pointer";
            sigma.refresh();
        });

        sigma.on("leaveNode", () => {
            hoveredNode = null;
            container.style.cursor = "default";
            sigma.refresh();
        });

        // Node dragging state
        let draggedNode: string | null = null;
        let isDragging = false;
        let hasDragged = false;

        // Node clicks — only navigate if the user didn't drag
        sigma.on("clickNode", ({ node }) => {
            if (hasDragged) return;
            if (/^[a-zA-Z0-9_-]+$/.test(node)) {
                navigate(`/notes/${encodeURIComponent(node)}`);
            } else {
                console.error("Invalid node ID:", node);
                toast.error("Unable to navigate to this node");
            }
        });

        let dragStartX = 0;
        let dragStartY = 0;

        sigma.on("downNode", (e) => {
            draggedNode = e.node;
            isDragging = true;
            hasDragged = false;
            dragStartX = graph.getNodeAttribute(draggedNode, "x");
            dragStartY = graph.getNodeAttribute(draggedNode, "y");
            if (sigma) {
                sigma.getGraph().setNodeAttribute(draggedNode, "highlighted", true);
            }
        });

        sigma.getMouseCaptor().on("mousemovebody", (e: any) => {
            if (isDragging && draggedNode && sigma) {
                hasDragged = true;
                const pos = sigma.viewportToGraph(e);
                const dx = pos.x - graph.getNodeAttribute(draggedNode, "x");
                const dy = pos.y - graph.getNodeAttribute(draggedNode, "y");

                graph.setNodeAttribute(draggedNode, "x", pos.x);
                graph.setNodeAttribute(draggedNode, "y", pos.y);

                // Drag neighbors along — smaller nodes move more, bigger ones resist
                graph.forEachNeighbor(draggedNode, (neighbor) => {
                    const nx = graph.getNodeAttribute(neighbor, "x");
                    const ny = graph.getNodeAttribute(neighbor, "y");
                    const size = graph.getNodeAttribute(neighbor, "size") || 3;
                    const influence = 2 / size;
                    graph.setNodeAttribute(neighbor, "x", nx + dx * influence);
                    graph.setNodeAttribute(neighbor, "y", ny + dy * influence);
                });

                e.preventSigmaDefault();
                e.original.preventDefault();
                e.original.stopPropagation();
            }
        });

        sigma.getMouseCaptor().on("mouseup", () => {
            if (draggedNode && sigma) {
                const graph = sigma.getGraph();
                if (graph.hasNode(draggedNode)) {
                    graph.setNodeAttribute(draggedNode, "highlighted", false);

                    if (hasDragged) {
                        // Run noverlap to resolve any overlaps caused by the drag
                        noverlap.assign(graph, {
                            maxIterations: 100,
                            settings: {
                                margin: 2,
                                ratio: 1.5,
                                speed: 3,
                            },
                        });

                        saveAllPositions(graph);
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

    onDestroy(() => {
        if (sigma) sigma.kill();
    });

    function handleZoomIn() {
        if (!sigma) return;
        sigma.getCamera().animatedZoom({ duration: 250 });
    }

    function handleZoomOut() {
        if (!sigma) return;
        sigma.getCamera().animatedUnzoom({ duration: 250 });
    }

    function handleReset() {
        if (!sigma) return;
        sigma.getCamera().animatedReset({ duration: 400 });
    }
</script>

<div class="graph-container">
    {#if loading}
        <div class="loading-container">
            <LoadingSpinner size="large" message="Loading graph..." />
        </div>
    {:else if error}
        <div class="error-container">
            <div class="error-icon-wrap">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="8" x2="12" y2="12"></line>
                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
            </div>
            <p class="error-message">{error}</p>
            <button class="retry-button" on:click={() => window.location.reload()}>
                Retry
            </button>
        </div>
    {:else}
        <div class="graph-canvas" bind:this={container}></div>

        <div class="controls">
            <button on:click={handleZoomIn} title="Zoom In" aria-label="Zoom in">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5">
                    <line x1="7" y1="2" x2="7" y2="12"></line>
                    <line x1="2" y1="7" x2="12" y2="7"></line>
                </svg>
            </button>
            <button on:click={handleZoomOut} title="Zoom Out" aria-label="Zoom out">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5">
                    <line x1="2" y1="7" x2="12" y2="7"></line>
                </svg>
            </button>
            <div class="control-divider"></div>
            <button on:click={handleReset} title="Reset View" aria-label="Reset view">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M2 7a5 5 0 0 1 9.2-2.7"></path>
                    <path d="M12 7a5 5 0 0 1-9.2 2.7"></path>
                    <polyline points="11 2 11.2 4.7 8.5 4.3"></polyline>
                    <polyline points="3 12 2.8 9.3 5.5 9.7"></polyline>
                </svg>
            </button>
        </div>

        <div class="graph-stats">
            <span class="stat">{nodeCount.toLocaleString()} nodes</span>
            <span class="stat-sep"></span>
            <span class="stat">{edgeCount.toLocaleString()} edges</span>
        </div>
    {/if}
</div>

<style>
    .graph-container {
        width: 100%;
        height: 100%;
        position: relative;
        background: var(--color-void);
    }

    .graph-canvas {
        width: 100%;
        height: 100%;
    }

    /* Override sigma's default canvas background */
    .graph-canvas :global(canvas) {
        background: transparent !important;
    }

    .controls {
        position: absolute;
        bottom: 24px;
        right: 24px;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2px;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: var(--radius-md);
        padding: 4px;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    }

    .controls button {
        width: 32px;
        height: 32px;
        border: none;
        background: transparent;
        color: var(--color-text-secondary);
        border-radius: var(--radius-sm);
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.15s var(--ease-out);
    }

    .controls button:hover {
        background: var(--color-accent-dim);
        color: var(--color-accent);
    }

    .control-divider {
        width: 18px;
        height: 1px;
        background: var(--color-border);
        margin: 2px 0;
    }

    .graph-stats {
        position: absolute;
        bottom: 24px;
        left: 24px;
        display: flex;
        align-items: center;
        gap: 10px;
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--color-text-muted);
        letter-spacing: 0.02em;
    }

    .stat-sep {
        width: 3px;
        height: 3px;
        background: var(--color-text-muted);
        border-radius: 50%;
        opacity: 0.5;
    }

    .loading-container,
    .error-container {
        width: 100%;
        height: 100%;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 1rem;
    }

    .error-icon-wrap {
        color: var(--color-text-muted);
    }

    .error-message {
        color: var(--color-text-secondary);
        font-size: 13px;
        text-align: center;
        max-width: 300px;
        font-weight: 300;
    }

    .retry-button {
        background: var(--color-surface);
        color: var(--color-text);
        border: 1px solid var(--color-border);
        padding: 8px 20px;
        border-radius: var(--radius-sm);
        cursor: pointer;
        font-family: var(--font-body);
        font-size: 13px;
        font-weight: 400;
        transition: all 0.2s var(--ease-out);
    }

    .retry-button:hover {
        border-color: var(--color-border-focus);
        background: var(--color-surface-raised);
    }
</style>
