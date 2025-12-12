package main

import (
	"fmt"
)

func parse(input string) (*SelectStmt, error) {
	l := newLexer(input)
	yyParse(l)
	if l.result == nil {
		return nil, fmt.Errorf("no result (input may be invalid)")
	}
	if stmt, ok := l.result.(*SelectStmt); ok {
		return stmt, nil
	}
	return nil, fmt.Errorf("expected SELECT statement")
}

func parseStmt(input string) (Stmt, error) {
	l := newLexer(input)
	yyParse(l)
	if l.result == nil {
		return nil, fmt.Errorf("no result (input may be invalid)")
	}
	return l.result, nil
}

func main() {
	server, err := NewPgServer(":5432")
	if err != nil {
		panic(err)
	}
	fmt.Printf("PostgreSQL wire protocol server listening on %s\n", server.Addr())
	fmt.Println("Connect with: psql -h localhost -p 5432 -U test")
	if err := server.Serve(); err != nil {
		panic(err)
	}
}
