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
    |   builtinVarStatement
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
    |   FOR condForExpr condTrueBlock   #ForStatement
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
    |   ASSERT LEFTPARA expr RIGHTPARA              # TermAssert
    |   BREAK (NEWLINE|SEMICOLON)                   # Break
    ;

decl
    :   LET IDENT EQ expr                          # DeclareVar
    |   LET IDENT COMMA IDENT EQ tupleExpr         # DeclareVarTupleExpr
    |   LET IDENT COMMA IDENT COMMA IDENT COMMA IDENT EQ tupleExpr # DeclareVarTupleExpr
    |   CONST IDENT EQ NUMBER                      # DeclareNumberConst
    |   CONST IDENT EQ STRING                      # DeclareStringConst
    ;

assignment
    :   IDENT EQ expr                              # Assign
    |   IDENT COMMA IDENT EQ tupleExpr             # AssignTuple
    |   IDENT COMMA IDENT COMMA IDENT COMMA IDENT EQ tupleExpr      # AssignTuple
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
    |   DIVMODW LEFTPARA ( expr COMMA expr COMMA expr COMMA expr ) RIGHTPARA
    |   builtinVarTupleExpr
    ;

builtinVarTupleExpr
    :   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (APPGETEX|ASSETHLDBALANCE|ASSETHLDFROZEN) LEFTPARA expr (COMMA expr)? RIGHTPARA
    |   APPS LEFTSQUARE expr RIGHTSQUARE DOT APPGETEX LEFTPARA expr RIGHTPARA
    |   ASSETS LEFTSQUARE expr RIGHTSQUARE DOT ASSETPARAMSFIELDS
    ;

builtinVarStatement
    :   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (APPPUT|APPDEL) LEFTPARA expr (COMMA expr)? RIGHTPARA
    |   APPS LEFTSQUARE expr RIGHTSQUARE DOT (APPPUT|APPDEL) LEFTPARA expr (COMMA expr)? RIGHTPARA
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
    |   accounts                                    # AccountsExpr
    |   apps                                        # AppsExpr
    ;

txn
    :   TXN DOT TXNFIELD                                            # TxnSingleFieldExpr
    |   TXN DOT TXNARRAYFIELD LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE # TxnArrayFieldExpr
    ;

gtxn
    :   GTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNFIELD                                             # GroupTxnSingleFieldExpr
    |   GTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNARRAYFIELD LEFTSQUARE (IDENT|NUMBER) RIGHTSQUARE  # GroupTxnArrayFieldExpr
    ;

args
    :   ARGS LEFTSQUARE NUMBER RIGHTSQUARE          # ArgsNumberExpr
    |   ARGS LEFTSQUARE IDENT RIGHTSQUARE           # ArgsIdentExpr
    ;

accounts
    :   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (BALANCE|MINIMUMBALANCE)                 # AccountsBalanceExpr
    |   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (OPTEDIN|APPGET) LEFTPARA expr RIGHTPARA # AccountsSingleMethodsExpr
    ;

apps
    :   APPS LEFTSQUARE expr RIGHTSQUARE DOT APPGET LEFTPARA expr RIGHTPARA   # AppsSingleMethodsExpr
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

condForExpr
    : expr                                          # ForExprCond
    ;
