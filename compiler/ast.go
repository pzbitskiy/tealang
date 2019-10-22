package compiler

import (
	// "bytes"
	// "encoding/hex"
	"fmt"
	// "io"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	gen "../gen/go"
)

//go:generate ./bundle_langspec_json.sh

type context struct {
	literals map[string]literal // literal value -> index in intc / bytec
	intc     []string
	bytec    [][]byte

	constants     map[string]string // constant name -> value
	variables     map[string]uint
	variableIndex uint
}

type exprType int

const (
	intType   exprType = 1
	bytesType exprType = 2
)

// TreeNodeIf represents a node in AST
type TreeNodeIf interface {
	append(ch TreeNodeIf)
	empty() bool
	name() string
	children() []TreeNodeIf
	Print()
}

// TreeNode contains base info about an AST node
type TreeNode struct {
	*gen.BaseTealangListener
	parentCtx *context
	ctx       *context

	nodeName      string
	parent        TreeNodeIf
	childrenNodes []TreeNodeIf
}

type programNode struct {
	*TreeNode
}

type declarationNode struct {
	*TreeNode
	impl TreeNodeIf
}

type logicNode struct {
	*TreeNode
	funArgs []string
	block   blockNode
}

type funDeclarationNode struct {
	*TreeNode
	funName string
	funArgs []string
	block   blockNode
}

type blockNode struct {
	*TreeNode
}

type varDeclarationNode struct {
	*TreeNode
	varName  string
	varType  exprType
	varValue *exprNode
}

type constNode struct {
	*TreeNode
	varName  string
	varType  exprType
	varValue string
}

type exprNode struct {
	*TreeNode
	exprType exprType
}

func newNode(ctx *context) (node *TreeNode) {
	node = new(TreeNode)
	node.ctx = ctx
	node.childrenNodes = make([]TreeNodeIf, 0)
	return node
}

func newProgramNode(ctx *context) (node *programNode) {
	node = new(programNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "program"
	return
}

func newDeclarationNode(ctx *context) (node *declarationNode) {
	node = new(declarationNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "declaration"
	return
}

func newLogicNode(ctx *context) (node *logicNode) {
	node = new(logicNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "logic"
	node.funArgs = make([]string, 3)
	return
}

func newFunDeclarationNode(ctx *context) (node *funDeclarationNode) {
	node = new(funDeclarationNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "func"
	return
}

func newVarDeclarationNode(ctx *context) (node *varDeclarationNode) {
	node = new(varDeclarationNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "var"
	return
}

func newConstNode(ctx *context) (node *constNode) {
	node = new(constNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "const"
	return
}

func newExprNode(ctx *context) (node *exprNode) {
	node = new(exprNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "expr"
	return
}

func (n *TreeNode) append(ch TreeNodeIf) {
	n.childrenNodes = append(n.childrenNodes, ch)
}

func (n *TreeNode) empty() bool {
	return n.nodeName == "empty"
}

func (n *TreeNode) name() string {
	return n.nodeName
}

func (n *TreeNode) children() []TreeNodeIf {
	return n.childrenNodes
}

// Print AST
func (n *TreeNode) Print() {
	dumpImpl(n, 0)
}

func dumpImpl(n TreeNodeIf, offset int) {
	fmt.Printf("%s%s\n", strings.Repeat(" ", offset), n.name())
	for _, ch := range n.children() {
		dumpImpl(ch, offset+4)
	}
}

// EnterProgram is an entry point to AST
func (n *programNode) EnterProgram(ctx *gen.ProgramContext) {
	declarations := ctx.AllDeclaration()
	for _, declaration := range declarations {
		node := newDeclarationNode(n.ctx)
		declaration.EnterRule(node)
		// cast to actual type - variable/constant/function
		upgraded := node.upgrade()
		if upgraded != nil {
			n.append(upgraded)
		}
	}

	node := newLogicNode(n.ctx)
	ctx.Logic().EnterRule(node)
	n.append(node)
}

func (n *declarationNode) upgrade() (node TreeNodeIf) {
	return n.impl
}

func (n *declarationNode) EnterDeclaration(ctx *gen.DeclarationContext) {
	if decl := ctx.Decl(); decl != nil {
		decl.EnterRule(n)
	} else if fun := ctx.FUNC(); fun != nil {
		count := len(ctx.AllIDENT())
		name := ctx.IDENT(0).GetText()
		args := make([]string, count-1)
		for i := 0; i < count-1; i++ {
			args[i] = ctx.IDENT(i + 1).GetText()
		}

		actual := newFunDeclarationNode(n.ctx)
		actual.funName = name
		actual.funArgs = args
		n.impl = actual
		fmt.Printf("impl %v actual %v", n.impl, actual)
	} else {
		n.nodeName = "empty"
	}
}

func (n *logicNode) EnterLogic(ctx *gen.LogicContext) {
	n.funArgs = append(n.funArgs, ctx.TXN().GetText())
	n.funArgs = append(n.funArgs, ctx.GTXN().GetText())
	n.funArgs = append(n.funArgs, ctx.ACCOUNT().GetText())
}

func (n *declarationNode) EnterDeclareVar(ctx *gen.DeclareVarContext) {
	varName := ctx.IDENT().GetText()
	expr := ctx.Expr()
	node := newExprNode(n.ctx)
	expr.EnterRule(node)

	actual := newVarDeclarationNode(n.ctx)
	actual.varName = varName
	actual.varValue = node
	n.impl = actual
}

func (n *declarationNode) EnterDeclareNumberConst(ctx *gen.DeclareNumberConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.NUMBER().GetText()

	actual := newConstNode(n.ctx)
	actual.varName = varName
	actual.varValue = varValue
	actual.varType = intType
	n.impl = actual
}

func (n *declarationNode) EnterDeclareStringConst(ctx *gen.DeclareStringConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.STRING().GetText()

	actual := newConstNode(n.ctx)
	actual.varName = varName
	actual.varValue = varValue
	actual.varType = bytesType
	n.impl = actual
}

func (n *varDeclarationNode) name() string {
	return fmt.Sprintf("var %s", n.varName)
}

func (n *constNode) name() string {
	return fmt.Sprintf("const (%d) %s = %s", n.varType, n.varName, n.varValue)
}

func (n *funDeclarationNode) name() string {
	return fmt.Sprintf("function %s", n.funName)
}

// Parse creates AST
func Parse(source string) (TreeNodeIf, []ParserError) {
	is := antlr.NewInputStream(source)
	lexer := gen.NewTealangLexer(is)
	lexer.RemoveErrorListeners()
	collector := newErrorCollector(source)
	lexer.AddErrorListener(collector)

	tokenStream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := gen.NewTealangParser(tokenStream)

	parser.RemoveErrorListeners()
	parser.AddErrorListener(collector)
	parser.BuildParseTrees = true

	tree := parser.Program()
	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	ctx := new(context)
	prog := newProgramNode(ctx)
	tree.EnterRule(prog)

	return prog, nil
}
