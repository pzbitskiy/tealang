package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	parser "../gen/go"
)

type tealangType int
type parserFlag int

const (
	integer tealangType = 1
	bytes   tealangType = 2
)

const (
	condElsePresent parserFlag = 1
)

type literal struct {
	offset int
	tp     tealangType
}

type parserState struct {
	flags   map[parserFlag]bool
	labelID uint
}

type tealangListener struct {
	*parser.BaseTealangListener

	literals map[string]literal // literal value -> index in intc / bytec
	intc     []string
	bytec    [][]byte

	constants     map[string]string // constant name -> value
	variables     map[string]uint
	variableIndex uint

	labelCounter    uint
	nestedCondStack []parserState

	program strings.Builder
}

const ifExprPrefix = "if_expr"

func newTealangListener() (listener tealangListener) {
	listener.literals = make(map[string]literal)
	listener.intc = make([]string, 0, 128)
	listener.bytec = make([][]byte, 0, 128)

	listener.variables = make(map[string]uint)
	listener.constants = make(map[string]string)

	listener.variableIndex = 0
	listener.labelCounter = 0
	listener.nestedCondStack = make([]parserState, 0, 128)

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
		parsed, err := ParseStringLiteral(rawValue)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", err))
		}
		l.bytec = append(l.bytec, parsed)
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
		parsed, err := ParseStringLiteral(rawValue)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", err))
		}
		l.bytec = append(l.bytec, parsed)
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

func (l *tealangListener) ExitAssign(ctx *parser.AssignContext) {
	varName := ctx.IDENT().GetSymbol().GetText()
	if _, ok := l.constants[varName]; ok {
		panic("assignment to a constant")
	}
	if _, ok := l.variables[varName]; !ok {
		panic("undefined variable")
	}

	l.program.WriteString(fmt.Sprintf("store %d\n", l.variables[varName]))
}

//--------------------------------------------------------------------------------------------------
// code generation for If-Expr
// the idea is to set ID on enter, and use this ID during tree walk to name goto labels
func (l *tealangListener) EnterIfExpr(ctx *parser.IfExprContext) {
	fmt.Println("EnterIfExpr")
	l.labelCounter++
	state := parserState{make(map[parserFlag]bool), l.labelCounter}
	l.nestedCondStack = append(l.nestedCondStack, state)
}

func (l *tealangListener) ExitIfExpr(ctx *parser.IfExprContext) {
	fmt.Println("ExitIfExpr")
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("%s_end_%d:\n", ifExprPrefix, labelSuffix))
	l.nestedCondStack = l.nestedCondStack[:len(l.nestedCondStack)]
}

func (l *tealangListener) ExitIfExprCond(ctx *parser.IfExprCondContext) {
	fmt.Println("// ExitIfExprCond")
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	labelSuffix := state.labelID
	l.program.WriteString(fmt.Sprintf("!\nbnz %s_false_%d\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) EnterIfExprTrue(ctx *parser.IfExprTrueContext) {
	fmt.Println("// EnterIfExprTrue")
	// do nothing
}

func (l *tealangListener) ExitIfExprTrue(ctx *parser.IfExprTrueContext) {
	fmt.Println("// ExitIfExprTrue")
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("int 1\nbnz %s_end_%d\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) EnterIfExprFalse(ctx *parser.IfExprFalseContext) {
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	state.flags[condElsePresent] = true
	l.nestedCondStack[len(l.nestedCondStack)-1] = state

	labelSuffix := state.labelID
	l.program.WriteString(fmt.Sprintf("%s_false_%d:\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) ExitIfExprFalse(ctx *parser.IfExprFalseContext) {
	// do nothing
}

// code generation for If-Statement
// similar to If-Expr but ELSE block might be optional,
// so EnterIfStatementFalse sets a signal variable
// that is taken into account in ExitIfStatement
func (l *tealangListener) EnterIfStatement(ctx *parser.IfStatementContext) {
	fmt.Println("IfStatement")
	l.labelCounter++
	state := parserState{make(map[parserFlag]bool), l.labelCounter}
	l.nestedCondStack = append(l.nestedCondStack, state)
}

func (l *tealangListener) ExitIfStatement(ctx *parser.IfStatementContext) {
	fmt.Println("ExitIfStatement")
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	labelSuffix := state.labelID
	if !state.flags[condElsePresent] {
		l.program.WriteString(fmt.Sprintf("%s_false_%d:\n", ifExprPrefix, labelSuffix))
	}
	l.program.WriteString(fmt.Sprintf("%s_end_%d:\n", ifExprPrefix, labelSuffix))

	l.nestedCondStack = l.nestedCondStack[:len(l.nestedCondStack)]
}

func (l *tealangListener) EnterIfStatementTrue(ctx *parser.IfStatementTrueContext) {
	fmt.Println("// EnterIfStatementTrue")
	// do nothing
}

func (l *tealangListener) ExitIfStatementTrue(ctx *parser.IfStatementTrueContext) {
	fmt.Println("// ExitIfStatementTrue")
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("int 1\nbnz %s_end_%d\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) EnterIfStatementFalse(ctx *parser.IfStatementFalseContext) {
	fmt.Println("// EnterIfStatementFalse")
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	state.flags[condElsePresent] = true
	l.nestedCondStack[len(l.nestedCondStack)-1] = state

	labelSuffix := state.labelID
	l.program.WriteString(fmt.Sprintf("%s_false_%d:\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) ExitIfStatementFalse(ctx *parser.IfStatementFalseContext) {
	// do nothing
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
			// TODO: decode string (remove quotes, decode \x)
			fmt.Printf("0x%s ", hex.EncodeToString(value))
		}
		fmt.Print("\n")
	}

	fmt.Println(l.program.String())
}

func main() {
	source := `
let a = 456; const b = 123; const c = "1234567890123"; let d = 1 + 2 ; let e = if a > 0 {1} else {2};
if e == 1 {
	let x = a + b;
} else {
	let y = 1;
}
x = 2;
`
	fmt.Print(source)
	is := antlr.NewInputStream(source)
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
