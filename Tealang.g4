grammar Tealang;

prog
    :   stmts? EOF
    ;

stmts
    :   stmt
    |   stmts stmt
    ;

stmt
    :   statement
    ;

statement
    :   declaration
    |   block
    |   condition
    |   expression
    |   termination
    ;

block
    :   NEWLINE? '{' stmts? expr? '}' (NEWLINE)?
    ;

declaration
    :   decl (NEWLINE|SEMICOLON)
    ;

expression
    :   expr? (NEWLINE|SEMICOLON)
    ;

condition
    :   IF expr block (ELSE block)?                 # IfStatement
    ;

termination
    :   ERR (NEWLINE|SEMICOLON)                     # TermError
    |   RET NUMBER (NEWLINE|SEMICOLON)              # TermReturn
    ;

decl
    :   LET IDENT '=' expr                          # DeclareVar
    |   CONST IDENT '=' NUMBER                      # DeclareNumberConst
    |   CONST IDENT '=' STRING                      # DeclareStringConst
    ;

expr
    :   IDENT                                       # Identifier
    |   NUMBER                                      # NumberLiteral
    |   STRING                                      # StringLiteral
    |	'(' expr ')'                                # Group
    |   funcCall                                    # FunctionCall
    |   arrayElem                                   # ArrayElement
    |   compoundElem                                # ObjElement
    |   op='!' expr                                 # Not
    |   op='~' expr                                 # BitNot
    |	expr op=('*'|'/'|'%') expr                  # MulDivMod
    |	expr op=('+'|'-') expr                      # SumSub
    |   expr op=('<'|'<='|'>'|'>='|'=='|'!=') expr  # Relation
    |   expr op=('|'|'^'|'&') expr                  # BitOp
    |   expr op=('&&'|'||') expr                    # AndOr
    |   condExpr                                    # IfExpr
    |   IDENT '=' expr                              # Assign
    ;

compoundElem
    :   IDENT '.' IDENT
    |   arrayElem '.' IDENT
    ;

arrayElem
    :   IDENT '[' NUMBER ']'
    ;

funcCall
    :   IDENT '(' expr ')'
    ;

// named rules for tree-walking only
condExpr
    : IF condIfExpr '{' condTrueExpr '}' ELSE '{' condFalseExpr '}'
    ;

condTrueExpr
    : expr                                          # IfExprTrue
    ;

condFalseExpr
    : expr                                          # IfExprFalse
    ;

condIfExpr
    : expr                                          # IfExprCond
    ;

LET         : 'let' ;
CONST       : 'const' ;
ERR         : 'error' ;
RET         : 'return' ;
IF          : 'if' ;
ELSE        : 'else' ;

STRING      : EncodingPrefix? '"' StringChar* '"' ;
NUMBER      : [0-9]+ ;
IDENT       : [a-zA-Z_]+[a-zA-Z0-9_]* ;
NEWLINE     : [\r\n]+ ;
SEMICOLON   : ';' ;
WHITESPACE  : (' ' | '\t')+ -> skip ;
COMMENT     : '//' ~[\r\n]* -> skip ;

fragment EncodingPrefix
    :   'b32'
    |   'b64'
    ;

fragment StringChar
    :   ~["\\\r\n]
    |   HexEscapeSeq
    ;

fragment HexEscapeSeq
    : '\\x' [0-9a-fA-F]+
    ;
