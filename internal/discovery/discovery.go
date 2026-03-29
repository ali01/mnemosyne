// Package discovery scans vault directories for GRAPH.yaml marker files.
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ali01/mnemosyne/internal/config"
	"gopkg.in/yaml.v3"
)

// GraphDef describes a graph discovered from a GRAPH.yaml file.
type GraphDef struct {
	RootPath           string                          // relative to vault, "" for vault root
	Name               string                          // from GRAPH.yaml or directory name
	NodeClassification *config.NodeClassificationConfig // optional
	RawConfig          string                          // raw YAML content
}

// graphYAML is the structure of a GRAPH.yaml file.
type graphYAML struct {
	Name               string                          `yaml:"name"`
	NodeClassification *config.NodeClassificationConfig `yaml:"node_classification"`
}

// Discover walks a vault directory and returns a GraphDef for each GRAPH.yaml found.
// Returns an error if nested GRAPH.yaml files are detected (one is an ancestor of another).
func Discover(vaultPath string) ([]GraphDef, error) {
	var defs []GraphDef

	err := filepath.WalkDir(vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != vaultPath {
			return filepath.SkipDir
		}

		if d.Name() != "GRAPH.yaml" {
			return nil
		}

		relDir, err := filepath.Rel(vaultPath, filepath.Dir(path))
		if err != nil {
			return nil
		}
		if relDir == "." {
			relDir = ""
		}

		def, err := parseGraphYAML(path, relDir)
		if err != nil {
			return nil // skip unparseable GRAPH.yaml
		}

		defs = append(defs, def)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := validateNoNesting(defs); err != nil {
		return nil, err
	}

	return defs, nil
}

// validateNoNesting returns an error if any GRAPH.yaml is an ancestor of another.
func validateNoNesting(defs []GraphDef) error {
	for i, a := range defs {
		for j, b := range defs {
			if i == j {
				continue
			}
			if isAncestor(a.RootPath, b.RootPath) {
				return fmt.Errorf("nested GRAPH.yaml files are not allowed: %q is an ancestor of %q",
					graphYAMLPath(a.RootPath), graphYAMLPath(b.RootPath))
			}
		}
	}
	return nil
}

func isAncestor(ancestor, descendant string) bool {
	if ancestor == descendant {
		return false
	}
	if ancestor == "" {
		return true // root is ancestor of everything
	}
	return strings.HasPrefix(descendant, ancestor+"/")
}

func graphYAMLPath(rootPath string) string {
	if rootPath == "" {
		return "GRAPH.yaml"
	}
	return rootPath + "/GRAPH.yaml"
}

func parseGraphYAML(path, relDir string) (GraphDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return GraphDef{}, err
	}

	def := GraphDef{
		RootPath:  relDir,
		RawConfig: string(data),
	}

	// Parse YAML if non-empty
	if len(strings.TrimSpace(string(data))) > 0 {
		var g graphYAML
		if err := yaml.Unmarshal(data, &g); err == nil {
			if g.Name != "" {
				def.Name = g.Name
			}
			def.NodeClassification = g.NodeClassification
		}
	}

	// Default name from directory
	if def.Name == "" {
		if relDir == "" {
			def.Name = "root"
		} else {
			def.Name = filepath.Base(relDir)
		}
	}

	return def, nil
}

// IsUnderPath returns true if filePath is within graphRootPath.
// Both paths are relative to the vault root.
func IsUnderPath(filePath, graphRootPath string) bool {
	if graphRootPath == "" {
		return true // root graph contains everything
	}
	dir := filepath.Dir(filePath)
	return dir == graphRootPath || strings.HasPrefix(dir, graphRootPath+"/") || filePath == graphRootPath
}
