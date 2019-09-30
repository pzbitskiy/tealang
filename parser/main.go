package main

import (
	"fmt"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	base "../gen/go"
)

type tealangListener struct {
	*base.BaseTealangListener

	constInt map[string]uint64
	constStr map[string]string

	variables map[string]uint64
}

func newTealangListener() (listener tealangListener) {
	listener.constInt = make(map[string]uint64)
	listener.constStr = make(map[string]string)
	listener.variables = make(map[string]uint64)
	return
}

func (l *tealangListener) EnterDeclareVar(ctx *base.DeclareVarContext) {
	fmt.Printf("EnterDeclareVar %v %v\n", ctx.IDENT(), ctx.Expr())
}

func (l *tealangListener) ExitDeclareVar(ctx *base.DeclareVarContext) {
	fmt.Printf("ExitDeclareVar %v %v\n", ctx.IDENT(), ctx.Expr())
}

func (l *tealangListener) EnterDeclareNumberConst(ctx *base.DeclareNumberConstContext) {
	fmt.Printf("EnterDeclareNumberConst %v %v\n", ctx.IDENT(), ctx.NUMBER())
	_, ok := l.constInt[ctx.IDENT().GetSymbol().GetText()]
	if !ok {
		fmt.Printf("parsing %s\n", ctx.NUMBER().GetSymbol().GetText())
		val, err := strconv.ParseUint(ctx.NUMBER().GetSymbol().GetText(), 10, 64)
		if err == nil {
			l.constInt[ctx.IDENT().GetSymbol().GetText()] = val
			return
		}
	}
	panic(fmt.Sprintf("Parsing %s failed", ctx.GetText()))
}

func (l *tealangListener) EnterDeclareStringConst(ctx *base.DeclareStringConstContext) {
	fmt.Printf("EnterDeclareStringConst %v %v\n", ctx.IDENT(), ctx.STRING())
}

func (l *tealangListener) EnterNumberLiteral(ctx *base.NumberLiteralContext) {
	fmt.Printf("EnterNumber %v\n", ctx.GetText())
}

func (l *tealangListener) ExitNumberLiteral(ctx *base.NumberLiteralContext) {
	fmt.Printf("ExitNumber %v\n", ctx.GetText())
}

func (l *tealangListener) ExitStringLiteral(ctx *base.StringLiteralContext) {
	fmt.Printf("ExitString %v\n", ctx.GetText())
}

func (l *tealangListener) Emit() {
}

func main() {
	is := antlr.NewInputStream("let a = 456; const a = 123; const b = \"123\";")
	lexer := base.NewTealangLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := base.NewTealangParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.Prog()

	listener := newTealangListener()
	fmt.Println("Starting walk")
	antlr.ParseTreeWalkerDefault.Walk(&listener, tree)
}
