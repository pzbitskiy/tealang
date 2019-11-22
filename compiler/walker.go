package compiler

import (
	b "bytes"
	"encoding/hex"
	"fmt"
	"io"
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
const endProgramLabel = "end_program"
const trueConstName = "TRUE_INTERNAL"
const falseConstName = "FALSE_INTERNAL"
const trueConstValue = "1"
const falseConstValue = "0"

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

/* Program start-end */
func (l *tealangListener) EnterProgram(ctx *parser.ProgramContext) {
	// add TRUE and FALSE int constants
	l.intc = append(l.intc, falseConstValue)
	l.literals[falseConstValue] = literal{0, integer}
	l.constants[falseConstName] = falseConstValue

	l.intc = append(l.intc, trueConstValue)
	l.literals[trueConstValue] = literal{1, integer}
	l.constants[trueConstName] = trueConstValue
}

func (l *tealangListener) ExitProgram(ctx *parser.ProgramContext) {
	l.program.WriteString(fmt.Sprintf("%s:\n", endProgramLabel))
}

/* Declarations */

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
		parsed, err := parseStringLiteral(rawValue)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", err))
		}
		l.bytec = append(l.bytec, parsed)
		l.literals[rawValue] = literal{idx, bytes}
	}
	l.constants[varName] = rawValue
}

/* Literals */

func (l *tealangListener) EnterNumberLiteral(ctx *parser.NumberLiteralContext) {
	rawValue := ctx.NUMBER().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.intc)
		l.intc = append(l.intc, rawValue)
		l.literals[rawValue] = literal{idx, integer}
	}
	l.program.WriteString(fmt.Sprintf("intc %d\n", l.literals[rawValue].offset))
}

func (l *tealangListener) EnterStringLiteral(ctx *parser.StringLiteralContext) {
	rawValue := ctx.STRING().GetSymbol().GetText()
	if _, ok := l.literals[rawValue]; !ok {
		idx := len(l.bytec)
		parsed, err := parseStringLiteral(rawValue)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", err))
		}
		l.bytec = append(l.bytec, parsed)
		l.literals[rawValue] = literal{idx, bytes}
	}
	l.program.WriteString(fmt.Sprintf("bytec %d\n", l.literals[rawValue].offset))
}

/* Binary operators */

func (l *tealangListener) ExitAddSub(ctx *parser.AddSubContext) {
	op := ctx.GetOp().GetText()
	if op != "+" && op != "-" {
		panic("Unknown op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) ExitMulDivMod(ctx *parser.MulDivModContext) {
	op := ctx.GetOp().GetText()
	if op != "*" && op != "/" && op != "%" {
		panic("Unknown MulDiv op")
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
		panic("Unknown Rel op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) ExitAndOr(ctx *parser.AndOrContext) {
	op := ctx.GetOp().GetText()
	if op != "&&" && op != "||" {
		panic("Unknown AndOr op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

func (l *tealangListener) ExitBitOp(ctx *parser.BitOpContext) {
	op := ctx.GetOp().GetText()
	if op != "|" && op != "^" && op != "&" {
		panic("Unknown BitOp op")
	}
	l.program.WriteString(fmt.Sprintf("%s\n", op))
}

/* Unary operations */

func (l *tealangListener) ExitNot(ctx *parser.NotContext) {
	l.program.WriteString(fmt.Sprintf("!\n"))
}

func (l *tealangListener) ExitBitNot(ctx *parser.BitNotContext) {
	l.program.WriteString(fmt.Sprintf("~\n"))
}

/* Indent and assignment */

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

/* If-Else */

// code generation for If-Expr
// the idea is to set ID on enter, and use this ID during tree walk to name goto labels
func (l *tealangListener) EnterIfExpr(ctx *parser.IfExprContext) {
	l.labelCounter++
	state := parserState{make(map[parserFlag]bool), l.labelCounter}
	l.nestedCondStack = append(l.nestedCondStack, state)
}

func (l *tealangListener) ExitIfExpr(ctx *parser.IfExprContext) {
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("%s_end_%d:\n", ifExprPrefix, labelSuffix))
	l.nestedCondStack = l.nestedCondStack[:len(l.nestedCondStack)]
}

func (l *tealangListener) ExitIfExprCond(ctx *parser.IfExprCondContext) {
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	labelSuffix := state.labelID
	l.program.WriteString(fmt.Sprintf("!\nbnz %s_false_%d\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) EnterIfExprTrue(ctx *parser.IfExprTrueContext) {
	// do nothing
}

func (l *tealangListener) ExitIfExprTrue(ctx *parser.IfExprTrueContext) {
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("intc %d\nbnz %s_end_%d\n", l.literals[trueConstValue].offset, ifExprPrefix, labelSuffix))
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
	l.labelCounter++
	state := parserState{make(map[parserFlag]bool), l.labelCounter}
	l.nestedCondStack = append(l.nestedCondStack, state)
}

func (l *tealangListener) ExitIfStatement(ctx *parser.IfStatementContext) {
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	labelSuffix := state.labelID
	if !state.flags[condElsePresent] {
		l.program.WriteString(fmt.Sprintf("%s_false_%d:\n", ifExprPrefix, labelSuffix))
	}
	l.program.WriteString(fmt.Sprintf("%s_end_%d:\n", ifExprPrefix, labelSuffix))

	l.nestedCondStack = l.nestedCondStack[:len(l.nestedCondStack)]
}

func (l *tealangListener) EnterIfStatementTrue(ctx *parser.IfStatementTrueContext) {
	// do nothing
}

func (l *tealangListener) ExitIfStatementTrue(ctx *parser.IfStatementTrueContext) {
	labelSuffix := l.nestedCondStack[len(l.nestedCondStack)-1].labelID
	l.program.WriteString(fmt.Sprintf("intc %d\nbnz %s_end_%d\n", l.literals[trueConstValue].offset, ifExprPrefix, labelSuffix))
}

func (l *tealangListener) EnterIfStatementFalse(ctx *parser.IfStatementFalseContext) {
	state := l.nestedCondStack[len(l.nestedCondStack)-1]
	state.flags[condElsePresent] = true
	l.nestedCondStack[len(l.nestedCondStack)-1] = state

	labelSuffix := state.labelID
	l.program.WriteString(fmt.Sprintf("%s_false_%d:\n", ifExprPrefix, labelSuffix))
}

func (l *tealangListener) ExitIfStatementFalse(ctx *parser.IfStatementFalseContext) {
	// do nothing
}

/* Builtin variables */

func (l *tealangListener) ExitGlobalFieldExpr(ctx *parser.GlobalFieldExprContext) {
	field := ctx.GLOBALFIELD().GetSymbol().GetText()
	l.program.WriteString(fmt.Sprintf("global %s\n", field))
}

func (l *tealangListener) ExitTxnFieldExpr(ctx *parser.TxnFieldExprContext) {
	field := ctx.TXNFIELD().GetSymbol().GetText()
	l.program.WriteString(fmt.Sprintf("txn %s\n", field))
}

func (l *tealangListener) ExitArgsExpr(ctx *parser.ArgsExprContext) {
	index := ctx.NUMBER().GetSymbol().GetText()
	l.program.WriteString(fmt.Sprintf("arg %s\n", index))
}

func (l *tealangListener) ExitGroupTxnFieldExpr(ctx *parser.GroupTxnFieldExprContext) {
	index := ctx.NUMBER().GetSymbol().GetText()
	field := ctx.TXNFIELD().GetSymbol().GetText()
	l.program.WriteString(fmt.Sprintf("gtxn %s %s\n", index, field))
}

func (l *tealangListener) ExitBuiltinFunCall(ctx *parser.BuiltinFunCallContext) {
	funcName := ctx.BUILTINFUNC().GetSymbol().GetText()
	l.program.WriteString(fmt.Sprintf("%s\n", funcName))
}

/* Return and Error */

func (l *tealangListener) ExitTermReturn(ctx *parser.TermReturnContext) {
	l.program.WriteString(fmt.Sprintf("intc %d\nbnz %s\n", l.literals[trueConstValue].offset, endProgramLabel))
}

func (l *tealangListener) ExitTermError(ctx *parser.TermErrorContext) {
	l.program.WriteString(fmt.Sprintf("err\n"))
}

/* Emit */

func (l *tealangListener) Emit(ostream io.Writer) {
	if len(l.literals) != len(l.intc)+len(l.bytec) {
		panic("literals unbalanced")
	}
	if len(l.intc) > 0 {
		fmt.Fprintf(ostream, "intcblock ")
		for _, value := range l.intc {
			fmt.Fprintf(ostream, "%s ", value)
		}
		fmt.Fprintf(ostream, "\n")
	}

	if len(l.bytec) > 0 {
		fmt.Fprintf(ostream, "bytecblock ")
		for _, value := range l.bytec {
			// TODO: decode string (remove quotes, decode \x)
			fmt.Fprintf(ostream, "0x%s ", hex.EncodeToString(value))
		}
		fmt.Fprintf(ostream, "\n")
	}

	io.WriteString(ostream, l.program.String())
}

// Compile returns TEAL assembler code for Tealang source
func Compile(source string) string {
	is := antlr.NewInputStream(source)
	lexer := parser.NewTealangLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTealangParser(stream)
	p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	p.SetErrorHandler(antlr.NewBailErrorStrategy())
	p.BuildParseTrees = true
	tree := p.Program()

	listener := newTealangListener()
	antlr.ParseTreeWalkerDefault.Walk(&listener, tree)

	buf := new(b.Buffer)
	listener.Emit(buf)
	return buf.String()
}
