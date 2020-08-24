//--------------------------------------------------------------------------------------------------
//
// Antlr-based parser
//
//--------------------------------------------------------------------------------------------------

package compiler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"runtime/debug"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	gen "../gen/go"
)

//go:generate sh ./bundle_langspec_json.sh

//--------------------------------------------------------------------------------------------------
//
// Antlr event listeners
//
//--------------------------------------------------------------------------------------------------

var mainFuncName = "main"

type treeNodeListener struct {
	*gen.BaseTealangParserListener
	ctx      *context
	node     TreeNodeIf
	parent   TreeNodeIf
	parseCtx *parseContext
}

func (l *treeNodeListener) getNode() TreeNodeIf {
	return l.node
}

func newTreeNodeListener(ctx *context, parent TreeNodeIf) *treeNodeListener {
	l := new(treeNodeListener)
	l.ctx = ctx
	l.parent = parent
	return l
}

func newRootTreeNodeListener(ctx *context, parent TreeNodeIf, parseCtx *parseContext) *treeNodeListener {
	l := new(treeNodeListener)
	l.ctx = ctx
	l.parent = parent
	l.parseCtx = parseCtx
	return l
}

type exprListener struct {
	*gen.BaseTealangParserListener
	ctx    *context
	expr   ExprNodeIf
	parent TreeNodeIf
}

func newExprListener(ctx *context, parent TreeNodeIf) *exprListener {
	l := new(exprListener)
	l.ctx = ctx
	l.parent = parent
	return l
}

func (l *exprListener) getExpr() ExprNodeIf {
	return l.expr
}

type parseContext struct {
	input          InputDesc
	collector      *errorCollector
	moduleResolver func(moduleName string, sourceDir string, currentDir string) (InputDesc, error)
	loadedModules  map[string]TreeNodeIf
}

func newParseContext(input InputDesc, collector *errorCollector) (ctx *parseContext) {
	ctx = new(parseContext)
	ctx.input = input
	ctx.collector = collector
	ctx.loadedModules = make(map[string]TreeNodeIf)
	return
}

func reportError(msg string, parser antlr.Parser, token antlr.Token, rule antlr.RuleContext) {
	e := newTealangBaseRecognitionException(msg, parser, token, rule)
	parser.NotifyErrorListeners(e.GetMessage(), e.GetOffendingToken(), e)
}

func reportParserError(err ParserError, parser antlr.Parser, token antlr.Token, rule antlr.RuleContext) {
	e := newTealangParserErrorException(err, parser, token, rule)
	parser.NotifyErrorListeners(e.GetMessage(), e.GetOffendingToken(), e)
}

//--------------------------------------------------------------------------------------------------
//
// ANTLR callbacks
//
//--------------------------------------------------------------------------------------------------

// EnterProgram is an entry point to AST
func (l *treeNodeListener) EnterProgram(ctx *gen.ProgramContext) {
	root := newProgramNode(l.ctx, l.parent)

	declarations := ctx.AllDeclaration()
	for _, declaration := range declarations {
		l := newRootTreeNodeListener(l.ctx, root, l.parseCtx)
		declaration.EnterRule(l)
		node := l.getNode()
		if node != nil {
			root.append(node)
		}
	}

	mainListener := newTreeNodeListener(l.ctx, root)

	ctx.Main().EnterRule(mainListener)
	main := mainListener.getNode()
	mainCtx := ctx.Main().(*gen.MainContext)
	if main == nil {
		reportError(
			"missing main function",
			ctx.GetParser(), mainCtx.FUNC().GetSymbol(), mainCtx.GetRuleContext(),
		)
		return
	}

	if !ensureBlockReturns(main) {
		reportError(
			"main function does not return",
			ctx.GetParser(), mainCtx.FUNC().GetSymbol(), mainCtx.GetRuleContext(),
		)
		return
	}

	tp, err := determineBlockReturnType(main, []exprType{})
	if err != nil {
		reportError(
			err.Error(),
			ctx.GetParser(), mainCtx.FUNC().GetSymbol(), mainCtx.GetRuleContext(),
		)
		return
	}
	if tp != intType {
		reportError(
			fmt.Sprintf("main function must return int but got %s", tp),
			ctx.GetParser(), mainCtx.FUNC().GetSymbol(), mainCtx.GetRuleContext(),
		)
		return
	}

	root.append(main)

	l.node = root
}

// EnterModule is an entry point to AST
func (l *treeNodeListener) EnterModule(ctx *gen.ModuleContext) {
	root := newProgramNode(l.ctx, l.parent)

	declarations := ctx.AllDeclaration()
	for _, declaration := range declarations {
		l := newRootTreeNodeListener(l.ctx, root, l.parseCtx)
		declaration.EnterRule(l)
		node := l.getNode()
		if node != nil {
			root.append(node)
		}
	}

	l.node = root
}

func parseFunDeclarationImpl(l *treeNodeListener, callNode *funCallNode, ctx *gen.DeclarationContext) {
	// start new scoped context
	scopedContext := newContext(l.ctx)
	name := ctx.IDENT(0).GetText()

	// get arguments vars
	argCount := len(ctx.AllIDENT()) - 1
	args := make([]string, argCount)
	actualArgs := callNode.children()
	if len(args) != len(actualArgs) {
		reportError("mismatching argument(s)", ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext())
		return
	}

	for i := 0; i < argCount; i++ {
		ident := ctx.IDENT(i + 1).GetText()

		theType, err := actualArgs[i].(ExprNodeIf).getType()
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.IDENT(i+1).GetSymbol(), ctx.GetRuleContext())
			return
		}

		err = scopedContext.newVar(ident, theType)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.IDENT(i+1).GetSymbol(), ctx.GetRuleContext())
			return
		}

		args[i] = ident
	}
	node := newFunDefNode(scopedContext, l.parent)
	node.name = name
	node.args = args

	// parse function body and add statements as children
	listener := newTreeNodeListener(scopedContext, node)
	ctx.Block().EnterRule(listener)
	blockNode := listener.getNode()
	for _, stmt := range blockNode.children() {
		node.append(stmt)
	}

	l.node = node
}

func (l *treeNodeListener) EnterDeclaration(ctx *gen.DeclarationContext) {
	if decl := ctx.Decl(); decl != nil {
		decl.EnterRule(l)
	} else if fun := ctx.FUNC(); fun != nil {
		name := ctx.IDENT(0).GetText()
		// register now and parse it later just before the call
		defParserCb := func(listener *treeNodeListener, callNode *funCallNode) {
			parseFunDeclarationImpl(listener, callNode, ctx)
		}
		err := l.ctx.newFunc(name, unknownType, defParserCb)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.FUNC().GetSymbol(), ctx.GetRuleContext())
			return
		}
	} else if fun := ctx.IMPORT(); fun != nil {
		moduleName := ctx.MODULENAME().GetText()
		tree, err := parseModule(moduleName, l.parseCtx, l.parent, l.ctx)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.MODULENAME().GetSymbol(), ctx.GetRuleContext())
			return
		}
		if tree == nil {
			reportError(
				fmt.Sprintf("module %s parsing failed", moduleName),
				ctx.GetParser(), ctx.MODULENAME().GetSymbol(), ctx.GetRuleContext(),
			)
			return
		}
		// Modules contains only functions and constants
		// and these are registered in the context and are already in AST.
		// So only need to check that children nodes are constants and func defs
		for _, ch := range tree.children() {
			switch ch.(type) {
			case *constNode, *funDefNode:
				continue
			default:
				msg := fmt.Sprintf("module %s has %s but can only hold constants and functions", moduleName, ch.String())
				reportError(msg, ctx.GetParser(), ctx.FUNC().GetSymbol(), ctx.GetRuleContext())
			}
		}
	}
}

func (l *treeNodeListener) EnterMain(ctx *gen.MainContext) {
	scopedContext := newContext(l.ctx)

	node := newFunDefNode(scopedContext, l.parent)
	node.name = mainFuncName

	listener := newTreeNodeListener(scopedContext, node)
	ctx.Block().EnterRule(listener)
	blockNode := listener.getNode()
	for _, stmt := range blockNode.children() {
		node.append(stmt)
	}

	l.node = node
}

func (l *treeNodeListener) EnterDeclareVar(ctx *gen.DeclareVarContext) {
	ident := ctx.IDENT().GetText()
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	varType, err := exprNode.getType()
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	err = l.ctx.newVar(ident, varType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newVarDeclNode(l.ctx, l.parent, ident, exprNode)
	l.node = node
}

func (l *treeNodeListener) EnterDeclareVarTupleExpr(ctx *gen.DeclareVarTupleExprContext) {
	identHigh := ctx.IDENT(0).GetText()
	identLow := ctx.IDENT(1).GetText()
	listener := newExprListener(l.ctx, l.parent)
	ctx.TupleExpr().EnterRule(listener)
	exprNode := listener.getExpr()

	varType, err := exprNode.getType()
	if err != nil {
		reportError(
			err.Error(), ctx.GetParser(),
			ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}

	err = l.ctx.newVar(identLow, varType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext())
		return
	}
	err = l.ctx.newVar(identHigh, varType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newVarDeclTupleNode(l.ctx, l.parent, identLow, identHigh, exprNode)
	l.node = node
}

func (l *treeNodeListener) EnterDeclareNumberConst(ctx *gen.DeclareNumberConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.NUMBER().GetText()

	node := newConstNode(l.ctx, l.parent, varName, varValue, intType)
	err := l.ctx.newConst(varName, intType, &varValue)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}
	l.node = node
}

func (l *treeNodeListener) EnterDeclareStringConst(ctx *gen.DeclareStringConstContext) {
	varName := ctx.IDENT().GetText()
	varValue := ctx.STRING().GetText()

	node := newConstNode(l.ctx, l.parent, varName, varValue, bytesType)
	err := l.ctx.newConst(varName, bytesType, &varValue)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}
	l.node = node
}

func (l *treeNodeListener) EnterBlock(ctx *gen.BlockContext) {
	block := newBlockNode(l.ctx, l.parent)
	statements := ctx.AllStatement()
	for _, stmt := range statements {
		l := newTreeNodeListener(l.ctx, block)
		stmt.EnterRule(l)
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
	}
}

func (l *treeNodeListener) EnterTermReturn(ctx *gen.TermReturnContext) {
	node := newReturnNode(l.ctx, l.parent)
	listener := newExprListener(l.ctx, node)
	ctx.Expr().EnterRule(listener)
	node.value = listener.getExpr()
	l.node = node

	parent := node.parent()
	enclosingFun := ""
	for parent != nil && enclosingFun == "" {
		switch tt := parent.(type) {
		case *funDefNode:
			enclosingFun = tt.name
			break
		}
		parent = parent.parent()
	}

	if enclosingFun == "" {
		reportError(
			"return without enclosing function",
			ctx.GetParser(), ctx.RET().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	node.enclosingFun = enclosingFun
}

func (l *treeNodeListener) EnterTermError(ctx *gen.TermErrorContext) {
	l.node = newErorrNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterIfStatement(ctx *gen.IfStatementContext) {
	node := newIfStatementNode(l.ctx, l.parent)

	exprlistener := newExprListener(l.ctx, node)
	ctx.CondIfExpr().EnterRule(exprlistener)
	node.condExpr = exprlistener.getExpr()

	scopedContextTrue := newContext(l.ctx)

	listener := newTreeNodeListener(scopedContextTrue, node)
	ctx.CondTrueBlock().EnterRule(listener)
	node.append(listener.getNode())

	scopedContextFalse := newContext(l.ctx)
	listener = newTreeNodeListener(scopedContextFalse, node)
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

func getVarInfoForAssignment(ident string, ctx *context) (varInfo, error) {
	info, err := ctx.lookup(ident)
	if err != nil {
		return varInfo{}, err
	}
	if info.constant {
		return varInfo{}, fmt.Errorf("cannot assign to a constant")
	}

	if info.function {
		return varInfo{}, fmt.Errorf("cannot assign to a function")
	}

	return info, nil
}

func (l *treeNodeListener) EnterAssign(ctx *gen.AssignContext) {
	ident := ctx.IDENT().GetSymbol().GetText()
	info, err := getVarInfoForAssignment(ident, l.ctx)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newAssignNode(l.ctx, l.parent, ident)
	listener := newExprListener(l.ctx, node)
	ctx.Expr().EnterRule(listener)
	rhs := listener.getExpr()
	node.value = rhs
	rhsType, err := rhs.getType()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if info.theType != rhsType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", info.theType, rhsType),
			ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	l.node = node
}

func (l *treeNodeListener) EnterAssignTuple(ctx *gen.AssignTupleContext) {
	identHigh := ctx.IDENT(0).GetSymbol().GetText()
	infoHigh, err := getVarInfoForAssignment(identHigh, l.ctx)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext())
		return
	}

	identLow := ctx.IDENT(1).GetSymbol().GetText()
	infoLow, err := getVarInfoForAssignment(identLow, l.ctx)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newAssignTupleNode(l.ctx, l.parent, identLow, identHigh)
	listener := newExprListener(l.ctx, node)
	ctx.TupleExpr().EnterRule(listener)
	rhs := listener.getExpr()
	node.value = rhs
	rhsType, err := rhs.getType()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}
	if infoHigh.theType != rhsType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoHigh.theType, rhsType),
			ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if infoLow.theType != rhsType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoLow.theType, rhsType),
			ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	l.node = node
}

func (l *exprListener) EnterIdentifier(ctx *gen.IdentifierContext) {
	ident := ctx.IDENT().GetSymbol().GetText()
	variable, err := l.ctx.lookup(ident)
	if err != nil {
		reportError("ident not found", ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newExprIdentNode(l.ctx, l.parent, ident, variable.theType)
	l.expr = node
}

func (l *exprListener) EnterNumberLiteral(ctx *gen.NumberLiteralContext) {
	value := ctx.NUMBER().GetText()
	node := newExprLiteralNode(l.ctx, l.parent, intType, value)
	_, err := l.ctx.addLiteral(value, intType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.NUMBER().GetSymbol(), ctx.GetRuleContext())
		return
	}
	l.expr = node
}

func (l *exprListener) EnterStringLiteral(ctx *gen.StringLiteralContext) {
	value := ctx.STRING().GetText()
	node := newExprLiteralNode(l.ctx, l.parent, bytesType, value)
	_, err := l.ctx.addLiteral(value, bytesType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.STRING().GetSymbol(), ctx.GetRuleContext())
		return
	}
	l.expr = node
}

func (l *exprListener) binOp(op string, lhs gen.IExprContext, rhs gen.IExprContext) {

	node := newExprBinOpNode(l.ctx, l.parent, op)

	subExprListener := newExprListener(l.ctx, node)
	lhs.EnterRule(subExprListener)
	node.lhs = subExprListener.getExpr()

	subExprListener = newExprListener(l.ctx, node)
	rhs.EnterRule(subExprListener)
	node.rhs = subExprListener.getExpr()

	l.expr = node
}

func (l *exprListener) unOp(op string, expr gen.IExprContext) {

	node := newExprUnOpNode(l.ctx, l.parent, op)

	subExprListener := newExprListener(l.ctx, node)
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
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	node := newExprGroupNode(l.ctx, l.parent, listener.getExpr())
	l.expr = node
}

func (l *exprListener) EnterIfExpr(ctx *gen.IfExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.CondExpr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterCondExpr(ctx *gen.CondExprContext) {
	node := newIfExprNode(l.ctx, l.parent)

	listener := newExprListener(l.ctx, node)
	ctx.CondIfExpr().EnterRule(listener)
	node.condExpr = listener.getExpr()

	listener = newExprListener(l.ctx, node)
	ctx.CondTrueExpr().EnterRule(listener)
	node.condTrueExpr = listener.getExpr()

	listener = newExprListener(l.ctx, node)
	ctx.CondFalseExpr().EnterRule(listener)
	node.condFalseExpr = listener.getExpr()

	l.expr = node
}

func (l *exprListener) EnterIfExprCond(ctx *gen.IfExprCondContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterIfExprTrue(ctx *gen.IfExprTrueContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterIfExprFalse(ctx *gen.IfExprFalseContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterFunctionCallExpr(ctx *gen.FunctionCallExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.FunctionCall().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterBuiltinFunCall(ctx *gen.BuiltinFunCallContext) {
	name := ctx.BUILTINFUNC().GetText()
	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())

	err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.BUILTINFUNC().GetSymbol()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.expr = exprNode
}

func (l *exprListener) EnterFunCall(ctx *gen.FunCallContext) {
	name := ctx.IDENT().GetText()
	parser := ctx.GetParser()
	token := ctx.IDENT().GetSymbol()
	rule := ctx.GetRuleContext()
	info, err := l.ctx.lookup(name)
	if err != nil {
		reportError(err.Error(), parser, token, rule)
		return
	}
	if !info.function {
		reportError("not a function", parser, token, rule)
		return
	}

	argExprNodes := ctx.AllExpr()
	funCallExprNode := l.funCallEnterImpl(name, argExprNodes)
	listener := newTreeNodeListener(l.ctx, funCallExprNode)
	info.parser(listener, funCallExprNode)
	if listener.node == nil {
		reportError("function parsing failed", parser, token, rule)
		return
	}
	defNode := listener.node.(*funDefNode)

	if !ensureBlockReturns(defNode) {
		reportError(
			fmt.Sprintf("%s function does not return", name),
			parser, token, rule,
		)
		return
	}

	if err != nil {
		reportError(err.Error(), parser, token, rule)
		return
	}

	funCallExprNode.definition = defNode
	l.expr = funCallExprNode
}

func (l *exprListener) funCallEnterImpl(name string, allExpr []gen.IExprContext) (node *funCallNode) {
	node = newFunCallNode(l.ctx, l.parent, name)
	for _, expr := range allExpr {
		listener := newExprListener(l.ctx, node)
		expr.EnterRule(listener)
		arg := listener.getExpr()
		node.append(arg)
	}
	return node
}

func (l *exprListener) EnterTupleExpr(ctx *gen.TupleExprContext) {
	// var node antlr.TerminalNode
	var name string
	if node := ctx.MULW(); node != nil {
		name = node.GetText()
	} else if node := ctx.ADDW(); node != nil {
		name = node.GetText()
	} else if node := ctx.APPLOCALGETEX(); node != nil {
		name = node.GetText()
	} else if node := ctx.APPGLOBALGETEX(); node != nil {
		name = node.GetText()
	} else if node := ctx.ASSETHOLDINGGET(); node != nil {
		name = node.GetText()
	}

	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())

	err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.GetParser().GetCurrentToken()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.expr = exprNode
}

func (l *exprListener) EnterBuiltinObject(ctx *gen.BuiltinObjectContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.BuiltinVarExpr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterGlobalFieldExpr(ctx *gen.GlobalFieldExprContext) {
	field := ctx.GLOBALFIELD().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "global", field, "")
	l.expr = node
}

func (l *exprListener) EnterTxnFieldExpr(ctx *gen.TxnFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "txn", field, "")
	l.expr = node
}

func (l *exprListener) EnterGroupTxnFieldExpr(ctx *gen.GroupTxnFieldExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Gtxn().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterGroupNumberTxnFieldExpr(ctx *gen.GroupNumberTxnFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	groupIndex := ctx.NUMBER().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "gtxn", field, groupIndex)
	l.expr = node
}

func (l *exprListener) EnterGroupIdentTxnFieldExpr(ctx *gen.GroupIdentTxnFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	ident := ctx.IDENT().GetText()

	info, err := l.ctx.lookup(ident)
	if err != nil || !info.constant {
		reportError(fmt.Sprintf("%s not a constant", ident), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newRuntimeFieldNode(l.ctx, l.parent, "gtxn", field, *info.value)
	l.expr = node
}

func (l *exprListener) EnterArgsExpr(ctx *gen.ArgsExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Args().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterArgsNumberExpr(ctx *gen.ArgsNumberExprContext) {
	number := ctx.NUMBER().GetText()
	node := newRuntimeArgNode(l.ctx, l.parent, "arg", number)
	l.expr = node
}

func (l *exprListener) EnterArgsIdentExpr(ctx *gen.ArgsIdentExprContext) {
	ident := ctx.IDENT().GetText()

	info, err := l.ctx.lookup(ident)
	if err != nil || !info.constant {
		reportError(fmt.Sprintf("%s not a constant", ident), ctx.GetParser(), ctx.IDENT().GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newRuntimeArgNode(l.ctx, l.parent, "arg", *info.value)
	l.expr = node
}

func (l *treeNodeListener) EnterOnelinecond(ctx *gen.OnelinecondContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	expr := listener.getExpr()
	_, err := expr.getType() // trigger type evaluation
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.Expr().GetStart(), ctx.GetRuleContext())
		return
	}

	root := newProgramNode(l.ctx, l.parent)
	root.append(expr)
	l.node = root
}

func newParser(source string, collector *errorCollector) *gen.TealangParser {
	is := antlr.NewInputStream(source)
	lexer := gen.NewTealangLexer(is)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(collector)

	tokenStream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := gen.NewTealangParser(tokenStream)

	parser.RemoveErrorListeners()
	parser.AddErrorListener(collector)
	parser.BuildParseTrees = true

	return parser

}

//--------------------------------------------------------------------------------------------------
//
// module API functions
//
//--------------------------------------------------------------------------------------------------

// InputDesc struct describe location of the source file
// This info is later used for imports
type InputDesc struct {
	Source     string
	SourceFile string
	SourceDir  string
	CurrentDir string
}

func parseModule(moduleName string, parseCtx *parseContext, parent TreeNodeIf, ctx *context) (TreeNodeIf, error) {
	resolver := resolveModule
	if parseCtx.moduleResolver != nil {
		resolver = parseCtx.moduleResolver
	}
	input, err := resolver(moduleName, parseCtx.input.SourceDir, parseCtx.input.CurrentDir)
	if err != nil {
		return nil, err
	}

	raw := md5.Sum([]byte(input.Source))
	checksum := hex.EncodeToString(raw[:])
	if tree, ok := parseCtx.loadedModules[checksum]; ok {
		return tree, nil
	}

	collector := newErrorCollector(input.Source, input.SourceFile)
	parser := newParser(input.Source, collector)

	tree := parser.Module()

	collector.filterAmbiguity()
	if len(collector.errors) > 0 {
		parseCtx.collector.copyErrors(collector)
		return nil, fmt.Errorf("error during module %s parsing", moduleName)
	}

	l := newRootTreeNodeListener(ctx, parent, parseCtx)

	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(collector.errors) == 0 {
					fmt.Printf("unexpected error: %s\n", r)
					fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				}
			}
		}()
		tree.EnterRule(l)
	}()

	parseCtx.collector.copyErrors(collector)
	if len(collector.errors) > 0 {
		return nil, fmt.Errorf("error during module %s parsing", moduleName)
	}

	mod := l.getNode()
	parseCtx.loadedModules[checksum] = mod
	return mod, nil
}

// ParseProgram accepts InputDesc that describes source location
func ParseProgram(input InputDesc) (TreeNodeIf, []ParserError) {
	collector := newErrorCollector(input.Source, input.SourceFile)
	parser := newParser(input.Source, collector)

	tree := parser.Program()

	collector.filterAmbiguity()
	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	ctx := newContext(nil)

	parseCtx := newParseContext(input, collector)
	l := newRootTreeNodeListener(ctx, nil, parseCtx)

	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(collector.errors) == 0 {
					fmt.Printf("unexpected error: %s\n", r)
					fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				}
			}
		}()
		tree.EnterRule(l)
	}()

	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	prog := l.getNode()
	return prog, nil
}

// Parse function creates AST
func Parse(source string) (TreeNodeIf, []ParserError) {
	input := InputDesc{source, "", "", ""}
	return ParseProgram(input)
}

func parseTestProgModule(progSource, moduleSource string) (TreeNodeIf, []ParserError) {
	input := InputDesc{progSource, "test.tl", "", ""}
	collector := newErrorCollector(input.Source, input.SourceFile)
	parser := newParser(progSource, collector)

	tree := parser.Program()

	collector.filterAmbiguity()
	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	ctx := newContext(nil)
	parseCtx := newParseContext(input, collector)
	parseCtx.moduleResolver = func(moduleName string, sourceDir string, currentDir string) (InputDesc, error) {
		input := InputDesc{moduleSource, moduleName, "", ""}
		return input, nil
	}
	l := newRootTreeNodeListener(ctx, nil, parseCtx)

	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(collector.errors) == 0 {
					fmt.Printf("unexpected error: %s\n", r)
					fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				}
			}
		}()
		tree.EnterRule(l)
	}()

	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	prog := l.getNode()
	return prog, nil

}

// ParseOneLineCond is for parsing one-liners like "(txn.fee == 1) && (global.MinTxnFee < 2000)"
func ParseOneLineCond(source string) (TreeNodeIf, []ParserError) {
	input := InputDesc{source, "", "", ""}
	collector := newErrorCollector(input.Source, input.SourceFile)
	parser := newParser(input.Source, collector)

	tree := parser.Onelinecond()

	collector.filterAmbiguity()
	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	ctx := newContext(nil)
	parseCtx := newParseContext(input, collector)
	l := newRootTreeNodeListener(ctx, nil, parseCtx)

	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(collector.errors) == 0 {
					fmt.Printf("unexpected error: %s\n", r)
					fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				}
			}
		}()
		tree.EnterRule(l)
	}()

	if len(collector.errors) > 0 {
		return nil, collector.errors
	}

	expr := l.getNode()
	return expr, nil
}
