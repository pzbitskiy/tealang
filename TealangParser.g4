parser grammar TealangParser;

options {
    tokenVocab = TealangLexer;
}

program
    :   declaration* logic EOF
    ;

module
    :   declaration* EOF
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
    :   LEFTFIGURE statement* RIGHTFIGURE
    ;

logic
    : FUNC LOGIC LEFTPARA TXN COMMA GTXN COMMA ARGS RIGHTPARA block NEWLINE*
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
    |   CONST IDENT EQ NUMBER                      # DeclareNumberConst
    |   CONST IDENT EQ STRING                      # DeclareStringConst
    ;

assignment
    :   IDENT '=' expr                              # Assign
    ;

expr
    :   IDENT                                       # Identifier
    |   NUMBER                                      # NumberLiteral
    |   STRING                                      # StringLiteral
    |	LEFTPARA expr RIGHTPARA                     # Group
    |   functionCall                                # FunctionCallExpr
    |   builtinVarExpr                              # BuiltinObject
    // |   compoundElem                                # ObjElement
    |   op=LNOT expr                                # Not
    |   op=BNOT expr                                # BitNot
    |	expr op=(MUL|DIV|MOD) expr                  # MulDivMod
    |	expr op=(PLUS|MINUS) expr                   # AddSub
    |   expr op=(LESS|LE|GREATER|GE|EE|NE) expr     # Relation
    |   expr op=(BOR|BXOR|BAND) expr                # BitOp
    |   expr op=(LAND|LOR) expr                     # AndOr
    |   condExpr                                    # IfExpr
    ;

functionCall
    :   BUILTINFUNC LEFTPARA (expr (COMMA expr)* )? RIGHTPARA    # BuiltinFunCall
    |   IDENT LEFTPARA (expr (COMMA expr)* )? RIGHTPARA          # FunCall
    ;

builtinVarExpr
    :   GLOBAL DOT GLOBALFIELD                      # GlobalFieldExpr
    |   TXN DOT TXNFIELD                            # TxnFieldExpr
    |   GTXN LEFTSQUARE NUMBER RIGHTSQUARE DOT TXNFIELD   # GroupTxnFieldExpr
    |   ARGS LEFTSQUARE NUMBER RIGHTSQUARE          # ArgsExpr
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
