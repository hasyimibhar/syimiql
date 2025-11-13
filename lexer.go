package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type lexer struct {
	input  string
	pos    int
	result Stmt
}

func newLexer(s string) *lexer { return &lexer{input: s} }

func (l *lexer) Error(s string) {
	// goyacc calls this on parse errors
	fmt.Printf("parse error: %s (at pos %d)\n", s, l.pos)
}

func (l *lexer) Lex(lval *yySymType) int {
	l.skipSpaces()
	if l.eof() {
		return 0 // EOF
	}

	// Single-char tokens
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	switch r {
	case ',':
		l.pos += w
		return COMMA
	case '(':
		l.pos += w
		return LPAREN
	case ')':
		l.pos += w
		return RPAREN
	case '\'':
		// String literal
		return l.lexStringLiteral(lval)
	}

	// Number literals
	if unicode.IsDigit(r) || (r == '-' && l.peekIsDigit()) {
		return l.lexNumberLiteral(lval)
	}

	// Identifiers / keywords
	if isIdentStart(r) {
		start := l.pos
		l.pos += w
		for !l.eof() {
			r2, w2 := utf8.DecodeRuneInString(l.input[l.pos:])
			if !isIdentPart(r2) {
				break
			}
			l.pos += w2
		}
		lit := l.input[start:l.pos]
		upper := strings.ToUpper(lit)
		switch upper {
		case "SELECT":
			return SELECT
		case "FROM":
			return FROM
		case "AS":
			return AS
		case "CREATE":
			return CREATE
		case "TABLE":
			return TABLE
		case "INSERT":
			return INSERT
		case "INTO":
			return INTO
		case "VALUES":
			return VALUES
		default:
			lval.str = lit // raw identifier text
			return IDENT
		}
	}

	// Unknown char -> skip and continue (or you could error)
	l.pos += w
	return l.Lex(lval)
}

func (l *lexer) lexStringLiteral(lval *yySymType) int {
	// Assume we're at the opening '
	l.pos++ // skip opening quote
	start := l.pos
	
	for !l.eof() {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if r == '\'' {
			// Found closing quote
			lit := l.input[start:l.pos]
			l.pos += w // skip closing quote
			lval.str = lit
			return STRING_LIT
		}
		l.pos += w
	}
	
	// Unclosed string - return empty string
	lval.str = l.input[start:l.pos]
	return STRING_LIT
}

func (l *lexer) lexNumberLiteral(lval *yySymType) int {
	start := l.pos
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	
	// Optional minus sign
	if r == '-' {
		l.pos += w
	}
	
	// Read digits
	for !l.eof() {
		r2, w2 := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsDigit(r2) && r2 != '.' {
			break
		}
		l.pos += w2
	}
	
	lit := l.input[start:l.pos]
	lval.str = lit
	return NUMBER_LIT
}

func (l *lexer) peekIsDigit() bool {
	if l.pos+1 >= len(l.input) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos+1:])
	return unicode.IsDigit(r)
}

func (l *lexer) skipSpaces() {
	for !l.eof() {
		r, w := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		l.pos += w
	}
}

func (l *lexer) eof() bool { return l.pos >= len(l.input) }

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
