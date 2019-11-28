grammar Tealang;

program
    :   declaration* logic EOF
    ;

statement
    :   decl (NEWLINE|SEMICOLON)
    |   condition
    |   expr (NEWLINE|SEMICOLON)
    |   termination
    |   assignment
    |   NEWLINE|SEMICOLON
    ;

block
    :   '{' statement* '}'
    ;

logic
    : FUNC 'logic' '(' TXN ',' GTXN ',' ARGS ')' block NEWLINE*
    ;

declaration
    :   decl (NEWLINE|SEMICOLON)
    |   FUNC IDENT '(' (IDENT (',' IDENT)* )? ')' block NEWLINE
    |   NEWLINE|SEMICOLON
    ;

// named rules for tree-walking only
condition
    :   IF condIfExpr condTrueBlock (NEWLINE? ELSE condFalseBlock)?   # IfStatement
    ;

condTrueBlock
    : block                                         # IfStatementTrue
    ;

condFalseBlock
    : block                                         # IfStatementFalse
    ;

termination
    :   ERR (NEWLINE|SEMICOLON)                     # TermError
    |   RET expr (NEWLINE|SEMICOLON)                # TermReturn
    ;

decl
    :   LET IDENT '=' expr                          # DeclareVar
    |   CONST IDENT '=' NUMBER                      # DeclareNumberConst
    |   CONST IDENT '=' STRING                      # DeclareStringConst
    ;

assignment
    :   IDENT '=' expr                              # Assign
    ;

expr
    :   IDENT                                       # Identifier
    |   NUMBER                                      # NumberLiteral
    |   STRING                                      # StringLiteral
    |	'(' expr ')'                                # Group
    |   functionCall                                # FunctionCallExpr
    |   builtinVarExpr                              # BuiltinObject
    // |   compoundElem                                # ObjElement
    |   op='!' expr                                 # Not
    |   op='~' expr                                 # BitNot
    |	expr op=('*'|'/'|'%') expr                  # MulDivMod
    |	expr op=('+'|'-') expr                      # AddSub
    |   expr op=('<'|'<='|'>'|'>='|'=='|'!=') expr  # Relation
    |   expr op=('|'|'^'|'&') expr                  # BitOp
    |   expr op=('&&'|'||') expr                    # AndOr
    |   condExpr                                    # IfExpr
    ;

functionCall
    :   BUILTINFUNC '(' (expr (',' expr)* )? ')'    # BuiltinFunCall
    |   IDENT '(' (expr (',' expr)* )? ')'          # FunCall
    ;

builtinVarExpr
    :   GLOBAL '.' GLOBALFIELD                      # GlobalFieldExpr
    |   TXN '.' TXNFIELD                            # TxnFieldExpr
    |   GTXN '[' NUMBER ']' '.' TXNFIELD            # GroupTxnFieldExpr
    |   ARGS '[' NUMBER ']'                         # ArgsExpr
    ;

compoundElem
    :   IDENT '.' IDENT
    |   arrayElem '.' IDENT
    ;

arrayElem
    :   IDENT '[' NUMBER ']'
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

GLOBALFIELD
    :   MINTXNFEE
    |   MINBALANCE
    |   MAXTXNLIFE
    |   ZEROADDRESS
    |   GROUPSIZE
    ;

TXNFIELD
    :   SENDER
    |   FEE
    |   FIRSTVALID
    |   LASTVALID
    |   NOTE
    |   LEASE
    |   RECEIVER
    |   AMOUNT
    |   CLOSEREMINDERTO
    |   VOTEPK
    |   SELECTIONPK
    |   VOTEFIRST
    |   VOTELAST
    |   VOTEKD
    |   TYPE
    |   TYPEENUM
    |   XFERASSET
    |   AAMOUNT
    |   ASENDER
    |   ARECEIVER
    |   ACLOSETO
    |   GROUPINDEX
    |   TXID
    ;

BUILTINFUNC
    :   SHA256
    |   KECCAK256
    |   SHA512
    |   ED25519
    |   MULW
    ;

LET         : 'let' ;
CONST       : 'const' ;
ERR         : 'error' ;
RET         : 'return' ;
IF          : 'if' ;
ELSE        : 'else' ;
FUNC        : 'function' ;

GLOBAL      : 'global';
TXN         : 'txn';
GTXN        : 'gtxn';
ARGS        : 'args';


MINTXNFEE   : 'MinTxnFee' ;
MINBALANCE  : 'MinBalance' ;
MAXTXNLIFE  : 'MaxTxnLife' ;
ZEROADDRESS : 'ZeroAddress' ;
GROUPSIZE   : 'GroupSize' ;

SENDER      : 'Sender' ;
FEE         : 'Fee' ;
FIRSTVALID  : 'FirstValid' ;
LASTVALID   : 'LastValid' ;
NOTE        : 'Note' ;
LEASE       : 'Lease';
RECEIVER    : 'Receiver' ;
AMOUNT      : 'Amount' ;
CLOSEREMINDERTO : 'CloseRemainderTo' ;
VOTEPK      : 'VotePK' ;
SELECTIONPK : 'SelectionPK' ;
VOTEFIRST   : 'VoteFirst' ;
VOTELAST    : 'VoteLast' ;
VOTEKD      : 'VoteKeyDilution' ;
TYPE        : 'Type' ;
TYPEENUM    : 'TypeEnum' ;
XFERASSET   : 'XferAsset' ;
AAMOUNT     : 'AssetAmount' ;
ASENDER     : 'AssetSender' ;
ARECEIVER   : 'AssetReceiver' ;
ACLOSETO    : 'AssetCloseTo' ;
GROUPINDEX  : 'GroupIndex' ;
TXID        : 'TxId' ;

SHA256      : 'sha256' ;
KECCAK256   : 'keccak256' ;
SHA512      : 'sha512_256' ;
ED25519     : 'ed25519verify' ;
MULW        : 'mulw';

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
