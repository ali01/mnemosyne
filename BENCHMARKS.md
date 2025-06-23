# Performance Benchmarks

This document tracks performance benchmarks for the Mnemosyne backend, providing a baseline for optimization efforts and capacity planning.

## Executive Summary

The Mnemosyne backend demonstrates excellent performance characteristics suitable for handling large Obsidian vaults with up to 50,000 nodes:

- **Vault parsing**: Sub-millisecond processing for typical documents
- **Link resolution**: O(1) lookups with 12-23ns response times
- **Memory usage**: Linear scaling with content size
- **Validation**: Sub-microsecond node validation

## Vault Parser Performance

### Link Extraction (`ExtractWikiLinks`)

| Pattern Type      | Time/Op  | Memory/Op | Allocations |
|-------------------|----------|-----------|-------------|
| Simple            | 850 ns   | 547 B     | 5           |
| Multiple links    | 1,394 ns | 1,160 B   | 11          |
| Complex patterns  | 1,838 ns | 1,176 B   | 12          |
| Large documents   | 1.13 ms  | 358 KB    | 3,010       |

### Frontmatter Parsing (`ExtractFrontmatter`)

| Document Size | Time/Op  | Memory/Op | Allocations |
|---------------|----------|-----------|-------------|
| Minimal       | 3.99 μs  | 14.6 KB   | 85          |
| Typical       | 12.06 μs | 21.6 KB   | 233         |
| Large         | 77.85 μs | 77.2 KB   | 1,258       |

### Link Resolution (`LinkResolver`)

Performance remains constant regardless of vault size (10 to 10,000 files):

| Lookup Type | Time/Op | Memory/Op | Allocations |
|-------------|---------|-----------|-------------|
| Exact path  | 12.5 ns | 0 B       | 0           |
| Basename    | 23.4 ns | 0 B       | 0           |
| Normalized  | 118 ns  | 56 B      | 4           |
| Relative    | 113 ns  | 80 B      | 3           |

### Full Document Processing (`ProcessMarkdownReader`)

| Document Type | Time/Op | Memory/Op | Allocations |
|---------------|---------|-----------|-------------|
| Minimal       | 4.53 μs | 15.3 KB   | 97          |
| Typical       | 9.15 μs | 19.1 KB   | 162         |
| Large         | 853 μs  | 375 KB    | 2,019       |

## Data Model Performance

### VaultNode Serialization

#### Large Content Scaling

| Content Size | Time/Op  | Memory/Op | Allocations |
|--------------|----------|-----------|-------------|
| 1 KB         | 4.8 μs   | 3.1 KB    | 14          |
| 10 KB        | 30.7 μs  | 21.4 KB   | 14          |
| 100 KB       | 288 μs   | 219 KB    | 14          |
| 1 MB         | 2.82 ms  | 2.43 MB   | 18          |
| 10 MB        | 28.3 ms  | 32.0 MB   | 23          |

#### Complex Metadata Performance

| Field Count | Time/Op | Memory/Op | Allocations |
|-------------|---------|-----------|-------------|
| 10          | 13.6 μs | 12.9 KB   | 321         |
| 50          | 60.6 μs | 59.8 KB   | 1,565       |
| 100         | 123 μs  | 120 KB    | 3,117       |
| 500         | 646 μs  | 639 KB    | 15,525      |

### VaultEdge Serialization

| Display Text Size | Time/Op | Memory/Op | Allocations |
|-------------------|---------|-----------|-------------|
| Empty             | 1.10 μs | 673 B     | 12          |
| 50 chars          | 1.36 μs | 817 B     | 13          |
| 200 chars         | 1.84 μs | 1.12 KB   | 13          |
| 1,000 chars       | 4.17 μs | 2.84 KB   | 13          |
| 5,000 chars       | 15.6 μs | 11.3 KB   | 13          |

### Validation Performance

- **VaultNode validation**: 398 ns/op with 280 B memory and 5 allocations

## Capacity Planning

Based on these benchmarks:

### Vault Size Estimates

For a 50,000 node vault:
- **Parsing time**: ~457 ms for typical documents (9.15 μs × 50,000)
- **Memory usage**: ~955 MB for typical nodes (19.1 KB × 50,000)
- **Link resolution**: Constant time lookups regardless of vault size

### Concurrent Processing

The parser supports parallel processing with configurable worker counts:
- Linear speedup up to CPU core count
- Optimal worker count typically equals CPU cores

## Performance Considerations

### Strengths
1. **O(1) link resolution**: Excellent scalability for large vaults
2. **Linear memory scaling**: Predictable resource usage
3. **Low allocation count**: Efficient memory management for most operations

### Areas to Monitor
1. **Metadata serialization**: High allocation count (15,525 for 500 fields)
2. **Large content**: Consider lazy loading for documents > 100KB
3. **Concurrent writes**: Database connection pooling for high throughput

## Testing Environment

- **Platform**: Darwin (macOS)
- **Go Version**: 1.23.0
- **CPU**: Benchmarks run with GOMAXPROCS=14

## Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific package benchmarks
go test -bench=. -benchmem ./internal/vault/...
go test -bench=. -benchmem ./internal/models/...

# Run with CPU profiling
go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/vault/...

# Analyze profile
go tool pprof cpu.prof
```

## Benchmark History

- **2025-06-23**: Initial benchmark documentation
  - Vault parser: 94% test coverage
  - Model validation: Comprehensive test suite added
