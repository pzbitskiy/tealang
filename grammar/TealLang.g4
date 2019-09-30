grammar TealLang;

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
    :   IF expr block (ELSE block)?
    ;

termination
    :   ERR (NEWLINE|SEMICOLON)
    |   RET NUMBER (NEWLINE|SEMICOLON)
    ;

decl
    :   LET IDENT '=' expr
    |   CONST IDENT '=' NUMBER
    |   CONST IDENT '=' STRING
    ;

expr
    :   IDENT '=' expr
    |	expr ('*'|'/') expr
    |	expr ('+'|'-') expr
    |   expr ('%') expr
    |   expr ('<'|'<='|'>'|'>='|'=='|'!=') expr
    |   expr ('&&'|'||') expr
    |   '!' expr
    |   expr ('|'|'&'|'^') expr
    |   '~' expr
    |	'(' expr ')'
    |   arrayElem
    |   compoundElem
    |   NUMBER
    |   STRING
    |   IDENT
//    |   IF expr '{' expr '}' ELSE '{' expr '}'
    ;

compoundElem
    :   IDENT '.' IDENT
    |   arrayElem '.' IDENT
    ;

arrayElem
    :   IDENT '[' NUMBER ']'
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
