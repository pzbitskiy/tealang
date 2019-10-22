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

//go:generate sh ./bundle_langspec_json.sh

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
	intType     exprType = 1
	bytesType   exprType = 2
	invalidType exprType = 99
)

func (n exprType) String() string {
	switch n {
	case intType:
		return "uint64"
	case bytesType:
		return "byte[]"
	}
	return "unknown"
}

// TreeNodeIf represents a node in AST
type TreeNodeIf interface {
	append(ch TreeNodeIf)
	empty() bool
	children() []TreeNodeIf
	String() string
	Print()
	TypeCheck() []TypeError
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

type funDeclNode struct {
	*TreeNode
	name  string
	args  []string
	block blockNode
}

type blockNode struct {
	*TreeNode
}

type varDeclNode struct {
	*TreeNode
	name     string
	exprType exprType
	value    ExprNodeIf
}

type constNode struct {
	*TreeNode
	name     string
	exprType exprType
	value    string
}

type exprIdentNode struct {
	*TreeNode
	exprType exprType
	name     string
}

type exprLiteralNode struct {
	*TreeNode
	exprType exprType
	value    string
}

type exprBinOpNode struct {
	*TreeNode
	exprType exprType
	op       string
	lhs      ExprNodeIf
	rhs      ExprNodeIf
}

type exprGroupNode struct {
	*TreeNode
	value ExprNodeIf
}

type ExprNodeIf interface {
	TreeNodeIf
	getType() exprType
}

//--------------------------------------------------------------------------------------------------
//
// listeners
//
//--------------------------------------------------------------------------------------------------

type genericListener struct {
	*gen.BaseTealangListener
	ctx  *context
	node TreeNodeIf
}

func (l *genericListener) getNode() TreeNodeIf {
	return l.node
}

func newGenericListener(ctx *context) *genericListener {
	l := new(genericListener)
	l.ctx = ctx
	return l
}

type declListener struct {
	*gen.BaseTealangListener
	ctx  *context
	decl TreeNodeIf
}

func newDeclListener(ctx *context) *declListener {
	l := new(declListener)
	l.ctx = ctx
	return l
}

func (l *declListener) getDecl() TreeNodeIf {
	return l.decl
}

type exprListener struct {
	*gen.BaseTealangListener
	ctx  *context
	expr ExprNodeIf
}

func newExprListener(ctx *context) *exprListener {
	l := new(exprListener)
	l.ctx = ctx
	return l
}

func (l *exprListener) getExpr() ExprNodeIf {
	return l.expr
}

//--------------------------------------------------------------------------------------------------
//
// AST nodes constructors
//
//--------------------------------------------------------------------------------------------------

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

func newFunDeclNode(ctx *context) (node *funDeclNode) {
	node = new(funDeclNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "func"
	return
}

func newvarDeclNode(ctx *context) (node *varDeclNode) {
	node = new(varDeclNode)
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

func newExprIdentNode(ctx *context, name string) (node *exprIdentNode) {
	node = new(exprIdentNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "expr ident"
	node.name = name
	return
}

func newExprLiteralNode(ctx *context, valType exprType, value string) (node *exprLiteralNode) {
	node = new(exprLiteralNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "expr ident"
	node.value = value
	node.exprType = valType
	return
}

func newExprBinOpNode(ctx *context, op string) (node *exprBinOpNode) {
	node = new(exprBinOpNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "expr OP expr"
	node.exprType = intType
	node.op = op
	return
}

func newExprGroupNode(ctx *context, value ExprNodeIf) (node *exprGroupNode) {
	node = new(exprGroupNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "(expr)"
	node.value = value
	return
}

func (n *exprLiteralNode) getType() exprType {
	return n.exprType
}

func (n *exprIdentNode) getType() exprType {
	return n.exprType
}

func (n *exprBinOpNode) getType() exprType {
	if tp, err := typeFromSpec(n.op); err != nil {
		return tp
	}
	return invalidType
}

func (n *exprGroupNode) getType() exprType {
	return n.value.getType()
}

func (n *TreeNode) append(ch TreeNodeIf) {
	n.childrenNodes = append(n.childrenNodes, ch)
}

func (n *TreeNode) empty() bool {
	return n.nodeName == "empty"
}

func (n *TreeNode) String() string {
	return n.nodeName
}

func (n *TreeNode) children() []TreeNodeIf {
	return n.childrenNodes
}

// Print AST
func (n *TreeNode) Print() {
	printImpl(n, 0)
}

func printImpl(n TreeNodeIf, offset int) {
	fmt.Printf("%s%s\n", strings.Repeat(" ", offset), n.String())
	for _, ch := range n.children() {
		printImpl(ch, offset+4)
	}
}

func (n *TreeNode) TypeCheck() (errors []TypeError) {
	for _, ch := range n.children() {
		errors = append(errors, ch.TypeCheck()...)
	}
	return
}

// EnterProgram is an entry point to AST
func (l *genericListener) EnterProgram(ctx *gen.ProgramContext) {
	root := newProgramNode(l.ctx)

	declarations := ctx.AllDeclaration()
	for _, declaration := range declarations {
		l := newDeclListener(l.ctx)
		declaration.EnterRule(l)
		node := l.getDecl()
		if node != nil {
			root.append(node)
		}
	}

	logicListener := newDeclListener(l.ctx)
	ctx.Logic().EnterRule(logicListener)
	logic := logicListener.getDecl()
	if logic == nil {
		// TODO: report error
		panic("no logic")
	}
	root.append(logic)

	l.node = root
}

func (l *declListener) EnterDeclaration(ctx *gen.DeclarationContext) {
	if decl := ctx.Decl(); decl != nil {
		decl.EnterRule(l)
	} else if fun := ctx.FUNC(); fun != nil {
		count := len(ctx.AllIDENT())
		name := ctx.IDENT(0).GetText()
		args := make([]string, count-1)
		for i := 0; i < count-1; i++ {
			args[i] = ctx.IDENT(i + 1).GetText()
		}
		node := newFunDeclNode(l.ctx)
		node.name = name
		node.args = args
		l.decl = node
	}
}

func (l *declListener) EnterLogic(ctx *gen.LogicContext) {
	node := newFunDeclNode(l.ctx)
	node.name = "logic"
	node.args = []string{ctx.TXN().GetText(), ctx.GTXN().GetText(), ctx.ACCOUNT().GetText()}
	l.decl = node
}

func (l *declListener) EnterDeclareVar(ctx *gen.DeclareVarContext) {
	varName := ctx.IDENT().GetText()
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	node := newvarDeclNode(l.ctx)
	node.name = varName
	node.value = exprNode
	l.decl = node
}

func (l *declListener) EnterDeclareNumberConst(ctx *gen.DeclareNumberConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.NUMBER().GetText()

	node := newConstNode(l.ctx)
	node.name = varName
	node.value = varValue
	node.exprType = intType
	l.decl = node
}

func (l *declListener) EnterDeclareStringConst(ctx *gen.DeclareStringConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.STRING().GetText()

	node := newConstNode(l.ctx)
	node.name = varName
	node.value = varValue
	node.exprType = bytesType
	l.decl = node
}

func (n *varDeclNode) String() string {
	return fmt.Sprintf("var %s = %s", n.name, n.value.String())
}

func (n *constNode) String() string {
	return fmt.Sprintf("const (%s) %s = %s", n.exprType, n.name, n.value)
}

func (n *funDeclNode) String() string {
	return fmt.Sprintf("function %s", n.name)
}

func (n *exprIdentNode) String() string {
	return fmt.Sprintf("var %s", n.name)
}

func (n *exprLiteralNode) String() string {
	return fmt.Sprintf("%s", n.value)
}

func (n *exprBinOpNode) String() string {
	return fmt.Sprintf("%s %s %s", n.lhs, n.op, n.rhs)
}

func (n *exprGroupNode) String() string {
	return fmt.Sprintf("(%s)", n.value)
}

func (n *exprBinOpNode) TypeCheck() (errors []TypeError) {
	errors = append(errors, n.lhs.TypeCheck()...)
	errors = append(errors, n.lhs.TypeCheck()...)

	lhs := n.lhs.getType()
	rhs := n.rhs.getType()
	if lhs != rhs {
		err := TypeError{fmt.Sprintf("mismatching types at '%s' expr", n)}
		errors = append(errors, err)
	}
	return
}

func (n *varDeclNode) TypeCheck() (errors []TypeError) {
	errors = n.value.TypeCheck()
	return
}

func (l *exprListener) EnterIdentifier(ctx *gen.IdentifierContext) {
	varName := ctx.IDENT().GetSymbol().GetText()
	node := newExprIdentNode(l.ctx, varName)
	l.expr = node
}

func (l *exprListener) EnterNumberLiteral(ctx *gen.NumberLiteralContext) {
	value := ctx.NUMBER().GetText()
	node := newExprLiteralNode(l.ctx, intType, value)
	l.expr = node
}

func (l *exprListener) EnterStringLiteral(ctx *gen.StringLiteralContext) {
	value := ctx.STRING().GetText()
	node := newExprLiteralNode(l.ctx, bytesType, value)
	l.expr = node
}

func (l *exprListener) binOp(op string, lhs gen.IExprContext, rhs gen.IExprContext) {

	node := newExprBinOpNode(l.ctx, op)

	subExprListener := newExprListener(l.ctx)
	lhs.EnterRule(subExprListener)
	node.lhs = subExprListener.getExpr()

	subExprListener = newExprListener(l.ctx)
	rhs.EnterRule(subExprListener)
	node.rhs = subExprListener.getExpr()

	l.expr = node
}

func (l *exprListener) EnterAddSub(ctx *gen.AddSubContext) {
	op := ctx.GetOp().GetText()
	l.binOp(op, ctx.Expr(0), ctx.Expr(1))
}

func (l *exprListener) EnterMulDivMod(ctx *gen.MulDivModContext) {
	op := ctx.GetOp().GetText()
	l.binOp(op, ctx.Expr(0), ctx.Expr(1))
}

func (l *exprListener) EnterRelation(ctx *gen.RelationContext) {
	op := ctx.GetOp().GetText()
	l.binOp(op, ctx.Expr(0), ctx.Expr(1))
}

func (l *exprListener) EnterBitOp(ctx *gen.BitOpContext) {
	op := ctx.GetOp().GetText()
	l.binOp(op, ctx.Expr(0), ctx.Expr(1))
}

func (l *exprListener) EnterAndOr(ctx *gen.AndOrContext) {
	op := ctx.GetOp().GetText()
	l.binOp(op, ctx.Expr(0), ctx.Expr(1))
}

func (l *exprListener) EnterGroup(ctx *gen.GroupContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	node := newExprGroupNode(l.ctx, listener.getExpr())
	l.expr = node
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
	l := newGenericListener(ctx)
	tree.EnterRule(l)

	prog := l.getNode()

	return prog, nil
}
