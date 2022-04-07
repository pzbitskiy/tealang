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
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	gen "github.com/pzbitskiy/tealang/gen/go"
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
	if tp != unknownType && tp != intType {
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

func parseFunDeclarationImpl(l *treeNodeListener, callNode *funCallNode, ctx *gen.DeclarationContext, inline bool) {
	// start new scoped context
	name := ctx.IDENT(0).GetText()
	scopedContext := newContext(name, l.ctx)

	// get arguments vars
	argCount := len(ctx.AllIDENT()) - 1
	args := make([]funArg, argCount)
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

		// arguments are variables in a new scope
		// for inline functions they are set when calling
		// for regular functions they re popped from the stack inside a function
		err = scopedContext.newVar(ident, theType)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.IDENT(i+1).GetSymbol(), ctx.GetRuleContext())
			return
		}
		args[i] = funArg{ident, theType}
	}
	node := newFunDefNode(scopedContext, l.parent)
	node.name = name
	node.args = args
	node.inline = inline

	// parse function body and add statements as children
	listener := newTreeNodeListener(scopedContext, node)
	ctx.Block().EnterRule(listener)
	blockNode := listener.getNode()
	for _, stmt := range blockNode.children() {
		node.append(stmt)
	}
	l.node = node

	ctx.Block().ExitRule(listener)
}

func (l *treeNodeListener) EnterDeclaration(ctx *gen.DeclarationContext) {
	if decl := ctx.Decl(); decl != nil {
		decl.EnterRule(l)
	} else if fun := ctx.FUNC(); fun != nil {
		name := ctx.IDENT(0).GetText()
		inline := false
		if ctx.INLINE() != nil {
			inline = true
		}
		// register now and parse it later just before the call
		defParserCb := func(context *context, callNode *funCallNode, vi *varInfo) *funDefNode {
			if inline || vi.node == nil {
				listener := newTreeNodeListener(context, callNode)
				parseFunDeclarationImpl(listener, callNode, ctx, inline)
				node := listener.node
				if node == nil {
					return nil
				}
				vi.node = node
				return node.(*funDefNode)
			}
			// otherwise fixup internal variable indices
			// the trick is to use scratch space slots that are not used yet
			// in order to guarantee function args do not shadow global/main/parent variables.
			// non-inline functions use stack for passing arguments, so remapping forward internal vars
			// on invocation is safe and does not cause any problems because the func ends up using
			// maximal addresses accross all invocations.
			// the only problem is it requires remapping all way up to nested functions.
			defNode := vi.node.(*funDefNode)
			thisCtx := defNode.ctx
			parentCtx := callNode.ctx
			if thisCtx.EntryAddress() < parentCtx.LastAddress() {
				thisCtx.remapTo(parentCtx.LastAddress())

				var remapRec func(defNode *funDefNode)
				remapRec = func(defNode *funDefNode) {
					subfuncs := defNode.ctx.functions
					for _, funCallExprNode := range subfuncs {
						thisCtx := funCallExprNode.definition.ctx
						parentCtx := thisCtx.parent
						if thisCtx.EntryAddress() < parentCtx.LastAddress() {
							thisCtx.remapTo(parentCtx.LastAddress())
						}
						remapRec(funCallExprNode.definition)
					}
				}

				remapRec(defNode)
			}
			return defNode
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
	scopedContext := newContext("main", l.ctx)

	node := newFunDefNode(scopedContext, l.parent)
	node.name = mainFuncName

	listener := newTreeNodeListener(scopedContext, node)
	ctx.Block().EnterRule(listener)
	blockNode := listener.getNode()
	for _, stmt := range blockNode.children() {
		node.append(stmt)
	}
	l.node = node
	ctx.Block().ExitRule(listener)
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

	hType, lType, err := exprNode.(*funCallNode).getTypeTuple()
	if err != nil {
		reportError(
			err.Error(), ctx.GetParser(),
			ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}

	err = l.ctx.newVar(identLow, lType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext())
		return
	}
	err = l.ctx.newVar(identHigh, hType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext())
		return
	}

	remhighctx := ctx.IDENT(2)
	remlowctx := ctx.IDENT(3)
	if remlowctx != nil && remhighctx != nil {
		remhigh := remhighctx.GetText()
		remlow := remlowctx.GetText()

		err = l.ctx.newVar(remlow, lType)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.IDENT(3).GetSymbol(), ctx.GetRuleContext())
			return
		}
		err = l.ctx.newVar(remhigh, hType)
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), ctx.IDENT(2).GetSymbol(), ctx.GetRuleContext())
			return
		}
		node := newVarDeclDivmodwTupleNode(l.ctx, l.parent, identLow, identHigh, remlow, remhigh, exprNode)
		l.node = node
	} else {
		node := newVarDeclTupleNode(l.ctx, l.parent, identLow, identHigh, exprNode)
		l.node = node
	}

}

func (l *treeNodeListener) EnterDeclareQuadrupleExpr(ctx *gen.DeclareQuadrupleExprContext) {
	identHigh := ctx.IDENT(0).GetText()
	identLow := ctx.IDENT(1).GetText()
	remHigh := ctx.IDENT(2).GetText()
	remLow := ctx.IDENT(3).GetText()

	listener := newExprListener(l.ctx, l.parent)
	ctx.TupleExpr().EnterRule(listener)
	exprNode := listener.getExpr()

	hType, lType, rhType, rlType, err := exprNode.(*funCallNode).getTypeQuadruple()
	if err != nil {
		reportError(
			err.Error(), ctx.GetParser(),
			ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}

	err = l.ctx.newVar(identLow, lType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext())
		return
	}
	err = l.ctx.newVar(identHigh, hType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext())
		return
	}
	err = l.ctx.newVar(remLow, rlType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(3).GetSymbol(), ctx.GetRuleContext())
		return
	}
	err = l.ctx.newVar(remHigh, rhType)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(2).GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newVarDeclDivmodwTupleNode(l.ctx, l.parent, identLow, identHigh, remLow, remHigh, exprNode)
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
		stmt.ExitRule(l)
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
	} else if ctx.BuiltinVarStatement() != nil {
		ctx.BuiltinVarStatement().EnterRule(l)
	} else if ctx.LogStatement() != nil {
		ctx.LogStatement().EnterRule(l)
	} else if ctx.Innertxn() != nil {
		ctx.Innertxn().EnterRule(l)
	}
}

func (l *treeNodeListener) EnterTermReturn(ctx *gen.TermReturnContext) {
	node := newReturnNode(l.ctx, l.parent)
	listener := newExprListener(l.ctx, node)
	ctx.Expr().EnterRule(listener)
	node.value = listener.getExpr()
	l.node = node

	parent := node.parent()
	var definition *funDefNode
	for parent != nil && definition == nil {
		switch tt := parent.(type) {
		case *funDefNode:
			definition = tt
			break
		}
		parent = parent.parent()
	}

	if definition == nil {
		reportError(
			"return without enclosing function",
			ctx.GetParser(), ctx.RET().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	node.definition = definition
}

func (l *treeNodeListener) EnterTermError(ctx *gen.TermErrorContext) {
	l.node = newErorrNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterTermAssert(ctx *gen.TermAssertContext) {
	name := ctx.ASSERT().GetText()

	listener := newExprListener(l.ctx, l.parent)
	exprNode := listener.funCallEnterImpl(name, []gen.IExprContext{ctx.Expr()})

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.Expr().GetStart()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.node = exprNode
}

func (l *treeNodeListener) EnterInnerTxnAssign(ctx *gen.InnerTxnAssignContext) {
	field := ctx.TXNFIELD().GetText()
	node := newAssignInnerTxnNode(l.ctx, l.parent, field)
	listener := newExprListener(l.ctx, node)
	ctx.Expr().EnterRule(listener)
	rhs := listener.getExpr()
	node.value = rhs
	rhsType, err := rhs.getType()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.TXNFIELD().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	exprType, err := runtimeFieldTypeFromSpec("txn", field)
	if err != nil {
		reportError(
			fmt.Sprintf("failed to retrieve type of field %s: %s", field, err.Error()),
			ctx.GetParser(), ctx.TXNFIELD().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if exprType != rhsType {
		reportError(
			fmt.Sprintf("incompatible types: (lhs) %s vs %s (expr)", exprType, rhsType),
			ctx.GetParser(), ctx.TXNFIELD().GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	l.node = node
}

func (l *treeNodeListener) EnterInnerTxnArrayAssign(ctx *gen.InnerTxnArrayAssignContext) {
	field := ctx.TXNARRAYFIELD().GetText()
	node := newArrayAssignInnerTxnNode(l.ctx, l.parent, field)
	listener := newExprListener(l.ctx, node)
	ctx.Expr().EnterRule(listener)
	exprToPush := listener.getExpr()
	exprToPushType, err := exprToPush.getType()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.Expr().GetStart(), ctx.GetRuleContext(),
		)
		return
	}
	exprType, err := runtimeFieldTypeFromSpec("txn", field)
	if err != nil {
		reportError(
			fmt.Sprintf("failed to retrieve type of field %s: %s", field, err.Error()),
			ctx.GetParser(), ctx.Expr().GetStart(), ctx.GetRuleContext(),
		)
		return
	}
	if exprType != exprToPushType {
		reportError(
			fmt.Sprintf("incompatible types: (lhs) %s vs %s (expr)", exprType, exprToPushType),
			ctx.GetParser(), ctx.Expr().GetStart(), ctx.GetRuleContext(),
		)
		return
	}
	node.append(exprToPush)
	l.node = node
}

func (l *treeNodeListener) EnterInnerTxnBegin(ctx *gen.InnerTxnBeginContext) {
	l.node = newInnertxnBeginNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterInnerTxnNext(ctx *gen.InnerTxnNextContext) {
	l.node = newInnertxnNextNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterInnerTxnEnd(ctx *gen.InnerTxnEndContext) {
	l.node = newInnertxnEndNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterDoLog(ctx *gen.DoLogContext) {
	name := ctx.LOG().GetText()

	listener := newExprListener(l.ctx, l.parent)
	exprNode := listener.funCallEnterImpl(name, []gen.IExprContext{ctx.Expr()})

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.Expr().GetStart()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.node = exprNode
}

func (l *treeNodeListener) EnterBreak(ctx *gen.BreakContext) {
	l.node = newBreakNode(l.ctx, l.parent)
}

func (l *treeNodeListener) EnterIfStatement(ctx *gen.IfStatementContext) {
	node := newIfStatementNode(l.ctx, l.parent)

	exprlistener := newExprListener(l.ctx, node)
	ctx.CondIfExpr().EnterRule(exprlistener)
	node.condExpr = exprlistener.getExpr()

	scopedContextTrue := newContext("if", l.ctx)

	listener := newTreeNodeListener(scopedContextTrue, node)
	ctx.CondTrueBlock().EnterRule(listener)
	node.append(listener.getNode())

	scopedContextFalse := newContext("else", l.ctx)
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
	if info.constant() {
		return varInfo{}, fmt.Errorf("cannot assign to a constant")
	}

	if info.function() {
		return varInfo{}, fmt.Errorf("cannot assign to a function")
	}

	return info, nil
}

func (l *treeNodeListener) EnterForStatement(ctx *gen.ForStatementContext) {
	node := newForStatementNode(l.ctx, l.parent)

	exprlistener := newExprListener(l.ctx, node)
	ctx.CondForExpr().EnterRule(exprlistener)
	node.condExpr = exprlistener.getExpr()

	scopedContextTrue := newContext("for", l.ctx)

	listener := newTreeNodeListener(scopedContextTrue, node)
	ctx.CondTrueBlock().EnterRule(listener)
	node.append(listener.getNode())
	l.node = node
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
	hType, lType, err := rhs.(*funCallNode).getTypeTuple()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}
	if infoHigh.theType != hType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoHigh.theType, hType),
			ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if infoLow.theType != lType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoLow.theType, lType),
			ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	l.node = node
}

func (l *treeNodeListener) EnterAssignQuadruple(ctx *gen.AssignQuadrupleContext) {
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

	remHigh := ctx.IDENT(2).GetSymbol().GetText()
	infoRemHigh, err := getVarInfoForAssignment(remHigh, l.ctx)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(2).GetSymbol(), ctx.GetRuleContext())
		return
	}

	remLow := ctx.IDENT(3).GetSymbol().GetText()
	infoRemLow, err := getVarInfoForAssignment(remLow, l.ctx)
	if err != nil {
		reportError(err.Error(), ctx.GetParser(), ctx.IDENT(3).GetSymbol(), ctx.GetRuleContext())
		return
	}

	node := newAssignQuadrupleNode(l.ctx, l.parent, identLow, identHigh, remLow, remHigh)
	listener := newExprListener(l.ctx, node)
	ctx.TupleExpr().EnterRule(listener)
	rhs := listener.getExpr()
	node.value = rhs
	hType, lType, rhType, rlType, err := rhs.(*funCallNode).getTypeQuadruple()
	if err != nil {
		reportError(
			fmt.Sprintf("failed type resolution type: %s", err.Error()),
			ctx.GetParser(), ctx.TupleExpr().GetParser().GetCurrentToken(), ctx.GetRuleContext(),
		)
		return
	}
	if infoHigh.theType != hType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoHigh.theType, hType),
			ctx.GetParser(), ctx.IDENT(0).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if infoLow.theType != lType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoLow.theType, lType),
			ctx.GetParser(), ctx.IDENT(1).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if infoRemHigh.theType != rhType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoRemHigh.theType, rhType),
			ctx.GetParser(), ctx.IDENT(2).GetSymbol(), ctx.GetRuleContext(),
		)
		return
	}
	if infoRemLow.theType != rlType {
		reportError(
			fmt.Sprintf("incompatible types: (var) %s vs %s (expr)", infoLow.theType, rlType),
			ctx.GetParser(), ctx.IDENT(3).GetSymbol(), ctx.GetRuleContext(),
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

func (l *exprListener) EnterForExprCond(ctx *gen.ForExprCondContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterTypeCastExpr(ctx *gen.TypeCastExprContext) {
	var node *typeCastNode
	if ctx.TOBYTE() != nil {
		node = newTypeCastExprNode(l.ctx, l.parent, bytesType)
	} else {
		node = newTypeCastExprNode(l.ctx, l.parent, intType)
	}

	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	node.expr = listener.getExpr()

	l.expr = node
}

func (l *exprListener) EnterFunctionCallExpr(ctx *gen.FunctionCallExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.FunctionCall().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterBuiltinFunCall(ctx *gen.BuiltinFunCallContext) {
	name := ctx.BUILTINFUNC().GetText()
	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())
	// convert builtin function name or args if needed
	if remapper, ok := builtinFunRemap[name]; ok {
		errPos, err := remapper(exprNode)
		if err != nil {
			errToken := ctx.Expr(errPos).GetStart()
			parser := ctx.GetParser()
			rule := ctx.GetRuleContext()
			reportError(err.Error(), parser, errToken, rule)
			return
		}
	}

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.BUILTINFUNC().GetSymbol()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.expr = exprNode
}

func validateAppsIndex(ctx *context, exprNode ExprNodeIf) (err error) {
	indexType, err := exprNode.getType()
	if err != nil {
		return
	}
	if indexType != intType {
		err = fmt.Errorf("apps index must be int type")
		return
	}

	var value string
	switch tt := exprNode.(type) {
	case *exprIdentNode:
		ident := tt.name
		var info varInfo
		info, err = ctx.lookup(ident)
		if err != nil || !info.constant() {
			err = fmt.Errorf("%s not a constant", ident)
			return
		}
		value = *info.value
	case *exprLiteralNode:
		value = tt.value
	default:
		err = fmt.Errorf("apps[%s] must be a literal number or a constant", value)
		return
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		err = fmt.Errorf("%s not a number", value)
		return
	}
	if val != 0 {
		err = fmt.Errorf("apps[%s] must be a zero literal number or a constant", value)
		return
	}

	return
}

func (l *treeNodeListener) EnterBuiltinVarStatement(ctx *gen.BuiltinVarStatementContext) {
	exprs := ctx.AllExpr()

	var tealOpName string
	if ctx.ACCOUNTS() != nil {
		tealOpName = "app_local"
	} else {
		tealOpName = "app_global"
		// currently only 'this app = 0' supported for apps, ensure this fact
		listener := newExprListener(l.ctx, l.parent)
		exprs[0].EnterRule(listener)
		exprNode := listener.getExpr()
		token := exprs[0].GetParser().GetCurrentToken()
		if err := validateAppsIndex(l.ctx, exprNode); err != nil {
			reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
			return
		}
		exprs = exprs[1:]
	}

	var token antlr.Token
	if ctx.APPPUT() != nil {
		token = ctx.APPPUT().GetSymbol()
	} else {
		token = ctx.APPDEL().GetSymbol()
	}
	tealOpName = fmt.Sprintf("%s_%s", tealOpName, token.GetText())

	listener := newExprListener(l.ctx, l.parent)
	exprNode := listener.funCallEnterImpl(tealOpName, exprs)

	errPos, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := exprs[errPos].GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}

	l.node = exprNode
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
	if !info.function() {
		reportError("not a function", parser, token, rule)
		return
	}

	argExprNodes := ctx.AllExpr()
	funCallExprNode := l.funCallEnterImpl(name, argExprNodes)
	// parse function body
	defNode := info.parser(l.ctx, funCallExprNode, &info)
	if defNode == nil {
		reportError("function parsing failed", parser, token, rule)
		return
	}
	l.ctx.update(name, info) // save reference to funNodeDef

	if !ensureBlockReturns(defNode) {
		reportError(
			fmt.Sprintf("%s function does not return", name),
			parser, token, rule,
		)
		return
	}

	// save parsed functions at the root node to generate them
	// save functions in the current context in order to refresh local vars bindings
	if !defNode.inline {
		var node TreeNodeIf = defNode
		for ; node != nil; node = node.parent() {
			if p, ok := node.(*programNode); ok {
				found := false
				for _, ch := range p.nonInlineFunc {
					if ch.name == defNode.name {
						found = true
						break
					}
				}
				if !found {
					p.nonInlineFunc = append(p.nonInlineFunc, defNode)
				}
				break
			}
		}
		if _, ok := l.ctx.functions[defNode.name]; !ok {
			l.ctx.functions[defNode.name] = funCallExprNode
		}
	}

	funCallExprNode.definition = defNode
	l.expr = funCallExprNode
}

func (l *exprListener) EnterEcDsaFunCall(ctx *gen.EcDsaFunCallContext) {
	name := ctx.ECDSAVERIFY().GetText()
	field := ctx.ECDSACURVE().GetText()
	exprNode := l.funCallEnterImpl(name, ctx.AllExpr(), field)

	errPos, err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.Expr(errPos).GetStart()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.expr = exprNode
}

func (l *exprListener) EnterExtractFunCall(ctx *gen.ExtractFunCallContext) {
	name := ctx.EXTRACT().GetText()
	if ctx.EXTRACTOPT() != nil {
		field := ctx.EXTRACTOPT().GetText()
		if len(ctx.AllExpr()) != 2 {
			parser := ctx.GetParser()
			token := ctx.EXTRACT().GetSymbol()
			rule := ctx.GetRuleContext()
			reportError(fmt.Sprintf("extract %s accepts only 2 args", field), parser, token, rule)
		}
		switch field {
		case "UINT16":
			name = "extract_uint16"
		case "UINT32":
			name = "extract_uint32"
		case "UINT64":
			name = "extract_uint64"
		}
	}

	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())
	if remapper, ok := builtinFunRemap[name]; ok {
		errPos, err := remapper(exprNode)
		if err != nil {
			errToken := ctx.Expr(errPos).GetStart()
			parser := ctx.GetParser()
			rule := ctx.GetRuleContext()
			reportError(err.Error(), parser, errToken, rule)
			return
		}
	}

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		parser := ctx.GetParser()
		token := ctx.EXTRACT().GetSymbol()
		rule := ctx.GetRuleContext()
		reportError(err.Error(), parser, token, rule)
		return
	}

	l.expr = exprNode
}

func (l *exprListener) funCallEnterImpl(name string, allExpr []gen.IExprContext, aux ...string) (node *funCallNode) {
	node = newFunCallNode(l.ctx, l.parent, name, aux...)
	for _, expr := range allExpr {
		listener := newExprListener(l.ctx, node)
		expr.EnterRule(listener)
		arg := listener.getExpr()
		node.append(arg)
	}
	return node
}

func (l *exprListener) EnterTupleExpr(ctx *gen.TupleExprContext) {
	if node := ctx.BuiltinVarTupleExpr(); node != nil {
		listener := newExprListener(l.ctx, l.parent)
		node.EnterRule(listener)
		exprNode := listener.getExpr()
		l.expr = exprNode
		return
	}

	var field string
	var name string
	if ctx.MULW() != nil {
		name = ctx.MULW().GetText()
	} else if ctx.ADDW() != nil {
		name = ctx.ADDW().GetText()
	} else if ctx.EXPW() != nil {
		name = ctx.EXPW().GetText()
	} else if ctx.DIVMODW() != nil {
		name = ctx.DIVMODW().GetText()
	} else if ctx.ECDSADECOMPRESS() != nil {
		name = ctx.ECDSADECOMPRESS().GetText()
		field = ctx.ECDSACURVE().GetText()
	} else if ctx.ECDSARECOVER() != nil {
		name = ctx.ECDSARECOVER().GetText()
		field = ctx.ECDSACURVE().GetText()
	} else {
		token := ctx.GetParser().GetCurrentToken()
		reportError("unexpected token", ctx.GetParser(), token, ctx.GetRuleContext())
	}

	exprNode := l.funCallEnterImpl(name, ctx.AllExpr(), field)

	errPos, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := ctx.Expr(errPos).GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}

	l.expr = exprNode
}

func (l *exprListener) EnterBuiltinVarTupleExpr(ctx *gen.BuiltinVarTupleExprContext) {
	var name string
	var fieldArgToken antlr.Token
	if node := ctx.ACCOUNTS(); node != nil {
		if ctx.ASSETHLDBALANCE() != nil {
			fieldArgToken = ctx.ASSETHLDBALANCE().GetSymbol()
			fieldArgToken.SetText("AssetBalance")
			name = "asset_holding_get"
		} else if ctx.ASSETHLDFROZEN() != nil {
			fieldArgToken = ctx.ASSETHLDFROZEN().GetSymbol()
			fieldArgToken.SetText("AssetFrozen")
			name = "asset_holding_get"
		} else if ctx.ACCTPARAMS() != nil {
			fieldArgToken = ctx.ACCTPARAMS().GetSymbol()
			origText := fieldArgToken.GetText()
			newText := strings.ToUpper(string(origText[0])) + origText[1:]
			fieldArgToken.SetText(newText)
			name = "acct_params_get"
		} else {
			name = "app_local_get_ex"
		}
	} else if node := ctx.APPS(); node != nil {
		if ctx.APPGETEX() != nil {
			name = "app_global_get_ex"
		} else {
			fieldArgToken = ctx.APPPARAMSFIELDS().GetSymbol()
			name = "app_params_get"
		}
	} else if node := ctx.ASSETS(); node != nil {
		fieldArgToken = ctx.ASSETPARAMSFIELDS().GetSymbol()
		name = "asset_params_get"
	}

	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())

	errPos, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := ctx.Expr(errPos).GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}

	if fieldArgToken != nil {
		err = exprNode.resolveFieldArg(fieldArgToken.GetText())
		if err != nil {
			reportError(err.Error(), ctx.GetParser(), fieldArgToken, ctx.GetRuleContext())
		}
	}

	l.expr = exprNode
}

func (l *exprListener) EnterAccountsBalanceExpr(ctx *gen.AccountsBalanceExprContext) {
	name := "balance"
	if ctx.MINIMUMBALANCE() != nil {
		name = "min_balance"
	}
	exprNode := l.funCallEnterImpl(name, []gen.IExprContext{ctx.Expr()})

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := ctx.Expr().GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}
	l.expr = exprNode
}

func (l *exprListener) EnterAccountsSingleMethodsExpr(ctx *gen.AccountsSingleMethodsExprContext) {
	var name string
	if ctx.OPTEDIN() != nil {
		name = "app_opted_in"
	} else {
		name = "app_local_get"
	}
	exprNode := l.funCallEnterImpl(name, ctx.AllExpr())

	errPos, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := ctx.Expr(errPos).GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}
	l.expr = exprNode
}

func (l *exprListener) EnterAppsSingleMethodsExpr(ctx *gen.AppsSingleMethodsExprContext) {
	name := "app_global_get"
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr(0).EnterRule(listener)
	if err := validateAppsIndex(l.ctx, listener.getExpr()); err != nil {
		token := ctx.Expr(0).GetParser().GetCurrentToken()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}

	exprNode := l.funCallEnterImpl(name, []gen.IExprContext{ctx.Expr(1)})

	_, err := exprNode.checkBuiltinArgs()
	if err != nil {
		token := ctx.Expr(1).GetStart()
		reportError(err.Error(), ctx.GetParser(), token, ctx.GetRuleContext())
		return
	}
	l.expr = exprNode
}

func (l *exprListener) EnterAccountsExpr(ctx *gen.AccountsExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Accounts().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterAppsExpr(ctx *gen.AppsExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Apps().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterBuiltinObject(ctx *gen.BuiltinObjectContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.BuiltinVarExpr().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterGlobalFieldExpr(ctx *gen.GlobalFieldExprContext) {
	field := ctx.GLOBALFIELD().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "global", field)
	l.expr = node
}

func (l *exprListener) EnterTxnFieldExpr(ctx *gen.TxnFieldExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Txn().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterTxnSingleFieldExpr(ctx *gen.TxnSingleFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "txn", field)
	l.expr = node
}

func (l *exprListener) EnterTxnArrayFieldExpr(ctx *gen.TxnArrayFieldExprContext) {
	field := ctx.TXNARRAYFIELD().GetText()

	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	var errToken antlr.Token
	var node ExprNodeIf

	switch expr := exprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "txna", field, expr.value)

		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "txna", field, expr.value)
		}
	default:
		node = newRuntimeFieldNode(l.ctx, l.parent, "txnas", field)
		node.append(exprNode)
	}

	if errToken != nil {
		reportError(fmt.Sprintf("%s not a number", exprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	l.expr = node
}

func (l *exprListener) EnterInnerTxnFieldExpr(ctx *gen.InnerTxnFieldExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Itxn().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterInnerTxnSingleFieldExpr(ctx *gen.InnerTxnSingleFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	node := newRuntimeFieldNode(l.ctx, l.parent, "itxn", field)
	l.expr = node
}

func (l *exprListener) EnterInnerTxnArrayFieldExpr(ctx *gen.InnerTxnArrayFieldExprContext) {
	field := ctx.TXNARRAYFIELD().GetText()

	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	var errToken antlr.Token
	var node ExprNodeIf

	switch expr := exprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "itxna", field, expr.value)

		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "itxna", field, expr.value)
		}
	default:
		node = newRuntimeFieldNode(l.ctx, l.parent, "itxnas", field)
		node.append(exprNode)
	}

	if errToken != nil {
		reportError(fmt.Sprintf("%s not a number", exprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	l.expr = node
}

func (l *exprListener) EnterGroupTxnFieldExpr(ctx *gen.GroupTxnFieldExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Gtxn().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterGroupTxnSingleFieldExpr(ctx *gen.GroupTxnSingleFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	var op string
	var groupIndex string
	var errToken antlr.Token
	var node ExprNodeIf

	switch expr := exprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			groupIndex = expr.value
			op = "gtxn"
			node = newRuntimeFieldNode(l.ctx, l.parent, op, field, groupIndex)

		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			groupIndex = expr.value
			op = "gtxn"
			node = newRuntimeFieldNode(l.ctx, l.parent, op, field, groupIndex)
		}
	default:
		op = "gtxns"
		node = newRuntimeFieldNode(l.ctx, l.parent, op, field)
		node.append(exprNode)
	}

	if errToken != nil {
		reportError(fmt.Sprintf("%s not a number", exprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	l.expr = node
}

func (l *exprListener) EnterGroupTxnArrayFieldExpr(ctx *gen.GroupTxnArrayFieldExprContext) {
	field := ctx.TXNARRAYFIELD().GetText()

	groupIndexExpr := ctx.AllExpr()[0]
	arrayIndexExpr := ctx.AllExpr()[1]

	listener := newExprListener(l.ctx, l.parent)
	groupIndexExpr.EnterRule(listener)
	groupIndexExprNode := listener.getExpr()

	listener = newExprListener(l.ctx, l.parent)
	arrayIndexExpr.EnterRule(listener)
	arrayIndexExprNode := listener.getExpr()

	var errToken antlr.Token
	var node ExprNodeIf

	var groupIndex string
	switch expr := groupIndexExprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = groupIndexExpr.GetStart()
		} else {
			groupIndex = expr.value
		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = groupIndexExpr.GetStart()
		} else {
			groupIndex = expr.value
		}
	default:
	}
	if errToken != nil {
		reportError(fmt.Sprintf("group index %s not a number", groupIndexExprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	var arrayIndex string
	switch expr := arrayIndexExprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = arrayIndexExpr.GetStart()
		} else {
			arrayIndex = expr.value
		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = arrayIndexExpr.GetStart()
		} else {
			arrayIndex = expr.value
		}
	default:
	}
	if errToken != nil {
		reportError(fmt.Sprintf("array index %s not a number", arrayIndexExprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	// now there are 4 combinations of groupIndex and arrayIndex (has/not has)
	// and 4 opcodes to generate
	if groupIndex != "" {
		if arrayIndex != "" {
			node = newRuntimeFieldNode(l.ctx, l.parent, "gtxna", field, groupIndex, arrayIndex)
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "gtxnas", field, groupIndex)
			node.append(arrayIndexExprNode)
		}
	} else {
		if arrayIndex != "" {
			node = newRuntimeFieldNode(l.ctx, l.parent, "gtxnsa", field, "", arrayIndex)
			node.append(groupIndexExprNode)
		} else {
			node = newRuntimeFieldNode(l.ctx, l.parent, "gtxnsas", field)
			node.append(groupIndexExprNode)
			node.append(arrayIndexExprNode)
		}
	}

	l.expr = node
}

func (l *exprListener) EnterGroupInnerTxnFieldExpr(ctx *gen.GroupInnerTxnFieldExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Gitxn().EnterRule(listener)
	l.expr = listener.getExpr()
}

func (l *exprListener) EnterGroupInnerTxnSingleFieldExpr(ctx *gen.GroupInnerTxnSingleFieldExprContext) {
	field := ctx.TXNFIELD().GetText()
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	var op string
	var groupIndex string
	var errToken antlr.Token
	var node ExprNodeIf

	switch expr := exprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			groupIndex = expr.value
			op = "gitxn"
			node = newRuntimeFieldNode(l.ctx, l.parent, op, field, groupIndex)

		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			groupIndex = expr.value
			op = "gitxn"
			node = newRuntimeFieldNode(l.ctx, l.parent, op, field, groupIndex)
		}
	default:
		// group index must be provided
		errToken = ctx.Expr().GetStart()
		reportError(fmt.Sprintf("group index not provided as literal or constant"), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	if errToken != nil {
		reportError(fmt.Sprintf("%s not a number", exprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	l.expr = node
}

func (l *exprListener) EnterGroupInnerTxnArrayFieldExpr(ctx *gen.GroupInnerTxnArrayFieldExprContext) {
	field := ctx.TXNARRAYFIELD().GetText()

	groupIndexExpr := ctx.AllExpr()[0]
	arrayIndexExpr := ctx.AllExpr()[1]

	listener := newExprListener(l.ctx, l.parent)
	groupIndexExpr.EnterRule(listener)
	groupIndexExprNode := listener.getExpr()

	listener = newExprListener(l.ctx, l.parent)
	arrayIndexExpr.EnterRule(listener)
	arrayIndexExprNode := listener.getExpr()

	var errToken antlr.Token
	var node ExprNodeIf

	var groupIndex string
	switch expr := groupIndexExprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = groupIndexExpr.GetStart()
		} else {
			groupIndex = expr.value
		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = groupIndexExpr.GetStart()
		} else {
			groupIndex = expr.value
		}
	default:
		// group index must be provided
		errToken = groupIndexExpr.GetStart()
		reportError(fmt.Sprintf("group index not provided as literal or constant"), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}
	if errToken != nil {
		reportError(fmt.Sprintf("group index %s not a number", groupIndexExprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	var arrayIndex string
	switch expr := arrayIndexExprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = arrayIndexExpr.GetStart()
		} else {
			arrayIndex = expr.value
		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = arrayIndexExpr.GetStart()
		} else {
			arrayIndex = expr.value
		}
	default:
	}
	if errToken != nil {
		reportError(fmt.Sprintf("array index %s not a number", arrayIndexExprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

	// unlike gtxn{..} opcodes, the array index must be provided as an immediate arg
	// so there are only two combinations
	if arrayIndex != "" {
		node = newRuntimeFieldNode(l.ctx, l.parent, "gitxna", field, groupIndex, arrayIndex)
	} else {
		node = newRuntimeFieldNode(l.ctx, l.parent, "gitxnas", field, groupIndex)
		node.append(arrayIndexExprNode)
	}

	l.expr = node
}

func (l *exprListener) EnterArgsExpr(ctx *gen.ArgsExprContext) {
	listener := newExprListener(l.ctx, l.parent)
	ctx.Expr().EnterRule(listener)
	exprNode := listener.getExpr()

	var errToken antlr.Token
	var node ExprNodeIf

	switch expr := exprNode.(type) {
	case *constNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeArgNode(l.ctx, l.parent, "arg", expr.value)

		}
	case *exprLiteralNode:
		if expr.exprType != intType {
			errToken = ctx.Expr().GetStart()
		} else {
			node = newRuntimeArgNode(l.ctx, l.parent, "arg", expr.value)
		}
	default:
		node = newRuntimeArgNode(l.ctx, l.parent, "args", "")
		node.append(exprNode)
	}

	if errToken != nil {
		reportError(fmt.Sprintf("%s not a number", exprNode.String()), ctx.GetParser(), errToken, ctx.GetRuleContext())
		return
	}

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

	ctx := newContext("root", nil)

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

	ctx := newContext("root", nil)
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

	ctx := newContext("root", nil)
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
