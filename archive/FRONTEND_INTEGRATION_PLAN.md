# Next Workstream: Post-Phase 3.5 MVP Completion

STATUS: IMPLEMENTED

## Overview

This document outlines the work required after completing Phase 3.5 (VaultService implementation) to reach a fully functional MVP of the Mnemosyne graph visualizer. The frontend has a solid foundation with working graph visualization, but lacks essential features for viewing node content and persisting user interactions.

## Current Frontend State

### What's Working
- Sigma.js graph visualization with WebGL rendering
- Node color coding by type (core, sub, detail)
- Zoom controls (+, -, reset)
- Node dragging functionality
- Basic graph store with node/edge management
- Dark theme and responsive layout
- API integration for fetching graph data

### What's Missing
- Node content viewer (critical)
- Node position persistence after dragging
- Search functionality
- Loading states and error handling
- Node type legend
- Level-based zoom controls
- Node tooltips and metadata display
- Edge labels and differentiation
- Settings/configuration UI
- Parse status indicators

## MVP Requirements

### 1. Critical Frontend Features (5-7 days)

#### Node Position Persistence ⭐ HIGHEST PRIORITY
**Implementation Plan:**
```
1. Capture drag-end events in graph visualization
2. Call API: PUT /api/v1/nodes/:id/position
   # TODO(CR): Inconsistent API Endpoint Naming - Verify these endpoints match the actual
   # backend API design. Consider using consistent pluralization (nodes vs node).
3. Update local store to reflect saved positions
4. Ensure positions reload on page refresh
5. Add visual feedback for save status
6. Implement batch updates for performance
   # TODO(CR): Race Condition in Position Updates - The plan mentions "batch updates for
   # performance" but doesn't address potential race conditions when multiple users drag
   # nodes simultaneously. Consider implementing: Optimistic updates with rollback on
   # failure, Version/timestamp checking to prevent overwriting newer positions,
   # Conflict resolution strategy
```

#### Node Content Viewer
**Implementation Plan:**
```
1. Create new route: /notes/[id]/+page.svelte
2. Fetch node content from API: GET /api/v1/nodes/:id/content
   # TODO(CR): Inconsistent API Endpoint Naming - Verify these endpoints match the actual
   # backend API design. Consider using consistent pluralization (nodes vs node).
3. Implement markdown renderer using 'marked' library
4. Handle WikiLink conversion to clickable links
   # TODO(CR): WikiLink Navigation Security - Converting WikiLinks to clickable links needs
   # XSS prevention. Ensure proper sanitization when rendering markdown and converting
   # WikiLinks to HTML. Use a trusted markdown parser with XSS protection.
5. Add navigation between linked notes
6. Include back-to-graph button
7. Handle non-existent nodes gracefully
   # TODO(CR): Missing Error Response Specifications - The implementation plan doesn't
   # specify how to handle error responses from the API. Add error handling specifications
   # for common scenarios: 404: Node not found, 500: Server error, Network failures
```

**Key Components:**
- `NoteViewer.svelte` - Main content display component
- `WikiLinkRenderer.svelte` - Custom renderer for WikiLinks
- Update `graphStore.js` to handle node selection and navigation

#### Search Functionality
**Implementation Plan:**
```
1. Add SearchBar.svelte component
2. Integrate with backend: GET /api/v1/search?q=query
   # TODO(CR): Inconsistent API Endpoint Naming - Verify these endpoints match the actual
   # backend API design. Consider using consistent pluralization (nodes vs node).
   # TODO(CR): Search Query Injection - Search endpoint needs input validation to prevent
   # injection attacks. Implement: Query parameter sanitization, Length limits on search
   # queries, Rate limiting for search requests
3. Display results in dropdown/sidebar
4. Implement graph highlighting for results
5. Add keyboard navigation for results
6. Support search filters (type, tags)
```

#### Error Handling & Loading States
**Implementation Plan:**
```
1. Create LoadingSpinner.svelte component
2. Add error boundary for graph failures
3. Implement toast notifications for errors
4. Add retry mechanisms for failed requests
5. Handle empty graph states
6. Show connection status indicators
```

### 2. Backend Completions (2-3 days)

#### WikiLink to HTML Conversion
```go
// Add to NodeService
func (s *NodeService) GetRenderedContent(ctx context.Context, nodeID string) (string, error) {
    // 1. Fetch node content
    // 2. Parse markdown with custom WikiLink renderer
    // 3. Convert [[Note]] to <a href="/notes/{id}">Note</a>
    // 4. Return HTML string
}
```

#### Graph Filtering API
```
GET /api/v1/graph?level=0&types=core,sub&tags=important&component=main
```
- Add query parameter parsing
- Implement filtering in repository layer
- Support OR/AND logic for filters

#### Vault Status Endpoint
```
GET /api/v1/vault/info
Response: {
    "last_sync": "2024-01-20T10:30:00Z",
    "last_parse": "2024-01-20T10:31:00Z",
    "node_count": 594,
    "edge_count": 1823,
    "status": "synced"
}
```

### 3. UI/UX Improvements (3-4 days)

#### Graph Controls Panel
**Components:**
- `GraphLegend.svelte` - Visual legend for node types
- `ZoomControls.svelte` - Level selector (0-3)
- `LayoutSelector.svelte` - Choose layout algorithm
- `FilterPanel.svelte` - Filter by type/tags

#### Node Information Display
**Features:**
- Hover tooltips showing title, type, tags
- Click to show detailed info panel
- Display connection count
- Show creation/modification dates
- List connected nodes

#### Visual Enhancements
- Different edge styles for link types
- Node size based on connection count
- Smooth transitions for state changes
- Highlight node neighborhoods on hover
- Animated graph layouts

### 4. Essential Configuration (2-3 days)

#### Parse Control UI
**Components:**
- `ParseButton.svelte` - Trigger vault re-parse
- `ParseStatus.svelte` - Show parsing progress
- `ParseHistory.svelte` - List previous parses
- `ErrorLog.svelte` - Display parse errors

#### Basic Settings Panel
**Features:**
- Theme toggle (dark/light)
- Performance options (render quality)
- Graph physics settings
- Auto-save toggle for positions
- Debug mode toggle

### 5. Performance & Polish (2-3 days)

#### Frontend Optimization
- Implement viewport culling for off-screen nodes
- Use Web Workers for graph calculations
- Add IndexedDB caching for graph data
  # TODO(CR): Memory Leak Risk with Graph Data - Caching entire graph data in IndexedDB
  # without cleanup strategy could lead to storage exhaustion. Implement: Cache size limits,
  # LRU eviction policy, Clear cache on version updates
- Debounce position updates (100ms)
- Lazy load node content

#### Backend Optimization
- Add cache headers (1 hour for graph data)
- Implement ETag for graph endpoints
- Enable gzip compression
- Add request coalescing for positions

### 6. Documentation & Testing (2-3 days)

#### User Documentation
- README with setup instructions
- Feature walkthrough with screenshots
- Keyboard shortcuts guide
- Troubleshooting common issues
- Video tutorial for first-time users

#### Testing Suite
- Playwright tests for critical paths
- Visual regression tests for graph
- API integration tests
- Performance benchmarks
- Accessibility testing
  # TODO(CR): Accessibility Testing Underspecified - Accessibility testing is mentioned but
  # not detailed. Specify: WCAG compliance level (AA recommended), Automated tools
  # (axe-core), Manual testing checklist

## Implementation Priority

### Week 1: Core Functionality
**Goal**: Users can view and interact with their knowledge graph

Day 1-2: Node Content Viewer
- Implement route and component
- Add markdown rendering
- Handle WikiLinks

Day 3: Position Persistence
- Wire up drag events
- Implement API calls
- Add save indicators

Day 4-5: Search
- Create search UI
- Integrate with API
- Add result highlighting

Day 6-7: Polish
- Loading states
- Error handling
- Bug fixes

### Week 2: Enhanced UX
**Goal**: Improve usability and add essential features

Day 1-2: Graph Controls
- Node type legend
- Zoom level controls
- Filter panel

Day 3-4: Information Display
- Node tooltips
- Info panel
- Metadata display

Day 5-7: Configuration
- Parse controls
- Settings panel
- Status indicators

### Week 3: Production Ready
**Goal**: Performance, testing, and documentation

Day 1-2: Performance
- Frontend optimizations
- Backend caching
- Load testing

Day 3-4: Testing
- E2E test suite
- Visual regression
- Cross-browser testing

Day 5-7: Documentation
- User guides
- API documentation
- Deployment guide

## Minimal MVP Path (1 week)

If time is critical, implement only these features:

1. **Node Content Viewer** (2 days)
   - Basic markdown display
   - WikiLink navigation
   - Back to graph button

2. **Position Persistence** (1 day)
   - Save on drag-end
   - Restore on load

3. **Basic Search** (1 day)
   - Simple search bar
   - Highlight results

4. **Error Handling** (1 day)
   - Loading spinner
   - Error messages
   - Retry buttons

5. **Testing & Fixes** (2 days)
   - Critical path testing
   - Bug fixes
   - Basic documentation

## Success Metrics

### Functional Requirements
- [ ] Users can click any node to read its content
- [ ] Graph layouts persist between sessions
- [ ] Search finds nodes by title and content
- [ ] All errors show helpful messages
- [ ] Parse status is visible to users

### Performance Requirements
- [ ] Graph loads in <2s for 1k nodes
- [ ] Search returns results in <200ms
- [ ] Position updates save in <100ms
- [ ] Content loads in <500ms
# TODO(CR): Missing Performance Test Specifications - Performance targets are defined but no
# testing strategy to verify them. Add performance testing plan: Load testing tools (k6,
# Artillery), Benchmark scenarios, Performance regression detection

### Quality Requirements
- [ ] No console errors in normal use
- [ ] All actions have loading states
- [ ] Mobile responsive (tablet+)
- [ ] Keyboard navigable
- [ ] 90%+ test coverage for critical paths

## Technical Decisions
# TODO(CR): No Mention of TypeScript Types - No mention of TypeScript interfaces for new API
# responses and components. Add section on type definitions needed for: API response types,
# Component prop types, Store state types

### Frontend Architecture
- **Routing**: Use SvelteKit file-based routing
- **State**: Enhance existing graphStore
  # TODO(CR): Missing State Management Details - "Enhance existing graphStore" is vague
  # without specifications. Define: Store structure for new features, Action/mutation
  # patterns, Subscription management
- **Styling**: Continue with existing dark theme
- **Components**: Small, focused, reusable

### API Design
- **RESTful**: Stick with current patterns
- **Errors**: Consistent error format
- **Pagination**: Already implemented
- **Caching**: Use HTTP cache headers

### Testing Strategy
- **E2E**: Playwright for user flows
- **Visual**: Percy for regression
- **Unit**: Vitest for components
- **API**: Supertest for endpoints

## Risks and Mitigations

### Technical Risks

1. **Large Graph Performance**
   - Risk: Browser crashes with 50k nodes
   - Mitigation: Viewport culling, level-based loading

2. **Search Performance**
   - Risk: Slow searches in large vaults
   - Mitigation: Backend indexing, debounced input

3. **Position Update Storms**
   - Risk: Too many API calls during dragging
   - Mitigation: Debounce, batch updates

### User Experience Risks

1. **Confusing Navigation**
   - Risk: Users get lost between graph/notes
   - Mitigation: Clear breadcrumbs, back buttons

2. **Lost Work**
   - Risk: Positions not saving
   - Mitigation: Auto-save, visual feedback

3. **Slow Initial Load**
   - Risk: Users abandon on blank screen
   - Mitigation: Progressive loading, skeleton UI

## Next Steps After MVP

Once the MVP is complete:

1. **Phase 4 Implementation**
   - Redis caching
   - Advanced algorithms
   - Performance monitoring

2. **Phase 5 Planning**
   - File watching design
   - WebSocket architecture
   - Incremental updates

3. **User Feedback**
   - Beta testing program
   - Feature prioritization
   - Performance profiling

This completes the post-Phase 3.5 workstream plan. The focus is on delivering a functional, usable graph visualizer that allows users to explore their Obsidian vault visually and read their notes seamlessly.
