package query

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenField
	TokenOperator
	TokenValue
	TokenAnd
	TokenOr
	TokenNot
	TokenLParen
	TokenRParen
	TokenExists
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer tokenizes a query expression
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		tokens: nil,
	}
}

// Tokenize converts the input string into a slice of tokens
func (l *Lexer) Tokenize() ([]Token, error) {
	l.tokens = nil
	l.pos = 0

	for l.pos < len(l.input) {
		// Skip whitespace
		if unicode.IsSpace(rune(l.input[l.pos])) {
			l.pos++
			continue
		}

		// Check for parentheses
		if l.input[l.pos] == '(' {
			l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "(", Pos: l.pos})
			l.pos++
			continue
		}
		if l.input[l.pos] == ')' {
			l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")", Pos: l.pos})
			l.pos++
			continue
		}

		// Check for keywords (AND, OR, NOT, exists)
		startPos := l.pos
		keyword := l.readKeyword()
		if keyword != "" {
			upper := strings.ToUpper(keyword)
			switch upper {
			case "AND", "&&":
				l.tokens = append(l.tokens, Token{Type: TokenAnd, Value: keyword, Pos: startPos})
				continue
			case "OR", "||":
				l.tokens = append(l.tokens, Token{Type: TokenOr, Value: keyword, Pos: startPos})
				continue
			case "NOT", "!":
				l.tokens = append(l.tokens, Token{Type: TokenNot, Value: keyword, Pos: startPos})
				continue
			case "EXISTS":
				// Check if this is really an exists function
				tempPos := l.pos
				for tempPos < len(l.input) && unicode.IsSpace(rune(l.input[tempPos])) {
					tempPos++
				}
				if tempPos < len(l.input) && l.input[tempPos] == '(' {
					l.tokens = append(l.tokens, Token{Type: TokenExists, Value: keyword, Pos: startPos})
					// Read the (field) part
					l.pos = tempPos // skip to (
					if err := l.readExistsArg(); err != nil {
						return nil, err
					}
					continue
				}
				// Not a function, rewind and treat as condition
				l.pos = startPos
			default:
				// It's not a keyword, it's a field - rewind and parse as a condition
				l.pos = startPos
			}
		}

		// Read a condition (field op value)
		if err := l.readCondition(); err != nil {
			return nil, err
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Value: "", Pos: l.pos})
	return l.tokens, nil
}

// readKeyword tries to read a keyword (AND, OR, NOT, EXISTS)
// Returns the keyword if found, or the identifier if not a keyword
func (l *Lexer) readKeyword() string {
	startPos := l.pos

	// Check for symbolic operators first
	if l.pos < len(l.input) {
		if strings.HasPrefix(l.input[l.pos:], "&&") {
			l.pos += 2
			return "&&"
		}
		if strings.HasPrefix(l.input[l.pos:], "||") {
			l.pos += 2
			return "||"
		}
		// Single ! followed by space or ( is a NOT
		if l.input[l.pos] == '!' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			if next == '(' || unicode.IsSpace(rune(next)) {
				l.pos++
				return "!"
			}
		}
	}

	// Read alphanumeric identifier
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			l.pos++
		} else {
			break
		}
	}

	word := l.input[startPos:l.pos]
	upper := strings.ToUpper(word)

	// Check if it's a known keyword (must be followed by space or ( for AND/OR/NOT, or ( for EXISTS)
	switch upper {
	case "AND", "OR", "NOT":
		// These keywords must be followed by whitespace or (
		if l.pos >= len(l.input) || unicode.IsSpace(rune(l.input[l.pos])) || l.input[l.pos] == '(' {
			return word
		}
	case "EXISTS":
		// EXISTS must be followed by (
		// Skip whitespace first
		tempPos := l.pos
		for tempPos < len(l.input) && unicode.IsSpace(rune(l.input[tempPos])) {
			tempPos++
		}
		if tempPos < len(l.input) && l.input[tempPos] == '(' {
			return word
		}
	}

	return word
}

// readCondition reads a condition like "field>=value" or "field >= value"
func (l *Lexer) readCondition() error {
	startPos := l.pos

	// Read field name (can include alphanumeric, _, -, .)
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			l.pos++
		} else {
			break
		}
	}

	if l.pos == startPos {
		return fmt.Errorf("expected field name at position %d", l.pos)
	}

	field := l.input[startPos:l.pos]
	l.tokens = append(l.tokens, Token{Type: TokenField, Value: field, Pos: startPos})

	// Skip whitespace before operator
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	// Read operator
	opStart := l.pos
	op := l.readOperator()
	if op == "" {
		return fmt.Errorf("expected operator after field '%s' at position %d", field, l.pos)
	}
	l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: op, Pos: opStart})

	// Read value (can be quoted or unquoted)
	value, err := l.readValue()
	if err != nil {
		return err
	}
	l.tokens = append(l.tokens, Token{Type: TokenValue, Value: value, Pos: l.pos - len(value)})

	return nil
}

// readOperator reads an operator (symbol or keyword like CONTAINS)
func (l *Lexer) readOperator() string {
	// Skip whitespace before operator
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	// Check for keyword operators (CONTAINS, LIKE, etc)
	startPos := l.pos
	keyword := ""
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if unicode.IsLetter(ch) {
			l.pos++
		} else {
			break
		}
	}

	if l.pos > startPos {
		keyword = l.input[startPos:l.pos]
		upper := strings.ToUpper(keyword)
		switch upper {
		case "CONTAINS":
			// Skip whitespace after keyword
			for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
				l.pos++
			}
			return "~=" // Map to HL contains operator
		case "LIKE":
			for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
				l.pos++
			}
			return "like"
		default:
			// Not a keyword operator, rewind
			l.pos = startPos
		}
	}

	// Check for symbol operators (order matters: check longer operators first)
	operators := []string{"!~=", "~=", "!=", ">=", "<=", ">", "<", "="}

	for _, op := range operators {
		if strings.HasPrefix(l.input[l.pos:], op) {
			l.pos += len(op)
			return op
		}
	}

	return ""
}

// readValue reads a value (quoted or unquoted)
func (l *Lexer) readValue() (string, error) {
	// Skip whitespace before value
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	if l.pos >= len(l.input) {
		return "", fmt.Errorf("expected value at position %d", l.pos)
	}

	// Check for quoted string
	if l.input[l.pos] == '"' || l.input[l.pos] == '\'' {
		return l.readQuotedString()
	}

	// Read unquoted value (until whitespace or end)
	startPos := l.pos
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		// Stop at whitespace, parentheses, or logical operators
		if unicode.IsSpace(ch) || ch == '(' || ch == ')' {
			break
		}
		// Check for && or ||
		if strings.HasPrefix(l.input[l.pos:], "&&") || strings.HasPrefix(l.input[l.pos:], "||") {
			break
		}
		l.pos++
	}

	if l.pos == startPos {
		return "", fmt.Errorf("expected value at position %d", l.pos)
	}

	return l.input[startPos:l.pos], nil
}

// readQuotedString reads a quoted string
func (l *Lexer) readQuotedString() (string, error) {
	quote := l.input[l.pos]
	l.pos++ // Skip opening quote

	startPos := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == byte(quote) {
			value := l.input[startPos:l.pos]
			l.pos++ // Skip closing quote
			return value, nil
		}
		// Handle escape sequences
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos += 2
			continue
		}
		l.pos++
	}

	return "", fmt.Errorf("unterminated quoted string starting at position %d", startPos-1)
}

// readExistsArg reads the (field) part of exists(field)
func (l *Lexer) readExistsArg() error {
	// Expect (
	if l.pos >= len(l.input) || l.input[l.pos] != '(' {
		return fmt.Errorf("expected '(' after 'exists' at position %d", l.pos)
	}
	l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "(", Pos: l.pos})
	l.pos++

	// Skip whitespace
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	// Read field name
	fieldStart := l.pos
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			l.pos++
		} else {
			break
		}
	}

	if l.pos == fieldStart {
		return fmt.Errorf("expected field name in exists() at position %d", l.pos)
	}

	field := l.input[fieldStart:l.pos]
	l.tokens = append(l.tokens, Token{Type: TokenField, Value: field, Pos: fieldStart})

	// Skip whitespace
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	// Expect )
	if l.pos >= len(l.input) || l.input[l.pos] != ')' {
		return fmt.Errorf("expected ')' in exists() at position %d", l.pos)
	}
	l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")", Pos: l.pos})
	l.pos++

	return nil
}
