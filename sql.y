%{
package main

// SelectExpr represents a column or literal in the select list
type SelectExpr struct {
	Expr  string // column name or literal value
	Alias string // optional alias (AS name)
	IsLiteral bool // true if this is a literal, false if column reference
}

// SelectStmt is the tiny AST we build.
type SelectStmt struct {
	Columns []SelectExpr
	Table   string
}

// ColumnDef represents a column definition in CREATE TABLE
type ColumnDef struct {
	Name string
	Type string
}

// CreateTableStmt represents a CREATE TABLE statement
type CreateTableStmt struct {
	Table   string
	Columns []ColumnDef
}

// InsertStmt represents an INSERT INTO statement
type InsertStmt struct {
	Table  string
	Values [][]string // list of value tuples
}

// Stmt is a union type for all statement types
type Stmt interface {
	isStmt()
}

func (s *SelectStmt) isStmt() {}
func (s *CreateTableStmt) isStmt() {}
func (s *InsertStmt) isStmt() {}
%}

%union{
    str       string
    expr      *SelectExpr
    exprs     []SelectExpr
    colDef    *ColumnDef
    colDefs   []ColumnDef
    values    []string
    valueList [][]string
}

%token SELECT FROM COMMA IDENT AS STRING_LIT NUMBER_LIT
%token CREATE TABLE INSERT INTO VALUES LPAREN RPAREN
%token <str> IDENT STRING_LIT NUMBER_LIT

%type  <str>       ident data_type
%type  <expr>      select_expr
%type  <exprs>     select_list
%type  <colDef>    column_def
%type  <colDefs>   column_def_list
%type  <values>    value_tuple literal_list
%type  <valueList> value_list

%%

stmt
    : SELECT select_list FROM ident
    {
        // Save the parse result on the lexer for retrieval in main.go
        yylex.(*lexer).result = &SelectStmt{Columns: $2, Table: $4}
    }
    | SELECT select_list
    {
        yylex.(*lexer).result = &SelectStmt{Columns: $2, Table: ""}
    }
    | CREATE TABLE ident LPAREN column_def_list RPAREN
    {
        yylex.(*lexer).result = &CreateTableStmt{Table: $3, Columns: $5}
    }
    | INSERT INTO ident VALUES value_list
    {
        yylex.(*lexer).result = &InsertStmt{Table: $3, Values: $5}
    }
    ;

select_list
    : select_expr                    { $$ = []SelectExpr{*$1} }
    | select_list COMMA select_expr  { $$ = append($1, *$3) }
    ;

select_expr
    : ident                { $$ = &SelectExpr{Expr: $1, IsLiteral: false} }
    | ident AS ident       { $$ = &SelectExpr{Expr: $1, Alias: $3, IsLiteral: false} }
    | STRING_LIT           { $$ = &SelectExpr{Expr: $1, IsLiteral: true} }
    | STRING_LIT AS ident  { $$ = &SelectExpr{Expr: $1, Alias: $3, IsLiteral: true} }
    | NUMBER_LIT           { $$ = &SelectExpr{Expr: $1, IsLiteral: true} }
    | NUMBER_LIT AS ident  { $$ = &SelectExpr{Expr: $1, Alias: $3, IsLiteral: true} }
    ;

column_def_list
    : column_def                         { $$ = []ColumnDef{*$1} }
    | column_def_list COMMA column_def   { $$ = append($1, *$3) }
    ;

column_def
    : ident data_type  { $$ = &ColumnDef{Name: $1, Type: $2} }
    ;

data_type
    : IDENT { $$ = $1 }
    ;

value_list
    : value_tuple                    { $$ = [][]string{$1} }
    | value_list COMMA value_tuple   { $$ = append($1, $3) }
    ;

value_tuple
    : LPAREN literal_list RPAREN  { $$ = $2 }
    ;

literal_list
    : STRING_LIT                     { $$ = []string{$1} }
    | NUMBER_LIT                     { $$ = []string{$1} }
    | literal_list COMMA STRING_LIT  { $$ = append($1, $3) }
    | literal_list COMMA NUMBER_LIT  { $$ = append($1, $3) }
    ;

ident
    : IDENT { $$ = $1 }
    ;

%%