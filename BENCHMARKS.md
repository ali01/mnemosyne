# Performance Benchmarks

## Overview

Mnemosyne indexes Obsidian vaults into SQLite and serves an interactive graph visualization. Key performance characteristics:

- **Vault parsing**: Sub-millisecond per file for typical documents
- **Full index** (1,588 files): ~130ms parse + ~200ms DB write = ~330ms total
- **Link resolution**: O(1) lookups (12-23ns)
- **Incremental re-index**: Same as full index (re-parses entire vault for correct link resolution)

## Vault Parser

### WikiLink Extraction

| Pattern Type | Time/Op | Memory/Op | Allocations |
|---|---|---|---|
| Simple | 850 ns | 547 B | 5 |
| Multiple links | 1,394 ns | 1,160 B | 11 |
| Complex patterns | 1,838 ns | 1,176 B | 12 |
| Large documents | 1.13 ms | 358 KB | 3,010 |

### Frontmatter Parsing

| Document Size | Time/Op | Memory/Op | Allocations |
|---|---|---|---|
| Minimal | 3.99 us | 14.6 KB | 85 |
| Typical | 12.06 us | 21.6 KB | 233 |
| Large | 77.85 us | 77.2 KB | 1,258 |

### Link Resolution

Constant time regardless of vault size (10 to 10,000 files):

| Lookup Type | Time/Op | Memory/Op |
|---|---|---|
| Exact path | 12.5 ns | 0 B |
| Basename | 23.4 ns | 0 B |
| Normalized | 118 ns | 56 B |

### Full Document Processing

| Document Type | Time/Op | Memory/Op |
|---|---|---|
| Minimal | 4.53 us | 15.3 KB |
| Typical | 9.15 us | 19.1 KB |
| Large | 853 us | 375 KB |

## Real Vault Performance

Tested with a 1,588-file Obsidian vault:

| Operation | Time |
|---|---|
| Full vault parse (4 workers) | ~130ms |
| Graph building | ~1.2ms |
| SQLite bulk insert (855 nodes + 518 edges) | ~200ms |
| Full index (parse + build + store) | ~330ms |
| API graph response | <50ms |
| FTS5 search query | <10ms |

## Capacity Planning

For a 50,000 node vault (estimated):
- **Parsing time**: ~457ms (9.15us x 50,000)
- **Memory**: ~955 MB in-memory during parse (19.1 KB x 50,000)
- **SQLite DB size**: ~50-100 MB
- **Graph API response**: May need pagination for acceptable latency

## Running Benchmarks

```bash
go test -bench=. -benchmem ./internal/vault/...
go test -bench=. -benchmem ./internal/models/...

# With profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/vault/...
go tool pprof cpu.prof
```

## Test Environment

- Platform: Darwin (macOS)
- Go: 1.23+
- SQLite: modernc.org/sqlite (pure Go)
- CPU: GOMAXPROCS=14
