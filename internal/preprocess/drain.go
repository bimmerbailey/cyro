package preprocess

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// DrainExtractor implements the Drain algorithm for log template extraction.
//
// The Drain algorithm is a tree-based approach that groups similar log messages
// into templates by building a parse tree with a fixed maximum depth.
//
// Key concepts:
//   - Templates represent the "shape" of log messages (e.g., "Connected to <*>:<*>")
//   - Tokens that vary (numbers, IPs, IDs) are replaced with wildcards
//   - Similar messages are clustered together based on token overlap
//
// Configuration:
//   - depth: Maximum tree depth (default: 4)
//   - simThreshold: Minimum similarity for template matching (default: 0.5)
//   - maxChildren: Maximum children per tree node (default: 100)
type DrainExtractor struct {
	root         *ParseTreeNode
	depth        int
	simThreshold float64
	maxChildren  int
	templates    map[string]*Template // Template ID -> Template
	tokenRegex   *regexp.Regexp       // For tokenizing messages
	mu           sync.RWMutex
}

// ParseTreeNode represents a node in the Drain parse tree.
type ParseTreeNode struct {
	nodeType    NodeType
	token       string                    // For token nodes
	children    map[string]*ParseTreeNode // Child nodes by token
	templateIDs []string                  // For leaf nodes - can have multiple templates
}

// NodeType represents the type of a parse tree node.
type NodeType int

const (
	RootNode     NodeType = iota
	LengthNode            // Groups by token count
	TokenNode             // Matches specific tokens
	WildcardNode          // Matches any token (variable fields)
)

// Template represents an extracted log template.
type Template struct {
	ID       string   // Unique identifier
	Pattern  string   // Template string with wildcards
	Tokens   []string // Tokenized pattern
	Count    int      // Number of log lines matching this template
	Examples []string // Sample raw messages (limited)
}

// DefaultDrainConfig provides sensible defaults for the Drain algorithm.
var DefaultDrainConfig = struct {
	Depth        int
	SimThreshold float64
	MaxChildren  int
}{
	Depth:        4,
	SimThreshold: 0.5,
	MaxChildren:  100,
}

// NewDrainExtractor creates a new Drain extractor with the specified configuration.
// Use nil for default configuration.
func NewDrainExtractor(depth int, simThreshold float64, maxChildren int) *DrainExtractor {
	if depth <= 0 {
		depth = DefaultDrainConfig.Depth
	}
	if simThreshold <= 0 || simThreshold > 1 {
		simThreshold = DefaultDrainConfig.SimThreshold
	}
	if maxChildren <= 0 {
		maxChildren = DefaultDrainConfig.MaxChildren
	}

	return &DrainExtractor{
		root: &ParseTreeNode{
			nodeType:    RootNode,
			children:    make(map[string]*ParseTreeNode),
			templateIDs: []string{},
		},
		depth:        depth,
		simThreshold: simThreshold,
		maxChildren:  maxChildren,
		templates:    make(map[string]*Template),
		tokenRegex:   regexp.MustCompile(`[^\s]+`), // Split by whitespace
	}
}

// Extract processes a log message and returns the template ID it matches.
// If no matching template exists, a new one is created.
func (d *DrainExtractor) Extract(message string) string {
	tokens := d.tokenize(message)
	if len(tokens) == 0 {
		return ""
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Walk the tree to find or create a matching template
	templateID := d.findOrCreateTemplate(tokens)

	// Update template statistics
	if template, ok := d.templates[templateID]; ok {
		template.Count++
		// Keep up to 3 examples
		if len(template.Examples) < 3 {
			template.Examples = append(template.Examples, message)
		}
	}

	return templateID
}

// ExtractWithTemplate processes a log message and returns both the template ID
// and the template pattern.
func (d *DrainExtractor) ExtractWithTemplate(message string) (string, string) {
	templateID := d.Extract(message)
	d.mu.RLock()
	defer d.mu.RUnlock()

	if template, ok := d.templates[templateID]; ok {
		return templateID, template.Pattern
	}
	return templateID, ""
}

// findOrCreateTemplate traverses the parse tree to find a matching template
// or create a new one.
func (d *DrainExtractor) findOrCreateTemplate(tokens []string) string {
	// Level 0: Root node - start here
	currentNode := d.root

	// Level 1: Length node - group by token count
	lengthKey := fmt.Sprintf("len_%d", len(tokens))
	if _, ok := currentNode.children[lengthKey]; !ok {
		currentNode.children[lengthKey] = &ParseTreeNode{
			nodeType:    LengthNode,
			children:    make(map[string]*ParseTreeNode),
			templateIDs: []string{},
		}
	}
	currentNode = currentNode.children[lengthKey]

	// Levels 2 to depth-1: Token matching
	for i := 0; i < len(tokens) && i < d.depth-1; i++ {
		token := tokens[i]

		// Check if we should treat this token as a wildcard (variable field)
		if d.isVariableToken(token) {
			token = "<*>"
		}

		if _, ok := currentNode.children[token]; !ok {
			// Check if we can create a new child
			if len(currentNode.children) >= d.maxChildren {
				// Too many children, use wildcard
				token = "<*>"
				if _, ok := currentNode.children[token]; !ok {
					currentNode.children[token] = &ParseTreeNode{
						nodeType:    WildcardNode,
						children:    make(map[string]*ParseTreeNode),
						templateIDs: []string{},
					}
				}
			} else {
				currentNode.children[token] = &ParseTreeNode{
					nodeType:    TokenNode,
					token:       token,
					children:    make(map[string]*ParseTreeNode),
					templateIDs: []string{},
				}
			}
		}
		currentNode = currentNode.children[token]
	}

	// Leaf level: Check similarity with all existing templates at this leaf
	for _, existingID := range currentNode.templateIDs {
		if template, ok := d.templates[existingID]; ok {
			similarity := d.calculateSimilarity(tokens, template.Tokens)
			if similarity >= d.simThreshold {
				// Merge with existing template
				newTokens := d.mergeTemplates(template.Tokens, tokens)
				template.Tokens = newTokens
				template.Pattern = d.tokensToPattern(newTokens)
				return existingID
			}
		}
	}

	// Create new template
	templateID := d.generateTemplateID()
	templateTokens := d.createTemplateTokens(tokens)
	template := &Template{
		ID:       templateID,
		Pattern:  d.tokensToPattern(templateTokens),
		Tokens:   templateTokens,
		Count:    0,
		Examples: []string{},
	}

	d.templates[templateID] = template
	currentNode.templateIDs = append(currentNode.templateIDs, templateID)

	return templateID
}

// isVariableToken checks if a token is likely a variable field (number, ID, etc.)
func (d *DrainExtractor) isVariableToken(token string) bool {
	// Numbers (integers, decimals, hex)
	if regexp.MustCompile(`^-?\d+(\.\d+)?$`).MatchString(token) {
		return true
	}
	if regexp.MustCompile(`^0[xX][0-9a-fA-F]+$`).MatchString(token) {
		return true
	}

	// IP addresses
	if regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`).MatchString(token) {
		return true
	}

	// UUIDs
	if regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(token) {
		return true
	}

	// Timestamps (ISO 8601 like)
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`).MatchString(token) {
		return true
	}

	// URLs/Paths that look like IDs
	if strings.HasPrefix(token, "/") && len(token) > 20 {
		return true
	}

	return false
}

// tokenize splits a message into tokens.
func (d *DrainExtractor) tokenize(message string) []string {
	return d.tokenRegex.FindAllString(message, -1)
}

// calculateSimilarity computes the Jaccard-like similarity between two token sequences.
// Returns a value between 0 and 1, where 1 means identical.
func (d *DrainExtractor) calculateSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	maxLen := len(tokens1)
	if len(tokens2) > maxLen {
		maxLen = len(tokens2)
	}

	matches := 0
	minLen := len(tokens1)
	if len(tokens2) < minLen {
		minLen = len(tokens2)
	}

	for i := 0; i < minLen; i++ {
		t1 := tokens1[i]
		t2 := tokens2[i]

		// Wildcards match anything
		if t1 == "<*>" || t2 == "<*>" {
			matches++
		} else if t1 == t2 {
			matches++
		}
	}

	return float64(matches) / float64(maxLen)
}

// mergeTemplates merges two token sequences into a template.
// Positions where tokens differ become wildcards.
func (d *DrainExtractor) mergeTemplates(existingTokens, newTokens []string) []string {
	maxLen := len(existingTokens)
	if len(newTokens) > maxLen {
		maxLen = len(newTokens)
	}

	result := make([]string, maxLen)
	for i := 0; i < maxLen; i++ {
		if i >= len(existingTokens) || i >= len(newTokens) {
			result[i] = "<*>"
		} else if existingTokens[i] == "<*>" || existingTokens[i] != newTokens[i] {
			result[i] = "<*>"
		} else {
			result[i] = existingTokens[i]
		}
	}

	return result
}

// createTemplateTokens creates initial template tokens from a message.
// Variable tokens are replaced with wildcards.
func (d *DrainExtractor) createTemplateTokens(tokens []string) []string {
	result := make([]string, len(tokens))
	for i, token := range tokens {
		if d.isVariableToken(token) {
			result[i] = "<*>"
		} else {
			result[i] = token
		}
	}
	return result
}

// tokensToPattern converts tokens back to a pattern string.
func (d *DrainExtractor) tokensToPattern(tokens []string) string {
	return strings.Join(tokens, " ")
}

// generateTemplateID creates a unique template ID.
func (d *DrainExtractor) generateTemplateID() string {
	return fmt.Sprintf("T_%d", len(d.templates)+1)
}

// GetTemplates returns all extracted templates sorted by frequency (descending).
func (d *DrainExtractor) GetTemplates() []*Template {
	d.mu.RLock()
	defer d.mu.RUnlock()

	templates := make([]*Template, 0, len(d.templates))
	for _, t := range d.templates {
		templates = append(templates, t)
	}

	// Sort by count descending
	for i := 0; i < len(templates)-1; i++ {
		for j := i + 1; j < len(templates); j++ {
			if templates[j].Count > templates[i].Count {
				templates[i], templates[j] = templates[j], templates[i]
			}
		}
	}

	return templates
}

// GetTemplateCount returns the number of unique templates extracted.
func (d *DrainExtractor) GetTemplateCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.templates)
}

// GetTotalLogCount returns the total number of log messages processed.
func (d *DrainExtractor) GetTotalLogCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	total := 0
	for _, t := range d.templates {
		total += t.Count
	}
	return total
}

// Reset clears all extracted templates and resets the tree.
func (d *DrainExtractor) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.root = &ParseTreeNode{
		nodeType:    RootNode,
		children:    make(map[string]*ParseTreeNode),
		templateIDs: []string{},
	}
	d.templates = make(map[string]*Template)
}

// GetTemplateByID returns a specific template by ID.
func (d *DrainExtractor) GetTemplateByID(templateID string) (*Template, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	template, ok := d.templates[templateID]
	return template, ok
}

// MatchTemplate checks if a message matches a specific template.
func (d *DrainExtractor) MatchTemplate(message, templateID string) bool {
	tokens := d.tokenize(message)

	d.mu.RLock()
	defer d.mu.RUnlock()

	template, ok := d.templates[templateID]
	if !ok {
		return false
	}

	similarity := d.calculateSimilarity(tokens, template.Tokens)
	return similarity >= d.simThreshold
}
