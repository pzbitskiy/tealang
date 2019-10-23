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

	vars          map[string]varInfo
	variableIndex uint
}

type varInfo struct {
	name    string
	theType exprType
	index   uint

	constant bool
	value    *string
}

func newContext() (ctx *context) {
	ctx = new(context)
	ctx.literals = make(map[string]literal)
	ctx.intc = make([]string, 0, 128)
	ctx.bytec = make([][]byte, 0, 128)
	ctx.vars = make(map[string]varInfo)
	ctx.variableIndex = 0
	return
}

func (ctx *context) lookup(name string) (varable varInfo, err error) {
	variable, ok := ctx.vars[name]
	if !ok {
		return varInfo{}, fmt.Errorf("ident %s not defined", name)
	}
	return variable, nil
}

func (ctx *context) newVar(name string, theType exprType) {
	ctx.vars[name] = varInfo{name, theType, ctx.variableIndex, false, nil}
	ctx.variableIndex++
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
	case invalidType:
		return "invalid"
	}
	return "unknown"
}

// TreeNodeIf represents a node in AST
type TreeNodeIf interface {
	append(ch TreeNodeIf)
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

type exprUnOpNode struct {
	*TreeNode
	op    string
	value ExprNodeIf
}

type ifExprNode struct {
	*TreeNode
	condExpr      ExprNodeIf
	condTrueExpr  ExprNodeIf
	condFalseExpr ExprNodeIf
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

func newExprIdentNode(ctx *context, name string, exprType exprType) (node *exprIdentNode) {
	node = new(exprIdentNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "expr ident"
	node.name = name
	node.exprType = exprType
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

func newExprUnOpNode(ctx *context, op string) (node *exprUnOpNode) {
	node = new(exprUnOpNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "OP expr"
	node.op = op
	return
}

func newIfExprNode(ctx *context) (node *ifExprNode) {
	node = new(ifExprNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "if expr"
	return
}

func (n *exprLiteralNode) getType() exprType {
	return n.exprType
}

func (n *exprIdentNode) getType() exprType {
	return n.exprType
}

func (n *exprBinOpNode) getType() exprType {
	tp, err := typeFromSpec(n.op)
	if err != nil {
		fmt.Println(err)
		return invalidType
	}

	return tp
}

func (n *exprUnOpNode) getType() exprType {
	tp, err := typeFromSpec(n.op)
	if err != nil {
		fmt.Println(err)
		return invalidType
	}

	return tp
}

func (n *ifExprNode) getType() exprType {
	return n.condTrueExpr.getType()
}

func (n *exprGroupNode) getType() exprType {
	return n.value.getType()
}

func (n *TreeNode) append(ch TreeNodeIf) {
	n.childrenNodes = append(n.childrenNodes, ch)
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
	ident := ctx.IDENT().GetText()
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	l.ctx.newVar(ident, exprNode.getType())

	node := newvarDeclNode(l.ctx)
	node.name = ident
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
	return fmt.Sprintf("ident %s", n.name)
}

func (n *exprLiteralNode) String() string {
	return fmt.Sprintf("%s", n.value)
}

func (n *exprBinOpNode) String() string {
	return fmt.Sprintf("%s %s %s", n.lhs, n.op, n.rhs)
}

func (n *exprUnOpNode) String() string {
	return fmt.Sprintf("%s %s", n.op, n.value)
}

func (n *exprGroupNode) String() string {
	return fmt.Sprintf("(%s)", n.value)
}

func (n *ifExprNode) String() string {
	return fmt.Sprintf("if %s { %s } else { %s }", n.condExpr, n.condTrueExpr, n.condFalseExpr)
}

func (n *exprBinOpNode) TypeCheck() (errors []TypeError) {
	errors = append(errors, n.lhs.TypeCheck()...)
	errors = append(errors, n.lhs.TypeCheck()...)

	lhs := n.lhs.getType()
	rhs := n.rhs.getType()
	if lhs != rhs {
		err := TypeError{fmt.Sprintf("types mismatch: %s %s %s in expr '%s'", lhs, n.op, rhs, n)}
		errors = append(errors, err)
	}
	return
}

func (n *varDeclNode) TypeCheck() (errors []TypeError) {
	errors = n.value.TypeCheck()
	return
}

func (n *ifExprNode) TypeCheck() (errors []TypeError) {
	errors = append(errors, n.condExpr.TypeCheck()...)
	errors = append(errors, n.condTrueExpr.TypeCheck()...)
	errors = append(errors, n.condFalseExpr.TypeCheck()...)

	condType := n.condExpr.getType()
	if condType != intType {
		err := TypeError{fmt.Sprintf("if cond: expected uint64, got %s", condType)}
		errors = append(errors, err)
	}

	condTrueExprType := n.condTrueExpr.getType()
	condFalseExprType := n.condFalseExpr.getType()
	if condTrueExprType != condFalseExprType {
		err := TypeError{fmt.Sprintf("if cond: different types: %s and %s", condTrueExprType, condFalseExprType)}
		errors = append(errors, err)
	}
	return
}

func (l *exprListener) EnterIdentifier(ctx *gen.IdentifierContext) {
	ident := ctx.IDENT().GetSymbol().GetText()
	variable, err := l.ctx.lookup(ident)
	if err != nil {
		// TODO: report error
		return
	}

	node := newExprIdentNode(l.ctx, ident, variable.theType)
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

func (l *exprListener) unOp(op string, expr gen.IExprContext) {

	node := newExprUnOpNode(l.ctx, op)

	subExprListener := newExprListener(l.ctx)
	expr.EnterRule(subExprListener)
	node.value = subExprListener.getExpr()

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

func (l *exprListener) EnterBitNot(ctx *gen.BitNotContext) {
	op := ctx.GetOp().GetText()
	l.unOp(op, ctx.Expr())
}

func (l *exprListener) EnterNot(ctx *gen.NotContext) {
	op := ctx.GetOp().GetText()
	l.unOp(op, ctx.Expr())
}

func (l *exprListener) EnterGroup(ctx *gen.GroupContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	node := newExprGroupNode(l.ctx, listener.getExpr())
	l.expr = node
}

func (l *exprListener) EnterIfExpr(ctx *gen.IfExprContext) {
	listener := newExprListener(l.ctx)
	ctx.CondExpr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterCondExpr(ctx *gen.CondExprContext) {
	node := newIfExprNode(l.ctx)

	listener := newExprListener(l.ctx)
	ctx.CondIfExpr().EnterRule(listener)
	node.condExpr = listener.getExpr()

	listener = newExprListener(l.ctx)
	ctx.CondTrueExpr().EnterRule(listener)
	node.condTrueExpr = listener.getExpr()

	listener = newExprListener(l.ctx)
	ctx.CondFalseExpr().EnterRule(listener)
	node.condFalseExpr = listener.getExpr()

	l.expr = node
}

func (l *exprListener) EnterIfExprCond(ctx *gen.IfExprCondContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterIfExprTrue(ctx *gen.IfExprTrueContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterIfExprFalse(ctx *gen.IfExprFalseContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

//--------------------------------------------------------------------------------------------------
//
// module API functions
//
//--------------------------------------------------------------------------------------------------

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

	ctx := newContext()
	l := newGenericListener(ctx)
	tree.EnterRule(l)

	prog := l.getNode()

	return prog, nil
}
