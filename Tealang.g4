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
    :   NEWLINE? '{' stmts? expr? '}' (NEWLINE|EOF)?
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
    :   IDENT '=' expr                              # Assign
    |	expr ('*'|'/') expr                         # MulDiv
    |	expr ('+'|'-') expr                         # SumSub
    |   expr ('%') expr                             # Mod
    |   expr ('<'|'<='|'>'|'>='|'=='|'!=') expr     # Relation
    |   expr ('&&'|'||') expr                       # AndOr
    |   '!' expr                                    # Not
    |   expr ('|'|'&'|'^') expr                     # BitOp
    |   '~' expr                                    # BitNot
    |	'(' expr ')'                                # Group
    |   arrayElem                                   # ArrayElement
    |   compoundElem                                # ObjElement
    |   funcCall                                    # FunctionCall
    |   condExpr                                    # IfExpr
    |   NUMBER                                      # NumberLiteral
    |   STRING                                      # StringLiteral
    |   IDENT                                       # Identifier
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

condExpr
    : IF expr '{' expr '}' ELSE '{' expr '}'
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
