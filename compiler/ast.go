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

type literalInfo struct {
	intc  []string
	bytec [][]byte
}

type context struct {
	literals     *literalInfo
	parent       *context
	vars         map[string]varInfo
	addressEntry uint
	addressNext  uint
}

type varInfo struct {
	name     string
	theType  exprType
	constant bool

	// for variables specifies allocated memory space
	// for constants sets index in intc/bytec arrays
	address uint

	// constants have value
	value *string
}

func newLiteralInfo() (literals *literalInfo) {
	literals = new(literalInfo)
	literals.intc = make([]string, 0, 128)
	literals.bytec = make([][]byte, 0, 128)
	return
}

func newContext(parent *context) (ctx *context) {
	ctx = new(context)
	ctx.parent = parent
	ctx.vars = make(map[string]varInfo)
	if parent != nil {
		ctx.literals = parent.literals
		ctx.addressEntry = parent.addressNext
		ctx.addressNext = parent.addressNext
	} else {
		ctx.literals = newLiteralInfo()
		ctx.addressEntry = 0
		ctx.addressNext = 0
	}
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
	ctx.vars[name] = varInfo{name, theType, false, ctx.addressNext, nil}
	ctx.addressNext++
}

type exprType int

const (
	unknownType exprType = 0
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

// ExprNodeIf extends TreeNode and can be evaluated and typed
type ExprNodeIf interface {
	TreeNodeIf
	getType() exprType
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
	name string
	args []string
}

type blockNode struct {
	*TreeNode
}

type returnNode struct {
	*TreeNode
	value ExprNodeIf
}

type errorNode struct {
	*TreeNode
}

type assignNode struct {
	*TreeNode
	name     string
	exprType exprType
	value    ExprNodeIf
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

type ifStatementNode struct {
	*TreeNode
	condExpr ExprNodeIf
}

type funCallNode struct {
	*TreeNode
	name string
}

//--------------------------------------------------------------------------------------------------
//
// listeners
//
//--------------------------------------------------------------------------------------------------

type treeNodeListener struct {
	*gen.BaseTealangListener
	ctx  *context
	node TreeNodeIf
}

func (l *treeNodeListener) getNode() TreeNodeIf {
	return l.node
}

func newTreeNodeListener(ctx *context) *treeNodeListener {
	l := new(treeNodeListener)
	l.ctx = ctx
	return l
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

func newBlockNode(ctx *context) (node *blockNode) {
	node = new(blockNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "block"
	return
}

func newReturnNode(ctx *context, value ExprNodeIf) (node *returnNode) {
	node = new(returnNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "ret"
	node.value = value
	return
}

func newErorrNode(ctx *context) (node *errorNode) {
	node = new(errorNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "error"
	return
}

func newAssignNode(ctx *context, ident string, value ExprNodeIf) (node *assignNode) {
	node = new(assignNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "assign"
	node.name = ident
	node.value = value
	return
}

func newFunDeclNode(ctx *context) (node *funDeclNode) {
	node = new(funDeclNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "func"
	return
}

func newVarDeclNode(ctx *context, ident string, value ExprNodeIf) (node *varDeclNode) {
	node = new(varDeclNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "var"
	node.name = ident
	node.value = value
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

func newIfStatementNode(ctx *context) (node *ifStatementNode) {
	node = new(ifStatementNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "if stmt"
	return
}

func newFunCallNode(ctx *context, name string) (node *funCallNode) {
	node = new(funCallNode)
	node.TreeNode = newNode(ctx)
	node.nodeName = "fun call"
	node.name = name
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

func (n *funCallNode) getType() exprType {
	// TODO
	// 1. Access func definition
	// 2. Get return statement
	// 3. Eval its type
	return unknownType
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

func (n *varDeclNode) String() string {
	return fmt.Sprintf("var %s = %s", n.name, n.value)
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

func (n *returnNode) String() string {
	return fmt.Sprintf("return %s", n.value)
}

func (n *assignNode) String() string {
	return fmt.Sprintf("%s = %s", n.name, n.value)
}

func (n *ifStatementNode) String() string {
	return fmt.Sprintf("if %s", n.condExpr)
}

func (n *funCallNode) String() string {
	return fmt.Sprintf("%s (%v)", n.name, n.children())
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

// TypeCheck runs typechecking on the result
func (n *TreeNode) TypeCheck() (errors []TypeError) {
	for _, ch := range n.children() {
		errors = append(errors, ch.TypeCheck()...)
	}
	return
}

//--------------------------------------------------------------------------------------------------
//
// ANTLR callbacks
//
//--------------------------------------------------------------------------------------------------

// EnterProgram is an entry point to AST
func (l *treeNodeListener) EnterProgram(ctx *gen.ProgramContext) {
	root := newProgramNode(l.ctx)

	declarations := ctx.AllDeclaration()
	for _, declaration := range declarations {
		l := newTreeNodeListener(l.ctx)
		declaration.EnterRule(l)
		node := l.getNode()
		if node != nil {
			root.append(node)
		}
	}

	logicListener := newTreeNodeListener(l.ctx)
	ctx.Logic().EnterRule(logicListener)
	logic := logicListener.getNode()
	if logic == nil {
		// TODO: report error
		panic("no logic")
	}
	root.append(logic)

	l.node = root
}

func (l *treeNodeListener) EnterDeclaration(ctx *gen.DeclarationContext) {
	if decl := ctx.Decl(); decl != nil {
		decl.EnterRule(l)
	} else if fun := ctx.FUNC(); fun != nil {
		// start new scoped context
		scopedContext := newContext(l.ctx)
		name := ctx.IDENT(0).GetText()

		// get arguments vars
		argCount := len(ctx.AllIDENT())
		args := make([]string, argCount-1)
		for i := 0; i < argCount-1; i++ {
			ident := ctx.IDENT(i + 1).GetText()
			scopedContext.newVar(ident, unknownType)
			args[i] = ident
		}
		node := newFunDeclNode(scopedContext)
		node.name = name
		node.args = args

		// parse function body and add statements as children
		listener := newTreeNodeListener(scopedContext)
		ctx.Block().EnterRule(listener)
		blockNode := listener.getNode()
		for _, stmt := range blockNode.children() {
			node.append(stmt)
		}

		l.node = node
	}
}

func (l *treeNodeListener) EnterLogic(ctx *gen.LogicContext) {
	node := newFunDeclNode(l.ctx)
	node.name = "logic"
	node.args = []string{ctx.TXN().GetText(), ctx.GTXN().GetText(), ctx.ARGS().GetText()}

	scopedContext := newContext(l.ctx)
	listener := newTreeNodeListener(scopedContext)
	ctx.Block().EnterRule(listener)
	blockNode := listener.getNode()
	for _, stmt := range blockNode.children() {
		node.append(stmt)
	}

	l.node = node
}

func (l *treeNodeListener) EnterDeclareVar(ctx *gen.DeclareVarContext) {
	ident := ctx.IDENT().GetText()
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	l.ctx.newVar(ident, exprNode.getType())

	node := newVarDeclNode(l.ctx, ident, exprNode)
	l.node = node
}

func (l *treeNodeListener) EnterDeclareNumberConst(ctx *gen.DeclareNumberConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.NUMBER().GetText()

	node := newConstNode(l.ctx)
	node.name = varName
	node.value = varValue
	node.exprType = intType
	l.node = node
}

func (l *treeNodeListener) EnterDeclareStringConst(ctx *gen.DeclareStringConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.STRING().GetText()

	node := newConstNode(l.ctx)
	node.name = varName
	node.value = varValue
	node.exprType = bytesType
	l.node = node
}

func (l *treeNodeListener) EnterBlock(ctx *gen.BlockContext) {
	block := newBlockNode(l.ctx)
	statements := ctx.AllStatement()
	for _, declaration := range statements {
		l := newTreeNodeListener(l.ctx)
		declaration.EnterRule(l)
		node := l.getNode()
		if node != nil {
			block.append(node)
		}
	}
	l.node = block
}

func (l *treeNodeListener) EnterStatement(ctx *gen.StatementContext) {
	if ctx.Decl() != nil {
		ctx.Decl().EnterRule(l)
	} else if ctx.Condition() != nil {
		ctx.Condition().EnterRule(l)
	} else if ctx.Termination() != nil {
		ctx.Termination().EnterRule(l)
	} else if ctx.Assignment() != nil {
		ctx.Assignment().EnterRule(l)
	} else if ctx.Expr() != nil {
		listener := newExprListener(l.ctx)
		ctx.Expr().EnterRule(listener)
		l.node = listener.getExpr()
	}
}

func (l *treeNodeListener) EnterTermReturn(ctx *gen.TermReturnContext) {
	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	node := newReturnNode(l.ctx, listener.getExpr())
	l.node = node
}

func (l *treeNodeListener) EnterTermError(ctx *gen.TermErrorContext) {
	l.node = newErorrNode(l.ctx)
}

func (l *treeNodeListener) EnterIfStatement(ctx *gen.IfStatementContext) {
	node := newIfStatementNode(l.ctx)

	exprlistener := newExprListener(l.ctx)
	ctx.CondIfExpr().EnterRule(exprlistener)
	node.condExpr = exprlistener.getExpr()

	scopedContext := newContext(l.ctx)

	listener := newTreeNodeListener(scopedContext)
	ctx.CondTrueBlock().EnterRule(listener)
	node.append(listener.getNode())

	listener = newTreeNodeListener(scopedContext)
	if ctx.CondFalseBlock() != nil {
		ctx.CondFalseBlock().EnterRule(listener)
		node.append(listener.getNode())
	}
	l.node = node
}

func (l *treeNodeListener) EnterIfStatementTrue(ctx *gen.IfStatementTrueContext) {
	ctx.Block().EnterRule(l)
	blockNode := l.getNode()
	l.node = blockNode
}

func (l *treeNodeListener) EnterIfStatementFalse(ctx *gen.IfStatementFalseContext) {
	ctx.Block().EnterRule(l)
	blockNode := l.getNode()
	l.node = blockNode
}

func (l *treeNodeListener) EnterAssign(ctx *gen.AssignContext) {
	ident := ctx.IDENT().GetSymbol().GetText()
	// TODO: check declared

	listener := newExprListener(l.ctx)
	ctx.Expr().EnterRule(listener)
	node := newAssignNode(l.ctx, ident, listener.getExpr())
	l.node = node
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

func (l *exprListener) EnterFunctionCallExpr(ctx *gen.FunctionCallExprContext) {
	listener := newExprListener(l.ctx)
	ctx.FunctionCall().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterBuiltinFunCall(ctx *gen.BuiltinFunCallContext) {
	l.expr = l.funCallEnterImpl(ctx.BUILTINFUNC().GetText(), ctx.AllExpr())
}

func (l *exprListener) EnterFunCall(ctx *gen.FunCallContext) {
	l.expr = l.funCallEnterImpl(ctx.IDENT().GetText(), ctx.AllExpr())
}

func (l *exprListener) funCallEnterImpl(name string, allExpr []gen.IExprContext) (node *funCallNode) {
	node = newFunCallNode(l.ctx, name)
	for _, expr := range allExpr {
		listener := newExprListener(l.ctx)
		expr.EnterRule(listener)
		arg := listener.getExpr()
		node.append(arg)
	}
	return node
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

	ctx := newContext(nil)
	l := newTreeNodeListener(ctx)
	tree.EnterRule(l)

	prog := l.getNode()

	return prog, nil
}
