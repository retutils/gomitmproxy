package httpql

import (
	"strings"
	"text/scanner"
)

type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_IDENT
	TOKEN_STRING
	TOKEN_INT
	TOKEN_DOT
	TOKEN_COLON
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
)

type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}

type Lexer struct {
	s   scanner.Scanner
	buf []Token // Lookahead buffer if needed
}

func NewLexer(input string) *Lexer {
	var s scanner.Scanner
	s.Init(strings.NewReader(input))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts
	s.Whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' '
	return &Lexer{s: s}
}

func (l *Lexer) NextToken() Token {
	tok := l.s.Scan()
	lit := l.s.TokenText()
	
	switch tok {
	case scanner.EOF:
		return Token{Type: TOKEN_EOF}
	case scanner.Ident:
		lit = strings.ToLower(lit)
		if lit == "and" {
			return Token{Type: TOKEN_AND, Literal: lit}
		} else if lit == "or" {
			return Token{Type: TOKEN_OR, Literal: lit}
		} else if lit == "not" {
			return Token{Type: TOKEN_NOT, Literal: lit}
		}
		return Token{Type: TOKEN_IDENT, Literal: lit}
	case scanner.String:
		// Remove quotes
		return Token{Type: TOKEN_STRING, Literal: strings.Trim(lit, "\"")}
	case scanner.Int:
		return Token{Type: TOKEN_INT, Literal: lit}
	case '.':
		return Token{Type: TOKEN_DOT, Literal: "."}
	case ':':
		return Token{Type: TOKEN_COLON, Literal: ":"}
	case '(':
		return Token{Type: TOKEN_LPAREN, Literal: "("}
	case ')':
		return Token{Type: TOKEN_RPAREN, Literal: ")"}
	default:
		// Handle other single chars or error?
		return Token{Type: TOKEN_IDENT, Literal: lit}
	}
}
