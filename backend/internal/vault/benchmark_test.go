// Package vault provides functionality for parsing and processing Obsidian vault markdown files
package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Benchmarks for WikiLink extraction
func BenchmarkExtractWikiLinks(b *testing.B) {
	contents := []struct {
		name    string
		content string
	}{
		{
			"simple",
			"This is a simple [[link]] in the text.",
		},
		{
			"multiple",
			"Multiple [[link1]], [[link2]], and [[link3]] in one line.",
		},
		{
			"complex",
			"Complex links like [[path/to/note|Display]], [[note#section]], and ![[embed.png]].",
		},
		{
			"large",
			strings.Repeat("Text with [[link]] and more content. ", 1000),
		},
	}

	for _, tc := range contents {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ExtractWikiLinks(tc.content)
			}
		})
	}
}

// Benchmarks for Frontmatter extraction
func BenchmarkExtractFrontmatter(b *testing.B) {
	contents := []struct {
		name    string
		content string
	}{
		{
			"minimal",
			`---
id: "test"
---
Content`,
		},
		{
			"typical",
			`---
id: "test"
tags: ["tag1", "tag2", "tag3"]
aliases: ["alias1", "alias2"]
created: 2023-01-01
---
# Title
Content here`,
		},
		{
			"large",
			`---
id: "test"
tags: [` + strings.Repeat(`"tag", `, 100) + `"last"]
metadata:
  key1: value1
  key2: value2
  nested:
    deep: value
---
` + strings.Repeat("Content line\n", 1000),
		},
	}

	for _, tc := range contents {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = ExtractFrontmatter(tc.content)
			}
		})
	}
}

// Benchmarks for Link Resolution
func BenchmarkLinkResolver(b *testing.B) {
	// Create resolver with various numbers of files
	fileCounts := []int{10, 100, 1000, 10000}

	for _, count := range fileCounts {
		b.Run(fmt.Sprintf("files_%d", count), func(b *testing.B) {
			resolver := NewLinkResolver()

			// Add files
			for i := 0; i < count; i++ {
				file := &MarkdownFile{
					Path: fmt.Sprintf("folder%d/file%d.md", i%10, i),
					Frontmatter: &FrontmatterData{
						ID: fmt.Sprintf("id-%d", i),
					},
				}
				resolver.AddFile(file)
			}

			// Benchmark different resolution patterns
			b.Run("exact_path", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resolver.ResolveLink("folder5/file55.md", "")
				}
			})

			b.Run("basename", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resolver.ResolveLink("file55", "")
				}
			})

			b.Run("normalized", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resolver.ResolveLink("FILE-55", "")
				}
			})

			b.Run("relative", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resolver.ResolveLink("../folder5/file55", "folder3/file33.md")
				}
			})
		})
	}
}

// Benchmark full markdown processing
func BenchmarkProcessMarkdownReader(b *testing.B) {
	contents := []struct {
		name    string
		content string
	}{
		{
			"minimal",
			`---
id: "test"
---
Simple content`,
		},
		{
			"typical",
			`---
id: "test"
tags: ["tag1", "tag2"]
---
# Title

This has [[link1]] and [[link2]].

## Section
More content with [[link3|Display Text]].`,
		},
		{
			"large",
			`---
id: "test"
tags: ["performance", "benchmark"]
metadata:
  size: large
---
# Large Document

` + strings.Repeat("This is a paragraph with [[links]] and content. ", 100) +
				"\n\n" + strings.Repeat("More paragraphs. [[another-link]] here. ", 500),
		},
	}

	for _, tc := range contents {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(tc.content)
				_, _ = ProcessMarkdownReader(reader, "test.md")
			}
		})
	}
}

// Benchmark vault parsing with concurrent workers
func BenchmarkParseVault(b *testing.B) {
	// Skip if short
	if testing.Short() {
		b.Skip("Skipping vault parsing benchmark in short mode")
	}

	// Create temporary vault with files
	tempDir := b.TempDir()
	fileCount := 100

	for i := 0; i < fileCount; i++ {
		content := fmt.Sprintf(`---
id: "note-%d"
tags: ["bench"]
---
# Note %d

Links: [[note-%d]], [[note-%d]]
`, i, i, (i+1)%fileCount, (i+10)%fileCount)

		path := filepath.Join(tempDir, fmt.Sprintf("note-%d.md", i))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			b.Fatal(err)
		}
	}

	// Benchmark with different concurrency levels
	concurrencyLevels := []int{1, 2, 4, 8}

	for _, workers := range concurrencyLevels {
		b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				parser := NewParser(tempDir, workers, 50)
				_, _ = parser.ParseVault()
			}
		})
	}
}

// Benchmark normalizeForMatching function
func BenchmarkNormalizeForMatching(b *testing.B) {
	inputs := []string{
		"simple",
		"hyphen-separated",
		"underscore_separated",
		"~prefix-removed",
		"MIXED-case_Example",
		"multiple---separators___here",
		"Very Long File Name With Many Words And Separators",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			_ = normalizeForMatching(input)
		}
	}
}

// Benchmark title extraction
func BenchmarkExtractTitle(b *testing.B) {
	paths := []string{
		"simple.md",
		"path/to/file.md",
		"complex-name_with-many_separators.md",
		"~special-prefix.md",
		strings.Repeat("very-long-", 20) + "filename.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = extractTitle(path)
		}
	}
}
