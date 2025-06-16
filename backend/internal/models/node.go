package models

type Node struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	FilePath string                 `json:"file_path,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Position Position               `json:"position"`
	Level    int                    `json:"level"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

type Edge struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
	Type   string  `json:"type"`
}