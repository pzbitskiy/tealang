package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	parser "../gen/go"
)

type tealangType int

const (
	integer tealangType = 1
	bytes   tealangType = 2
)

type literal struct {
	offset int
	tp     tealangType
}

type tealangListener struct {
	*parser.BaseTealangListener

	literals map[string]literal // literal value -> index in intc / bytec
	intc     []string
	bytec    []string

	constants     map[string]string // constant name -> value
	variables     map[string]uint
	variableIndex uint

	counter         uint
	nestedCondStack []uint

	program strings.Builder
}

func newTealangListener() (listener tealangListener) {
	listener.literals = make(map[string]literal)
	listener.intc = make([]string, 0, 128)
	listener.bytec = make([]string, 0, 128)

	listener.variables = make(map[string]uint)
	listener.constants = make(map[string]string)

	listener.variableIndex = 0
	listener.counter = 0
	listener.nestedCondStack = make([]uint, 0, 128)

	listener.program = strings.Builder{}
	return
}

func (l *tealangListener) ExitDeclareVar(ctx *parser.DeclareVarContext) {
	// fmt.Printf("ExitDeclareVar %v %v\n", ctx.IDENT(), ctx.Expr())
	varName := ctx.IDENT().GetSymbol().GetText()
	if _, ok := l.constants[varName]; ok {
		panic("already declared")
	}
	if _, ok := l.variables[varName]; ok {
		panic("already allocated")
	}

	l.variables[varName] = l.variableIndex
	l.variableIndex++

	l.program.WriteString(fmt.Sprintf("store %d\n", l.variables[varName]))
}

func (l *tealangListener) EnterDeclareNumberConst(ctx *parser.DeclareNumberConstContext) {
	// fmt.Printf("EnterDeclareNumberConst %v %v\n", ctx.IDENT(), ctx.NUMBER())
	varName := ctx.IDENT().GetSymbol().GetText()
	if _, ok := l.constants[varName]; ok {
		panic("already defined as a constant")
	}
	if _, ok := l.variables[varName]; ok {
		panic("already defined as a variable")
	}

	rawValue := ctx.NUMBER().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.intc)
		l.intc = append(l.intc, rawValue)
		l.literals[rawValue] = literal{idx, integer}
	}
	l.constants[varName] = rawValue
}

func (l *tealangListener) EnterDeclareStringConst(ctx *parser.DeclareStringConstContext) {
	// fmt.Printf("EnterDeclareStringConst %v %v\n", ctx.IDENT(), ctx.STRING())
	varName := ctx.IDENT().GetSymbol().GetText()
	if _, ok := l.constants[varName]; ok {
		panic("already defined as a constant")
	}
	if _, ok := l.variables[varName]; ok {
		panic("already defined as a variable")
	}

	rawValue := ctx.STRING().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.bytec)
		l.bytec = append(l.bytec, rawValue)
		l.literals[rawValue] = literal{idx, bytes}
	}
	l.constants[varName] = rawValue
}

func (l *tealangListener) EnterNumberLiteral(ctx *parser.NumberLiteralContext) {
	// fmt.Printf("Number %v\n", ctx.GetText())
	rawValue := ctx.NUMBER().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.intc)
		l.intc = append(l.intc, rawValue)
		l.literals[rawValue] = literal{idx, integer}
	}
	l.program.WriteString(fmt.Sprintf("intc %d\n", l.literals[rawValue].offset))
}

func (l *tealangListener) EnterStringLiteral(ctx *parser.StringLiteralContext) {
	// fmt.Printf("String %v\n", ctx.GetText())
	rawValue := ctx.STRING().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.bytec)
		l.bytec = append(l.bytec, rawValue)
		l.literals[rawValue] = literal{idx, bytes}
	}
	l.program.WriteString(fmt.Sprintf("bytec %d", l.literals[rawValue].offset))
}

func (l *tealangListener) ExitSumSub(ctx *parser.SumSubContext) {
	op := ctx.GetOp().GetText()
	if op != "+" && op != "-" {
		panic("Unknown op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) ExitMulDivMod(ctx *parser.MulDivModContext) {
	op := ctx.GetOp().GetText()
	if op != "*" && op != "/" && op != "%" {
		panic("Unknown op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) ExitRelation(ctx *parser.RelationContext) {
	op := ctx.GetOp().GetText()
	ops := map[string]bool{
		"<":  true,
		"<=": true,
		">":  true,
		">=": true,
		"==": true,
		"!=": true,
	}
	if !ops[op] {
		panic("Unknown op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) EnterIdentifier(ctx *parser.IdentifierContext) {
	varName := ctx.IDENT().GetSymbol().GetText()
	// TODO: add globals ?

	if value, ok := l.constants[varName]; ok {
		// replace constant with its value
		lit := l.literals[value]
		opcode := "bytec "
		if lit.tp == integer {
			opcode = "intc"
		}
		l.program.WriteString(fmt.Sprintf("%s %d\n", opcode, lit.offset))
		return
	}

	if _, ok := l.variables[varName]; ok {
		// load variable
		l.program.WriteString(fmt.Sprintf("load %d\n", l.variables[varName]))
		return
	}

	panic("Unknown identifier")
}

func (l *tealangListener) EnterIfExpr(ctx *parser.IfExprContext) {
	l.counter++
	l.nestedCondStack = append(l.nestedCondStack, l.counter)
}

func (l *tealangListener) ExitIfExpr(ctx *parser.IfExprContext) {
	l.nestedCondStack = l.nestedCondStack[:len(l.nestedCondStack)]
}

func (l *tealangListener) ExitIfExprCond(ctx *parser.IfExprCondContext) {
	suffix := l.nestedCondStack[len(l.nestedCondStack)-1]
	l.program.WriteString(fmt.Sprintf("!\nbnz if_expr_false_%d\n", suffix))
}

func (l *tealangListener) EnterIfExprTrue(ctx *parser.IfExprTrueContext) {
	// do nothing
}

func (l *tealangListener) ExitIfExprTrue(ctx *parser.IfExprTrueContext) {
	suffix := l.nestedCondStack[len(l.nestedCondStack)-1]
	l.program.WriteString(fmt.Sprintf("int 1\nbnz if_expr_end_%d\n", suffix))
}

func (l *tealangListener) EnterIfExprFalse(ctx *parser.IfExprFalseContext) {
	suffix := l.nestedCondStack[len(l.nestedCondStack)-1]
	l.program.WriteString(fmt.Sprintf("if_expr_false_%d:\n", suffix))
}

func (l *tealangListener) ExitIfExprFalse(ctx *parser.IfExprFalseContext) {
	suffix := l.nestedCondStack[len(l.nestedCondStack)-1]
	l.program.WriteString(fmt.Sprintf("if_expr_end_%d:\n", suffix))
}

func (l *tealangListener) Emit() {
	if len(l.literals) != len(l.intc)+len(l.bytec) {
		panic("literals unbalanced")
	}
	if len(l.intc) > 0 {
		fmt.Print("intcblock ")
		for _, value := range l.intc {
			fmt.Printf("%s ", value)
		}
		fmt.Print("\n")
	}

	if len(l.bytec) > 0 {
		fmt.Print("bytecblock ")
		for _, value := range l.bytec {
			fmt.Printf("0x%s ", hex.EncodeToString([]byte(value)))
		}
		fmt.Print("\n")
	}

	fmt.Println(l.program.String())
}

func main() {
	is := antlr.NewInputStream("let a = 456; const b = 123; const c = \"1234567890123\"; let d = 1 + 2 ; let e = if a > 0 {1} else {2}\n")
	lexer := parser.NewTealangLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTealangParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.BuildParseTrees = true
	tree := p.Prog()

	listener := newTealangListener()
	antlr.ParseTreeWalkerDefault.Walk(&listener, tree)

	listener.Emit()

}
