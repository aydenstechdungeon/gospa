package sfc

import (
	"fmt"
	"strings"
	"unicode"
)

// TemplateParser parses a GoSPA template string into an AST.
type TemplateParser struct {
	input      string
	pos        int
	baseLine   int
	baseColumn int
	baseOffset int
}

// NewTemplateParser creates a new TemplateParser.
func NewTemplateParser(input string, offset int, line, col int) *TemplateParser {
	return &TemplateParser{
		input:      input,
		baseOffset: offset,
		baseLine:   line,
		baseColumn: col,
	}
}

// Parse returns the root list of nodes.
func (p *TemplateParser) Parse() ([]Node, error) {
	return p.parseNodes("")
}

func (p *TemplateParser) parseNodes(closingTag string) ([]Node, error) {
	var nodes []Node
	for p.pos < len(p.input) {
		if closingTag != "" && strings.HasPrefix(p.input[p.pos:], closingTag) {
			break
		}

		switch char := p.input[p.pos]; char {
		case '<':
			if strings.HasPrefix(p.input[p.pos:], "</") {
				// Unexpected end tag or handled by caller
				if closingTag == "" {
					return nil, p.error("unexpected end tag")
				}
				return nodes, nil // break loop and return what we have
			}
			node, err := p.parseTag()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case '{':
			if strings.HasPrefix(p.input[p.pos:], "{/") || strings.HasPrefix(p.input[p.pos:], "{:") {
				return nodes, nil
			}
			node, err := p.parseCurly()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case '@':
			// Handle @Component(...) syntax (templ-style component calls).
			// If this is plain text (e.g., email/user handle), keep it as text.
			if p.canParseAtComponentCall() {
				node, err := p.parseAtComponent()
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, node)
			} else {
				nodes = append(nodes, p.parseText())
			}
		default:
			nodes = append(nodes, p.parseText())
		}
	}
	return nodes, nil
}

// parseAtComponent parses @Component(...) syntax (templ-style component calls).
func (p *TemplateParser) parseAtComponent() (Node, error) {
	start := p.pos
	p.pos++ // skip @

	name := p.parseIdentifierWithDots()
	if name == "" {
		return nil, p.error("expected component name after @")
	}
	p.skipWhitespace()
	attrs, err := p.parseParenArgs(true)
	if err != nil {
		return nil, err
	}

	node := &ComponentNode{
		Name:       name,
		Attributes: attrs,
	}
	p.setPos(&node.BaseNode, start, p.pos)
	return node, nil
}

// parseIdentifierWithDots parses an identifier that may contain dots (e.g., components.CodeBlock).
func (p *TemplateParser) parseIdentifierWithDots() string {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_' || p.input[p.pos] == '.' || p.input[p.pos] == ':') {
		p.pos++
	}
	return p.input[start:p.pos]
}

// parseParenArgs parses parenthesized arguments like (arg1, "arg2", `arg3`).
// Returns attributes where each positional arg gets a generated name.
func (p *TemplateParser) parseParenArgs(required bool) ([]Attribute, error) {
	if !p.consume("(") {
		if required {
			return nil, p.error("expected (")
		}
		return nil, nil
	}

	var attrs []Attribute
	argIndex := 0

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] == ')' {
			break
		}

		// Handle comma separator
		if argIndex > 0 {
			if !p.consume(",") {
				break
			}
			p.skipWhitespace()
		}

		// Parse the argument value
		var attr Attribute
		attr.Name = fmt.Sprintf("_arg%d", argIndex)

		switch {
		case p.pos < len(p.input) && p.input[p.pos] == '`':
			// Backtick string
			p.pos++ // skip opening `
			start := p.pos
			for p.pos < len(p.input) && p.input[p.pos] != '`' {
				p.pos++
			}
			if p.pos >= len(p.input) {
				return nil, p.error("unterminated backtick string argument")
			}
			attr.Value = p.input[start:p.pos]
			p.pos++ // skip closing `
			attr.IsExpression = true // Mark as expression to preserve raw value
		case p.pos < len(p.input) && p.input[p.pos] == '"':
			// Double-quoted string
			p.pos++ // skip opening "
			attr.Value = p.consumeEscapedUntil("\"")
			if p.pos >= len(p.input) {
				return nil, p.error("unterminated quoted string argument")
			}
			p.pos++ // skip closing "
		case p.pos < len(p.input) && p.input[p.pos] == '{':
			// Expression in braces
			p.pos++ // skip {
			attr.Value = p.consumeUntil("}")
			if p.pos >= len(p.input) {
				return nil, p.error("unterminated expression argument")
			}
			attr.IsExpression = true
			p.pos++ // skip }
		default:
			// Unquoted value (identifier, number, etc.)
			start := p.pos
			for p.pos < len(p.input) && p.input[p.pos] != ',' && p.input[p.pos] != ')' && !unicode.IsSpace(rune(p.input[p.pos])) {
				p.pos++
			}
			attr.Value = p.input[start:p.pos]
		}

		attrs = append(attrs, attr)
		argIndex++
	}

	if !p.consume(")") {
		return nil, p.error("expected )")
	}
	return attrs, nil
}

func (p *TemplateParser) parseTag() (Node, error) {
	start := p.pos
	p.pos++ // skip <

	if p.pos < len(p.input) && p.input[p.pos] == '@' {
		// Component @Component
		p.pos++
		name := p.parseIdentifier()
		attrs := p.parseAttributes()

		var children []Node
		selfClosing := false
		if p.consume("/") {
			selfClosing = true
		}
		if !p.consume(">") {
			return nil, p.error("expected >")
		}

		if !selfClosing {
			children, _ = p.parseNodes("")
			// GoSPA currently uses @Name(...) { children } style in Templ
			// But in SFC template it's <@Component /> or <@Component>...</@Component>
			// Wait, GoSPA compiler.go:219 uses PascalCase components <Component>
			// and @snippet(args) for snippet calls.
			// Let's stick to what compiler.go expects.
		}

		node := &ComponentNode{
			Name:       name,
			Attributes: attrs,
			Children:   children,
		}
		p.setPos(&node.BaseNode, start, p.pos)
		return node, nil
	}

	tagName := p.parseIdentifier()
	attrs := p.parseAttributes()

	selfClosing := p.consume("/")
	if !p.consume(">") {
		return nil, p.error("expected >")
	}

	node := &ElementNode{
		TagName:     tagName,
		Attributes:  attrs,
		SelfClosing: selfClosing,
	}

	if !selfClosing && !isVoidTag(tagName) {
		children, err := p.parseNodes("</" + tagName + ">")
		if err != nil {
			return nil, err
		}
		node.Children = children
		if !p.consume("</" + tagName + ">") {
			return nil, p.error("expected </" + tagName + ">")
		}
	}

	p.setPos(&node.BaseNode, start, p.pos)
	return node, nil
}

func (p *TemplateParser) parseCurly() (Node, error) {
	start := p.pos
	p.pos++ // skip {

	if p.consume("#") {
		keyword := p.parseIdentifier()
		switch keyword {
		case "if":
			p.skipWhitespace()
			cond := p.consumeUntil("}")
			if !p.consume("}") {
				return nil, p.error("expected }")
			}
			then, err := p.parseNodes("") // stops at {/if} or {:else}
			if err != nil {
				return nil, err
			}

			node := &IfNode{
				Condition: cond,
				Then:      then,
			}

			for p.consume("{:else") {
				if p.consume(" if") {
					p.skipWhitespace()
					elseIfCond := p.consumeUntil("}")
					if !p.consume("}") {
						return nil, p.error("expected }")
					}
					elseIfThen, err := p.parseNodes("")
					if err != nil {
						return nil, err
					}
					node.ElseIfs = append(node.ElseIfs, ElseIfNode{
						Condition: elseIfCond,
						Then:      elseIfThen,
					})
				} else {
					if !p.consume("}") {
						return nil, p.error("expected }")
					}
					elseThen, err := p.parseNodes("")
					if err != nil {
						return nil, err
					}
					node.Else = elseThen
					break
				}
			}

			if !p.consume("{/if}") {
				return nil, p.error("expected {/if}")
			}
			p.setPos(&node.BaseNode, start, p.pos)
			return node, nil

		case "each":
			p.skipWhitespace()
			iteratee := p.consumeUntil(" as ")
			if !p.consume(" as ") {
				return nil, p.error("expected 'as' in each block")
			}
			as := p.consumeUntil("}")
			if !p.consume("}") {
				return nil, p.error("expected }")
			}
			children, err := p.parseNodes("{/each}")
			if err != nil {
				return nil, err
			}
			if !p.consume("{/each}") {
				return nil, p.error("expected {/each}")
			}
			node := &EachNode{
				Iteratee: iteratee,
				As:       as,
				Children: children,
			}
			p.setPos(&node.BaseNode, start, p.pos)
			return node, nil

		case "snippet":
			p.skipWhitespace()
			name := p.parseIdentifier()
			p.consume("(")
			args := p.consumeUntil(")")
			p.consume(")")
			p.skipWhitespace()
			if !p.consume("}") {
				return nil, p.error("expected }")
			}
			children, err := p.parseNodes("{/snippet}")
			if err != nil {
				return nil, err
			}
			if !p.consume("{/snippet}") {
				return nil, p.error("expected {/snippet}")
			}
			node := &SnippetNode{
				Name:     name,
				Args:     args,
				Children: children,
			}
			p.setPos(&node.BaseNode, start, p.pos)
			return node, nil
		}
	}

	// Expression {expr}
	content := p.consumeUntil("}")
	if !p.consume("}") {
		return nil, p.error("expected }")
	}
	node := &ExpressionNode{Content: content}
	p.setPos(&node.BaseNode, start, p.pos)
	return node, nil
}

func (p *TemplateParser) parseText() Node {
	start := p.pos
	var sb strings.Builder
	for p.pos < len(p.input) {
		if p.input[p.pos] == '<' || p.input[p.pos] == '{' || (p.input[p.pos] == '@' && p.canParseAtComponentCall()) {
			break
		}
		sb.WriteByte(p.input[p.pos])
		p.pos++
	}
	node := &TextNode{Content: sb.String()}
	p.setPos(&node.BaseNode, start, p.pos)
	return node
}

func (p *TemplateParser) parseAttributes() []Attribute {
	var attrs []Attribute
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] == '>' || p.input[p.pos] == '/' {
			break
		}

		name := p.parseIdentifier()
		if name == "" {
			break
		}

		p.skipWhitespace()
		if p.consume("=") {
			p.skipWhitespace()
			switch {
			case p.consume("{"):
				val := p.consumeUntil("}")
				p.consume("}")
				attrs = append(attrs, Attribute{Name: name, Value: val, IsExpression: true})
			case p.consume("\""):
				val := p.consumeEscapedUntil("\"")
				p.consume("\"")
				attrs = append(attrs, Attribute{Name: name, Value: val})
			case p.consume("'"):
				val := p.consumeEscapedUntil("'")
				p.consume("'")
				attrs = append(attrs, Attribute{Name: name, Value: val})
			default:
				// unquoted val
				start := p.pos
				for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && p.input[p.pos] != '>' && p.input[p.pos] != '/' {
					p.pos++
				}
				val := p.input[start:p.pos]
				attrs = append(attrs, Attribute{Name: name, Value: val})
			}
		} else {
			attrs = append(attrs, Attribute{Name: name})
		}
	}
	return attrs
}

func (p *TemplateParser) parseIdentifier() string {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_' || p.input[p.pos] == ':') {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *TemplateParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *TemplateParser) consume(s string) bool {
	if strings.HasPrefix(p.input[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *TemplateParser) consumeUntil(s string) string {
	start := p.pos
	idx := strings.Index(p.input[p.pos:], s)
	if idx == -1 {
		p.pos = len(p.input)
		return p.input[start:]
	}
	p.pos += idx
	return p.input[start:p.pos]
}

// consumeEscapedUntil consumes characters until the delimiter is found,
// handling backslash-escaped delimiters (e.g., \" inside a double-quoted string).
// The escape sequences are preserved in the output.
func (p *TemplateParser) consumeEscapedUntil(delimiter string) string {
	var sb strings.Builder
	for p.pos < len(p.input) {
		if p.input[p.pos] == '\\' && p.pos+1 < len(p.input) {
			// Preserve the escape sequence (keep both backslash and char)
			sb.WriteByte(p.input[p.pos])
			sb.WriteByte(p.input[p.pos+1])
			p.pos += 2
			continue
		}
		if strings.HasPrefix(p.input[p.pos:], delimiter) {
			break
		}
		sb.WriteByte(p.input[p.pos])
		p.pos++
	}
	return sb.String()
}

func (p *TemplateParser) setPos(base *BaseNode, start, end int) {
	base.StartLine, base.StartColumn = OffsetToPosition(p.input, start)
	base.EndLine, base.EndColumn = OffsetToPosition(p.input, end)
	// Add base offsets from SFC block
	base.StartLine += p.baseLine
	base.EndLine += p.baseLine
	if base.StartLine == p.baseLine {
		base.StartColumn += p.baseColumn
	}
	if base.EndLine == p.baseLine {
		base.EndColumn += p.baseColumn
	}
}

func (p *TemplateParser) error(msg string) error {
	line, col := OffsetToPosition(p.input, p.pos)
	return fmt.Errorf("at %d:%d: %s", line+p.baseLine, col+p.baseColumn, msg)
}

func isVoidTag(tag string) bool {
	switch strings.ToLower(tag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

func isIdentifierStart(ch byte) bool {
	return ch == '_' || unicode.IsLetter(rune(ch))
}

func isIdentifierPart(ch byte) bool {
	return ch == '-' || ch == '_' || ch == '.' || ch == ':' || unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch))
}

func (p *TemplateParser) canParseAtComponentCall() bool {
	if p.pos >= len(p.input) || p.input[p.pos] != '@' {
		return false
	}
	i := p.pos + 1
	if i >= len(p.input) || !isIdentifierStart(p.input[i]) {
		return false
	}
	for i < len(p.input) && isIdentifierPart(p.input[i]) {
		i++
	}
	for i < len(p.input) && unicode.IsSpace(rune(p.input[i])) {
		i++
	}
	return i < len(p.input) && p.input[i] == '('
}
