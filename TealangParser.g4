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
    |   logStatement
    |   innertxn
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
    |   INLINE? FUNC IDENT LEFTPARA (IDENT (COMMA IDENT)* )? RIGHTPARA block NEWLINE
    |   NEWLINE|SEMICOLON
    ;

// named rules for tree-walking only
condition
    :   IF condIfExpr condTrueBlock (NEWLINE? ELSE condFalseBlock)?   # IfStatement
    |   FOR condForExpr condTrueBlock   # ForStatement
    ;

condTrueBlock
    : block                                         # IfStatementTrue
    ;

condFalseBlock
    : block                                         # IfStatementFalse
    ;

innertxn
    :   INNERTXN DOT ITXNBEGIN LEFTPARA RIGHTPARA                       # InnerTxnBegin
    |   INNERTXN DOT ITXNNEXT LEFTPARA RIGHTPARA                        # InnerTxnNext
    |   INNERTXN DOT ITXNEND LEFTPARA RIGHTPARA                         # InnerTxnEnd
    |   INNERTXN DOT TXNFIELD EQ expr                                   # InnerTxnAssign
    |   INNERTXN DOT TXNARRAYFIELD DOT ITXNPUSH LEFTPARA expr RIGHTPARA # InnerTxnArrayAssign
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
    |   LET IDENT COMMA IDENT COMMA IDENT COMMA IDENT EQ tupleExpr # DeclareQuadrupleExpr
    |   CONST IDENT EQ NUMBER                      # DeclareNumberConst
    |   CONST IDENT EQ STRING                      # DeclareStringConst
    ;

assignment
    :   IDENT EQ expr                              # Assign
    |   IDENT COMMA IDENT EQ tupleExpr             # AssignTuple
    |   IDENT COMMA IDENT COMMA IDENT COMMA IDENT EQ tupleExpr      # AssignQuadruple
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
    |   (TOINT|TOBYTE) LEFTPARA (expr) RIGHTPARA    # TypeCastExpr
    ;

tupleExpr
    :   MULW LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   ADDW LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   EXPW LEFTPARA ( expr COMMA expr ) RIGHTPARA
    |   DIVMODW LEFTPARA ( expr COMMA expr COMMA expr COMMA expr ) RIGHTPARA
    |   ECDSADECOMPRESS LEFTPARA ( ECDSACURVE COMMA expr ) RIGHTPARA
    |   ECDSARECOVER LEFTPARA ( ECDSACURVE COMMA expr COMMA expr COMMA expr COMMA expr ) RIGHTPARA
    |   builtinVarTupleExpr
    ;

builtinVarTupleExpr
    :   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (APPGETEX|ASSETHLDBALANCE|ASSETHLDFROZEN) LEFTPARA expr (COMMA expr)? RIGHTPARA
    |   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT ACCTPARAMS LEFTPARA RIGHTPARA
    |   APPS LEFTSQUARE expr RIGHTSQUARE DOT (APPGETEX|APPPARAMSFIELDS) LEFTPARA expr RIGHTPARA
    |   APPS LEFTSQUARE expr RIGHTSQUARE DOT APPPARAMSFIELDS
    |   ASSETS LEFTSQUARE expr RIGHTSQUARE DOT ASSETPARAMSFIELDS
    ;

builtinVarStatement
    :   ACCOUNTS LEFTSQUARE expr RIGHTSQUARE DOT (APPPUT|APPDEL) LEFTPARA expr (COMMA expr)? RIGHTPARA
    |   APPS LEFTSQUARE expr RIGHTSQUARE DOT (APPPUT|APPDEL) LEFTPARA expr (COMMA expr)? RIGHTPARA
    ;

logStatement
    :   LOG LEFTPARA expr RIGHTPARA                 # DoLog
    ;

functionCall
    :   BUILTINFUNC LEFTPARA ( expr (COMMA expr)* )? RIGHTPARA    # BuiltinFunCall
    |   IDENT LEFTPARA ( expr (COMMA expr)* )? RIGHTPARA          # FunCall
    |   ECDSAVERIFY LEFTPARA ( ECDSACURVE COMMA expr COMMA expr COMMA expr COMMA expr COMMA expr ) RIGHTPARA    # EcDsaFunCall
    |   EXTRACT LEFTPARA ( (EXTRACTOPT COMMA)? expr COMMA expr (COMMA expr)? ) RIGHTPARA    # ExtractFunCall
    ;

builtinVarExpr
    :   GLOBAL DOT GLOBALFIELD                      # GlobalFieldExpr
    |   txn                                         # TxnFieldExpr
    |   gtxn                                        # GroupTxnFieldExpr
    |   ARGS LEFTSQUARE expr RIGHTSQUARE            # ArgsExpr
    |   accounts                                    # AccountsExpr
    |   apps                                        # AppsExpr
    |   itxn                                        # InnerTxnFieldExpr
    |   gitxn                                       # GroupInnerTxnFieldExpr
    ;

txn
    :   TXN DOT TXNFIELD                                    # TxnSingleFieldExpr
    |   TXN DOT TXNARRAYFIELD LEFTSQUARE (expr) RIGHTSQUARE # TxnArrayFieldExpr
    ;

gtxn
    :   GTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNFIELD                                   # GroupTxnSingleFieldExpr
    |   GTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNARRAYFIELD LEFTSQUARE expr RIGHTSQUARE  # GroupTxnArrayFieldExpr
    ;

itxn
    :   INNERTXN DOT TXNFIELD                                    # InnerTxnSingleFieldExpr
    |   INNERTXN DOT TXNARRAYFIELD LEFTSQUARE (expr) RIGHTSQUARE # InnerTxnArrayFieldExpr
    ;

gitxn
    :   GINNERTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNFIELD                                    # GroupInnerTxnSingleFieldExpr
    |   GINNERTXN LEFTSQUARE expr RIGHTSQUARE DOT TXNARRAYFIELD LEFTSQUARE expr RIGHTSQUARE   # GroupInnerTxnArrayFieldExpr
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
