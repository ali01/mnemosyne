// Package search implements an Obsidian-compatible search query parser and evaluator.
// Supported operators: path:, tag:, file:, [field:value], bare text, *.
// Boolean logic: implicit AND (space), OR, NOT (-), parentheses.
package search

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NodeData provides the data needed for query matching.
type NodeData struct {
	FilePath    string
	Title       string
	Tags        []string
	Frontmatter map[string]interface{}
}

// Query represents a parsed search expression that can match against nodes.
type Query interface {
	Match(n *NodeData) bool
}

// Parse parses an Obsidian search query string into a Query.
// Empty input or "*" returns a match-all query.
func Parse(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if input == "" || input == "*" {
		return matchAll{}, nil
	}
	p := &parser{input: input}
	q, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected input at position %d: %q", p.pos, p.input[p.pos:])
	}
	return q, nil
}

// --- Query types ---

type matchAll struct{}

func (matchAll) Match(*NodeData) bool { return true }

type pathFilter struct{ value string }

func (f pathFilter) Match(n *NodeData) bool {
	return strings.Contains(strings.ToLower(n.FilePath), strings.ToLower(f.value))
}

type tagFilter struct{ tag string }

func (f tagFilter) Match(n *NodeData) bool {
	needle := strings.TrimPrefix(strings.ToLower(f.tag), "#")
	for _, t := range n.Tags {
		stored := strings.TrimPrefix(strings.ToLower(t), "#")
		if stored == needle {
			return true
		}
	}
	return false
}

type fileFilter struct{ value string }

func (f fileFilter) Match(n *NodeData) bool {
	base := filepath.Base(n.FilePath)
	return strings.Contains(strings.ToLower(base), strings.ToLower(f.value))
}

type propFilter struct{ key, value string }

func (f propFilter) Match(n *NodeData) bool {
	if n.Frontmatter == nil {
		return false
	}
	val, ok := n.Frontmatter[f.key]
	if !ok {
		return false
	}
	return strings.EqualFold(fmt.Sprintf("%v", val), f.value)
}

type textFilter struct{ text string }

func (f textFilter) Match(n *NodeData) bool {
	lower := strings.ToLower(f.text)
	return strings.Contains(strings.ToLower(n.Title), lower) ||
		strings.Contains(strings.ToLower(filepath.Base(n.FilePath)), lower)
}

type andQuery struct{ clauses []Query }

func (q andQuery) Match(n *NodeData) bool {
	for _, c := range q.clauses {
		if !c.Match(n) {
			return false
		}
	}
	return true
}

type orQuery struct{ clauses []Query }

func (q orQuery) Match(n *NodeData) bool {
	for _, c := range q.clauses {
		if c.Match(n) {
			return true
		}
	}
	return false
}

type notQuery struct{ inner Query }

func (q notQuery) Match(n *NodeData) bool {
	return !q.inner.Match(n)
}

// --- Parser ---

type parser struct {
	input string
	pos   int
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) hasPrefix(prefix string) bool {
	return strings.HasPrefix(p.input[p.pos:], prefix)
}

// isORKeyword checks if the current position is the "OR" keyword
// (followed by whitespace, ')', or end of input).
func (p *parser) isORKeyword() bool {
	if p.pos+2 > len(p.input) {
		return false
	}
	if p.input[p.pos:p.pos+2] != "OR" {
		return false
	}
	if p.pos+2 == len(p.input) {
		return true
	}
	next := p.input[p.pos+2]
	return next == ' ' || next == ')'
}

// parseOr: andExpr ("OR" andExpr)*
func (p *parser) parseOr() (Query, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	clauses := []Query{left}
	for {
		p.skipWhitespace()
		if !p.isORKeyword() {
			break
		}
		p.pos += 2
		p.skipWhitespace()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, right)
	}

	if len(clauses) == 1 {
		return clauses[0], nil
	}
	return orQuery{clauses: clauses}, nil
}

// parseAnd: unary (unary)* -- implicit AND via adjacency
func (p *parser) parseAnd() (Query, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	clauses := []Query{left}
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] == ')' {
			break
		}
		if p.isORKeyword() {
			break
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, right)
	}

	if len(clauses) == 1 {
		return clauses[0], nil
	}
	return andQuery{clauses: clauses}, nil
}

// parseUnary: '-' atom | atom
func (p *parser) parseUnary() (Query, error) {
	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
		atom, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		return notQuery{inner: atom}, nil
	}
	return p.parseAtom()
}

// parseAtom: '(' query ')' | operator | '[' field ':' value ']' | text | '*'
func (p *parser) parseAtom() (Query, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of query")
	}

	ch := p.input[p.pos]

	// Parenthesized subexpression
	if ch == '(' {
		p.pos++
		q, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		p.pos++
		return q, nil
	}

	// Match all
	if ch == '*' {
		p.pos++
		return matchAll{}, nil
	}

	// Frontmatter filter: [field:"value"] or [field:value]
	if ch == '[' {
		return p.parsePropFilter()
	}

	// Operator prefixes
	if p.hasPrefix("path:") {
		p.pos += 5
		val, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("path filter: %w", err)
		}
		return pathFilter{value: val}, nil
	}
	if p.hasPrefix("tag:") {
		p.pos += 4
		val, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("tag filter: %w", err)
		}
		return tagFilter{tag: val}, nil
	}
	if p.hasPrefix("file:") {
		p.pos += 5
		val, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("file filter: %w", err)
		}
		return fileFilter{value: val}, nil
	}

	// Bare text (quoted or unquoted word)
	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	return textFilter{text: val}, nil
}

// parsePropFilter parses [field:"value"] or [field:value]
func (p *parser) parsePropFilter() (Query, error) {
	p.pos++ // skip '['

	// Read key until ':'
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != ':' && p.input[p.pos] != ']' {
		p.pos++
	}
	if p.pos >= len(p.input) || p.input[p.pos] != ':' {
		return nil, fmt.Errorf("expected ':' in frontmatter filter at position %d", p.pos)
	}
	key := strings.TrimSpace(p.input[start:p.pos])
	p.pos++ // skip ':'

	val, err := p.parseValue()
	if err != nil {
		return nil, fmt.Errorf("frontmatter filter value: %w", err)
	}

	if p.pos >= len(p.input) || p.input[p.pos] != ']' {
		return nil, fmt.Errorf("expected ']' at position %d", p.pos)
	}
	p.pos++ // skip ']'

	return propFilter{key: key, value: val}, nil
}

// parseValue reads a quoted string or an unquoted word.
func (p *parser) parseValue() (string, error) {
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("expected value at position %d", p.pos)
	}
	if p.input[p.pos] == '"' {
		return p.parseQuoted()
	}
	w := p.parseWord()
	if w == "" {
		return "", fmt.Errorf("expected value at position %d", p.pos)
	}
	return w, nil
}

func (p *parser) parseQuoted() (string, error) {
	p.pos++ // skip opening "
	var b strings.Builder
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '\\' && p.pos+1 < len(p.input) {
			p.pos++
			b.WriteByte(p.input[p.pos])
			p.pos++
			continue
		}
		if ch == '"' {
			p.pos++ // skip closing "
			return b.String(), nil
		}
		b.WriteByte(ch)
		p.pos++
	}
	return "", fmt.Errorf("unterminated quoted string")
}

// parseWord reads until whitespace or a structural character.
func (p *parser) parseWord() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == ' ' || ch == ')' || ch == ']' || ch == '(' {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos]
}
