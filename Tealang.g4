grammar Tealang;

prog
    :   statement* EOF                              # Program
    ;

statement
    :   declaration
    |   block
    |   condition
    |   expression
    |   termination
    |   assignment
    |   NEWLINE|SEMICOLON
    ;

block
    :   '{' statement* '}'
    ;

declaration
    :   decl (NEWLINE|SEMICOLON)
    ;

expression
    :   expr (NEWLINE|SEMICOLON)
    ;

// named rules for tree-walking only
condition
    :   IF condIfExpr condTrueBlock (ELSE condFalseBlock)?   # IfStatement
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
    // |   funcCall                                    # FunctionCall
    |   builtinFuncCall                             # BuiltinFunction
    |   builtinVarExpr                              # BuiltinObject
    // |   compoundElem                                # ObjElement
    |   op='!' expr                                 # Not
    |   op='~' expr                                 # BitNot
    |	expr op=('*'|'/'|'%') expr                  # MulDivMod
    |	expr op=('+'|'-') expr                      # SumSub
    |   expr op=('<'|'<='|'>'|'>='|'=='|'!=') expr  # Relation
    |   expr op=('|'|'^'|'&') expr                  # BitOp
    |   expr op=('&&'|'||') expr                    # AndOr
    |   condExpr                                    # IfExpr
    ;

builtinFuncCall
    :   BUILTINFUNC '(' argList ')'                 # BuiltinFunctionCall
    ;

argList
    :   expr (',' expr)*
    ;

builtinVarExpr
    :   GLOBAL '.' GLOBALFIELD                      # GlobalFieldExpr
    |   TXN '.' TXNFIELD                            # TxnFieldExpr
    |   GTXN '[' NUMBER ']' '.' TXNFIELD            # GroupTxnFieldExpr
    |   ACCOUNT '.' ACCOUNTFIELD                    # AccountFieldExpr
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

GLOBALFIELD
    :   ROUND
    |   MINTXNFEE
    |   MINBALANCE
    |   MAXTXNLIFE
    |   BLOCKTIME
    |   ZEROADDRESS
    |   GROUPSIZE
    ;

TXNFIELD
    :   SENDER
    |   FEE
    |   FIRSTVALID
    |   LASTVALID
    |   NOTE
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
    |   SENDERBALANCE
    ;

ACCOUNTFIELD
    :   BALANCE
    |   FROZEN
    |   EXISTS
    ;

BUILTINFUNC
    :   SHA256
    |   KECCAK256
    |   SHA512
    |   ED25519
    |   RAND
    ;

LET         : 'let' ;
CONST       : 'const' ;
ERR         : 'error' ;
RET         : 'return' ;
IF          : 'if' ;
ELSE        : 'else' ;

GLOBAL      : 'global';
TXN         : 'txn';
GTXN        : 'gtxn';
ACCOUNT     : 'account';

ROUND       : 'Round' ;
MINTXNFEE   : 'MinTxnFee' ;
MINBALANCE  : 'MinBalance' ;
MAXTXNLIFE  : 'MaxTxnLife' ;
BLOCKTIME   : 'BlockTime' ;
ZEROADDRESS : 'ZeroAddress' ;
GROUPSIZE   : 'GroupSize' ;

SENDER      : 'Sender' ;
FEE         : 'Fee' ;
FIRSTVALID  : 'FirstValid' ;
LASTVALID   : 'LastValid' ;
NOTE        : 'Note' ;
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
SENDERBALANCE : 'SenderBalance' ;

BALANCE     : 'Balance' ;
FROZEN      : 'Frozen' ;
EXISTS      : 'Exists' ;

SHA256      : 'sha256' ;
KECCAK256   : 'keccak256' ;
SHA512      : 'sha512_256' ;
ED25519     : 'ed25519verify' ;
RAND        : 'rand' ;


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
