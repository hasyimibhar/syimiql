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
	// Test basic column selection
	stmt, serr := parse("SELECT foo, bar FROM zoom")
	if serr != nil {
		panic(serr)
	}

	fmt.Println("=== Test 1: Basic SELECT ===")
	fmt.Println("Parsed successfully!")
	for i, col := range stmt.Columns {
		fmt.Printf("Column %d: %s (IsLiteral: %v, Alias: %s)\n", i+1, col.Expr, col.IsLiteral, col.Alias)
	}
	fmt.Printf("Table:   %s\n\n", stmt.Table)

	// Test with AS aliases
	stmt2, serr2 := parse("SELECT foo AS f, bar AS b FROM zoom")
	if serr2 != nil {
		panic(serr2)
	}

	fmt.Println("=== Test 2: SELECT with AS aliases ===")
	fmt.Println("Parsed successfully!")
	for i, col := range stmt2.Columns {
		fmt.Printf("Column %d: %s AS %s (IsLiteral: %v)\n", i+1, col.Expr, col.Alias, col.IsLiteral)
	}
	fmt.Printf("Table:   %s\n\n", stmt2.Table)

	// Test with string literals
	stmt3, serr3 := parse("SELECT 'hello' AS greeting, foo, 'world' FROM zoom")
	if serr3 != nil {
		panic(serr3)
	}

	fmt.Println("=== Test 3: SELECT with string literals ===")
	fmt.Println("Parsed successfully!")
	for i, col := range stmt3.Columns {
		if col.Alias != "" {
			fmt.Printf("Column %d: '%s' AS %s (IsLiteral: %v)\n", i+1, col.Expr, col.Alias, col.IsLiteral)
		} else {
			fmt.Printf("Column %d: '%s' (IsLiteral: %v)\n", i+1, col.Expr, col.IsLiteral)
		}
	}
	fmt.Printf("Table:   %s\n\n", stmt3.Table)

	// Test with number literals
	stmt4, serr4 := parse("SELECT 42, foo AS f, 3.14 AS pi FROM zoom")
	if serr4 != nil {
		panic(serr4)
	}

	fmt.Println("=== Test 4: SELECT with number literals ===")
	fmt.Println("Parsed successfully!")
	for i, col := range stmt4.Columns {
		if col.Alias != "" {
			fmt.Printf("Column %d: %s AS %s (IsLiteral: %v)\n", i+1, col.Expr, col.Alias, col.IsLiteral)
		} else {
			fmt.Printf("Column %d: %s (IsLiteral: %v)\n", i+1, col.Expr, col.IsLiteral)
		}
	}
	fmt.Printf("Table:   %s\n\n", stmt4.Table)

	// Test CREATE TABLE
	createStmt, cerr := parseStmt("CREATE TABLE zoom (foo STRING)")
	if cerr != nil {
		panic(cerr)
	}

	fmt.Println("=== Test 5: CREATE TABLE ===")
	fmt.Println("Parsed successfully!")
	if ct, ok := createStmt.(*CreateTableStmt); ok {
		fmt.Printf("Table: %s\n", ct.Table)
		for i, col := range ct.Columns {
			fmt.Printf("Column %d: %s %s\n", i+1, col.Name, col.Type)
		}
	}
	fmt.Println()

	// Test CREATE TABLE with multiple columns
	createStmt2, cerr2 := parseStmt("CREATE TABLE users (id NUMBER, name STRING, age NUMBER)")
	if cerr2 != nil {
		panic(cerr2)
	}

	fmt.Println("=== Test 6: CREATE TABLE (multiple columns) ===")
	fmt.Println("Parsed successfully!")
	if ct, ok := createStmt2.(*CreateTableStmt); ok {
		fmt.Printf("Table: %s\n", ct.Table)
		for i, col := range ct.Columns {
			fmt.Printf("Column %d: %s %s\n", i+1, col.Name, col.Type)
		}
	}
	fmt.Println()

	// Test INSERT INTO with single row
	insertStmt, ierr := parseStmt("INSERT INTO zoom VALUES ('haha')")
	if ierr != nil {
		panic(ierr)
	}

	fmt.Println("=== Test 7: INSERT INTO (single row) ===")
	fmt.Println("Parsed successfully!")
	if ins, ok := insertStmt.(*InsertStmt); ok {
		fmt.Printf("Table: %s\n", ins.Table)
		for i, row := range ins.Values {
			fmt.Printf("Row %d: %v\n", i+1, row)
		}
	}
	fmt.Println()

	// Test INSERT INTO with multiple rows
	insertStmt2, ierr2 := parseStmt("INSERT INTO zoom VALUES ('haha'), ('dope')")
	if ierr2 != nil {
		panic(ierr2)
	}

	fmt.Println("=== Test 8: INSERT INTO (multiple rows) ===")
	fmt.Println("Parsed successfully!")
	if ins, ok := insertStmt2.(*InsertStmt); ok {
		fmt.Printf("Table: %s\n", ins.Table)
		for i, row := range ins.Values {
			fmt.Printf("Row %d: %v\n", i+1, row)
		}
	}
	fmt.Println()

	// Test INSERT INTO with multiple columns
	insertStmt3, ierr3 := parseStmt("INSERT INTO users VALUES (1, 'Alice', 30), (2, 'Bob', 25)")
	if ierr3 != nil {
		panic(ierr3)
	}

	fmt.Println("=== Test 9: INSERT INTO (multiple columns & rows) ===")
	fmt.Println("Parsed successfully!")
	if ins, ok := insertStmt3.(*InsertStmt); ok {
		fmt.Printf("Table: %s\n", ins.Table)
		for i, row := range ins.Values {
			fmt.Printf("Row %d: %v\n", i+1, row)
		}
	}
	fmt.Println()

	db, err := Create("foobar")
	if err != nil {
		panic(err)
	}

	if err := db.Exec("CREATE TABLE zoom (foo STRING)"); err != nil {
		panic(err)
	}

	if err := db.Exec("INSERT INTO zoom VALUES ('haha'), ('dope')"); err != nil {
		panic(err)
	}

	if result, err := db.Query("SELECT foo FROM zoom"); err != nil {
		panic(err)
	} else {
		fmt.Println(result)
	}
}
