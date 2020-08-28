parser grammar TealangParser;

options {
    tokenVocab = TealangLexer;
}

program
    :   declaration* (main) EOF
    ;

module
    :   declaration* EOF
    ;

onelinecond
    :   expr EOF
    ;

statement
    :   decl (NEWLINE|SEMICOLON)
    |   condition
    |   termination
    |   assignment
    |   noRetFunctionCall
    |   NEWLINE|SEMICOLON
    ;

block
    :   LEFTFIGURE statement* RIGHTFIGURE
    ;

main
    : FUNC MAINFUNC LEFTPARA RIGHTPARA block NEWLINE*
    ;

declaration
    :   decl (NEWLINE|SEMICOLON)
    |   IMPORT MODULENAME MODULENAMEEND
    |   FUNC IDENT LEFTPARA (IDENT (COMMA IDENT)* )? RIGHTPARA block NEWLINE
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
    :   LET IDENT EQ expr                          # DeclareVar
    |   LET IDENT COMMA IDENT EQ tupleExpr         # DeclareVarTupleExpr
    |   CONST IDENT EQ NUMBER                      # DeclareNumberConst
    |   CONST IDENT EQ STRING                      # DeclareStringConst
    ;

assignment
    :   IDENT EQ expr                              # Assign
    |   IDENT COMMA IDENT EQ tupleExpr             # AssignTuple
    ;

expr
    :   IDENT                                       # Identifier
    |   NUMBER                                      # NumberLiteral
    |   STRING                                      # StringLiteral
    |	LEFTPARA expr RIGHTPARA                     # Group
    |   functionCall                                # FunctionCallExpr
    |   builtinVarExpr                              # BuiltinObject
    |   op=LNOT expr                                # Not
    |   op=BNOT expr                                # BitNot
    |	expr op=(MUL|DIV|MOD) expr                  # MulDivMod
    |	expr op=(PLUS|MINUS) expr                   # AddSub
    |   expr op=(LESS|LE|GREATER|GE|EE|NE) expr     # Relation
    |   expr op=(BOR|BXOR|BAND) expr                # BitOp
    |   expr op=(LAND|LOR) expr                     # AndOr
    |   condExpr                                    # IfExpr
    ;

tupleExpr
    :   MULW LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   ADDW LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   APPLOCALGETEX LEFTPARA ( expr COMMA expr COMMA expr ) RIGHTPARA
    |   APPGLOBALGETEX LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   ASSETHOLDINGGET LEFTPARA ( ASSETHOLDINGFIELDS COMMA expr COMMA expr ) RIGHTPARA
    |   ASSETPARAMSGET LEFTPARA ( ASSETPARAMSFIELDS COMMA expr ) RIGHTPARA
    ;

noRetFunctionCall
    :   BUILTINNORETFUNC LEFTPARA (expr (COMMA expr)* )? RIGHTPARA    # BuiltinNoRetFunCall
    ;

functionCall
    :   BUILTINFUNC LEFTPARA (expr (COMMA expr)* )? RIGHTPARA    # BuiltinFunCall
    |   IDENT LEFTPARA (expr (COMMA expr)* )? RIGHTPARA          # FunCall
    ;

builtinVarExpr
    :   GLOBAL DOT GLOBALFIELD                      # GlobalFieldExpr
    |   txn                                         # TxnFieldExpr
    |   gtxn                                        # GroupTxnFieldExpr
    |   args                                        # ArgsExpr
    ;

txn
    :   TXN DOT TXNFIELD                                               # TxnSingleFieldExpr
    |   TXN DOT TXNARRAYFIELD LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE    # TxnArrayFieldExpr
    ;

gtxn
    :   GTXN LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE DOT TXNFIELD  # GroupTxnSingleFieldExpr
    |   GTXN LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE DOT TXNARRAYFIELD LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE   # GroupTxnArrayFieldExpr
    ;

args
    :   ARGS LEFTSQUARE NUMBER RIGHTSQUARE          # ArgsNumberExpr
    |   ARGS LEFTSQUARE IDENT RIGHTSQUARE           # ArgsIdentExpr
    ;

compoundElem
    :   IDENT DOT IDENT
    |   arrayElem DOT IDENT
    ;

arrayElem
    :   IDENT LEFTSQUARE NUMBER RIGHTSQUARE
    ;

// named rules for tree-walking only
condExpr
    : IF condIfExpr LEFTFIGURE condTrueExpr RIGHTFIGURE ELSE LEFTFIGURE condFalseExpr RIGHTFIGURE
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
