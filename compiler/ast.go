package compiler

import (
	"fmt"
	"io"
	"strings"
)

type literalDesc struct {
	offset  uint
	theType exprType
}

type literalInfo struct {
	literals map[string]literalDesc

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
	function bool

	// for variables specifies allocated memory space
	// for constants sets index in intc/bytec arrays
	address uint

	// constants have value
	value *string

	// function has reference lazy parser
	parser func(listener *treeNodeListener, callNode *funCallNode)
}

func newLiteralInfo() (literals *literalInfo) {
	literals = new(literalInfo)
	literals.literals = make(map[string]literalDesc)
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

		// global context, add internal literals
		ctx.addLiteral(falseConstValue, intType)
		ctx.addLiteral(trueConstValue, intType)
	}
	return
}

func (ctx *context) lookup(name string) (varable varInfo, err error) {
	current := ctx
	for current != nil {
		variable, ok := current.vars[name]
		if ok {
			return variable, nil
		}
		current = current.parent
	}
	return varInfo{}, fmt.Errorf("ident '%s' not defined", name)
}

func (ctx *context) update(name string, info varInfo) (err error) {
	current := ctx
	for current != nil {
		_, ok := current.vars[name]
		if ok {
			current.vars[name] = info
			return nil
		}
		current = current.parent
	}
	return fmt.Errorf("Failed to update ident %s", name)
}

func (ctx *context) newVar(name string, theType exprType) error {
	if _, ok := ctx.vars[name]; ok {
		return fmt.Errorf("variable '%s' already declared", name)
	}
	ctx.vars[name] = varInfo{name, theType, false, false, ctx.addressNext, nil, nil}
	ctx.addressNext++
	return nil
}

func (ctx *context) newConst(name string, theType exprType, value *string) error {
	if _, ok := ctx.vars[name]; ok {
		return fmt.Errorf("const '%s' already declared", name)
	}
	offset, err := ctx.addLiteral(*value, theType)
	if err != nil {
		return err
	}
	ctx.vars[name] = varInfo{name, theType, true, false, offset, value, nil}
	return nil
}

func (ctx *context) newFunc(name string, theType exprType, parser func(listener *treeNodeListener, callNode *funCallNode)) error {
	if _, ok := ctx.vars[name]; ok {
		return fmt.Errorf("function '%s' already defined", name)
	}

	ctx.vars[name] = varInfo{name, theType, false, true, 0, nil, parser}
	return nil
}

func (ctx *context) addLiteral(value string, theType exprType) (offset uint, err error) {
	info, exists := ctx.literals.literals[value]
	if !exists {
		if theType == intType {
			offset = uint(len(ctx.literals.intc))
			ctx.literals.intc = append(ctx.literals.intc, value)
			ctx.literals.literals[value] = literalDesc{offset, intType}
		} else if theType == bytesType {
			offset = uint(len(ctx.literals.bytec))
			parsed, err := parseStringLiteral(value)
			if err != nil {
				return 0, err
			}
			ctx.literals.bytec = append(ctx.literals.bytec, parsed)
			ctx.literals.literals[value] = literalDesc{offset, bytesType}
		} else {
			return 0, fmt.Errorf("unknown literal type %s (%s)", theType, value)
		}
	} else {
		offset = info.offset
	}

	return offset, err
}

func (ctx *context) Print() {
	for name, value := range ctx.vars {
		fmt.Printf("%v %v\n", name, value)
	}
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

var builtinFun = map[string]bool{
	"sha256":            true,
	"keccak256":         true,
	"sha512_256":        true,
	"ed25519verify":     true,
	"len":               true,
	"itob":              true,
	"btoi":              true,
	"concat":            true,
	"substring":         true,
	"substring3":        true,
	"mulw":              true,
	"addw":              true,
	"expw":              true,
	"exp":               true,
	"balance":           true,
	"min_balance":       true,
	"app_opted_in":      true,
	"app_local_get":     true,
	"app_local_get_ex":  true,
	"app_global_get":    true,
	"app_global_get_ex": true,
	"app_local_put":     true, // accounts[x].put
	"app_global_put":    true, // apps[0].put
	"app_local_del":     true, // accounts[x].del
	"app_global_del":    true, // apps[0].del
	"asset_holding_get": true,
	"asset_params_get":  true,
	"assert":            true,
	"getbit":            true,
	"getbyte":           true,
	"setbit":            true,
	"setbyte":           true,
	"shl":               true,
	"shr":               true,
	"sqrt":              true,
	"bitlen":            true,
}

var builtinFunDependantTypes = map[string]int{
	"setbit": 0, // op type matches to first arg type
}

// TreeNodeIf represents a node in AST
type TreeNodeIf interface {
	append(ch TreeNodeIf)
	children() []TreeNodeIf
	parent() TreeNodeIf
	String() string
	Print()
	Codegen(ostream io.Writer)
}

// ExprNodeIf extends TreeNode and can be evaluated and typed
type ExprNodeIf interface {
	TreeNodeIf
	getType() (exprType, error)
}

// TreeNode contains base info about an AST node
type TreeNode struct {
	ctx *context

	nodeName      string
	parentNode    TreeNodeIf
	childrenNodes []TreeNodeIf
}

type programNode struct {
	*TreeNode
}

type funDefNode struct {
	*TreeNode
	name string
	args []string
}

type blockNode struct {
	*TreeNode
}

type returnNode struct {
	*TreeNode
	value      ExprNodeIf
	definition *funDefNode
}

type errorNode struct {
	*TreeNode
}

type breakNode struct {
	*TreeNode
	value ExprNodeIf
}

type assignNode struct {
	*TreeNode
	name     string
	exprType exprType
	value    ExprNodeIf
}

type assignTupleNode struct {
	*TreeNode
	low      string
	high     string
	exprType exprType
	value    ExprNodeIf
}

type varDeclNode struct {
	*TreeNode
	name     string
	exprType exprType
	value    ExprNodeIf
}

type varDeclTupleNode struct {
	*TreeNode
	low      string
	high     string
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

type forStatementNode struct {
	*TreeNode
	condExpr     ExprNodeIf
	condTrueExpr ExprNodeIf
}

type ifStatementNode struct {
	*TreeNode
	condExpr ExprNodeIf
}

type funCallNode struct {
	*TreeNode
	name       string
	field      string
	index1     string
	index2     string
	funType    exprType
	definition *funDefNode
}

type runtimeFieldNode struct {
	*TreeNode
	op       string
	field    string
	index1   string
	index2   string
	exprType exprType
}

type runtimeArgNode struct {
	*TreeNode
	op       string
	number   string
	exprType exprType
}

//--------------------------------------------------------------------------------------------------
//
// AST nodes constructors
//
//--------------------------------------------------------------------------------------------------

func newNode(ctx *context, parent TreeNodeIf) (node *TreeNode) {
	node = new(TreeNode)
	node.ctx = ctx
	node.childrenNodes = make([]TreeNodeIf, 0)
	node.parentNode = parent
	return node
}

func newProgramNode(ctx *context, parent TreeNodeIf) (node *programNode) {
	node = new(programNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "program"
	return
}

func newBlockNode(ctx *context, parent TreeNodeIf) (node *blockNode) {
	node = new(blockNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "block"
	return
}

func newReturnNode(ctx *context, parent TreeNodeIf) (node *returnNode) {
	node = new(returnNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "ret"
	node.value = nil
	return
}

func newErorrNode(ctx *context, parent TreeNodeIf) (node *errorNode) {
	node = new(errorNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "error"
	return
}

func newBreakNode(ctx *context, parent TreeNodeIf) (node *breakNode) {
	node = new(breakNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "break"
	node.value = nil
	return
}

func newAssignNode(ctx *context, parent TreeNodeIf, ident string) (node *assignNode) {
	node = new(assignNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "assign"
	node.name = ident
	node.value = nil
	return
}

func newAssignTupleNode(ctx *context, parent TreeNodeIf, identLow string, identHigh string) (node *assignTupleNode) {
	node = new(assignTupleNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "assign tuple"
	node.low = identLow
	node.high = identHigh
	node.value = nil
	return
}

func newFunDefNode(ctx *context, parent TreeNodeIf) (node *funDefNode) {
	node = new(funDefNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "func"
	return
}

func newVarDeclNode(ctx *context, parent TreeNodeIf, ident string, value ExprNodeIf) (node *varDeclNode) {
	node = new(varDeclNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "var"
	node.name = ident
	node.value = value
	tp, _ := value.getType()
	node.exprType = tp
	return
}

func newVarDeclTupleNode(ctx *context, parent TreeNodeIf, identLow string, identHigh string, value ExprNodeIf) (node *varDeclTupleNode) {
	node = new(varDeclTupleNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "var, var"
	node.low = identLow
	node.high = identHigh
	node.value = value
	tp, _ := value.getType()
	node.exprType = tp
	return
}

func newConstNode(ctx *context, parent TreeNodeIf, ident string, value string, exprType exprType) (node *constNode) {
	node = new(constNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "const"
	node.name = ident
	node.value = value
	node.exprType = exprType
	return
}

func newExprIdentNode(ctx *context, parent TreeNodeIf, name string, exprType exprType) (node *exprIdentNode) {
	node = new(exprIdentNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "expr ident"
	node.name = name
	node.exprType = exprType
	return
}

func newExprLiteralNode(ctx *context, parent TreeNodeIf, valType exprType, value string) (node *exprLiteralNode) {
	node = new(exprLiteralNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "expr liter"
	node.value = value
	node.exprType = valType
	return
}

func newExprBinOpNode(ctx *context, parent TreeNodeIf, op string) (node *exprBinOpNode) {
	node = new(exprBinOpNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "expr OP expr"
	node.exprType = intType
	node.op = op
	return
}

func newExprGroupNode(ctx *context, parent TreeNodeIf, value ExprNodeIf) (node *exprGroupNode) {
	node = new(exprGroupNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "(expr)"
	node.value = value
	return
}

func newExprUnOpNode(ctx *context, parent TreeNodeIf, op string) (node *exprUnOpNode) {
	node = new(exprUnOpNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "OP expr"
	node.op = op
	return
}

func newIfExprNode(ctx *context, parent TreeNodeIf) (node *ifExprNode) {
	node = new(ifExprNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "if expr"
	return
}

func newIfStatementNode(ctx *context, parent TreeNodeIf) (node *ifStatementNode) {
	node = new(ifStatementNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "if stmt"
	return
}

func newForStatementNode(ctx *context, parent TreeNodeIf) (node *forStatementNode) {
	node = new(forStatementNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "for stmt"
	return
}

func newFunCallNode(ctx *context, parent TreeNodeIf, name string) (node *funCallNode) {
	node = new(funCallNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "fun call"
	node.name = name
	node.funType = unknownType
	return
}

func newRuntimeFieldNode(ctx *context, parent TreeNodeIf, op string, field string, aux ...string) (node *runtimeFieldNode) {
	node = new(runtimeFieldNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "runtime field"
	node.op = op
	node.field = field
	if len(aux) > 0 {
		node.index1 = aux[0]
	}
	if len(aux) > 1 {
		node.index2 = aux[1]
	}
	node.exprType = unknownType
	return
}

func newRuntimeArgNode(ctx *context, parent TreeNodeIf, op string, number string) (node *runtimeArgNode) {
	node = new(runtimeArgNode)
	node.TreeNode = newNode(ctx, parent)
	node.nodeName = "runtime arg"
	node.op = op
	node.number = number
	node.exprType = unknownType
	return
}

//--------------------------------------------------------------------------------------------------
//
// Type checks
//
//--------------------------------------------------------------------------------------------------

func (n *exprLiteralNode) getType() (exprType, error) {
	return n.exprType, nil
}

func (n *exprIdentNode) getType() (exprType, error) {
	if n.exprType == unknownType {
		info, err := n.ctx.lookup(n.name)
		if err != nil || info.theType == invalidType {
			return invalidType, fmt.Errorf("ident lookup for %s failed: %s", n.name, err.Error())
		}
		n.exprType = info.theType
	}
	return n.exprType, nil
}

func (n *exprBinOpNode) getType() (exprType, error) {
	tp, err := opTypeFromSpec(n.op, 0)
	if err != nil {
		return invalidType, fmt.Errorf("bin op '%s' not it the language: %s", n.op, err.Error())
	}

	lhs, err := n.lhs.getType()
	if err != nil {
		return invalidType, fmt.Errorf("left operand '%s' has invalid type: %s", n.lhs.String(), err.Error())
	}
	rhs, err := n.rhs.getType()
	if err != nil {
		return invalidType, fmt.Errorf("right operand '%s' has invalid type: %s", n.rhs.String(), err.Error())
	}

	opLHS, err := argOpTypeFromSpec(n.op, 0)
	if err != nil {
		return invalidType, err
	}
	if opLHS != unknownType && lhs != opLHS {
		return invalidType, fmt.Errorf("incompatible left operand type: '%s' vs '%s' in expr '%s'", opLHS, lhs, n)
	}

	opRHS, err := argOpTypeFromSpec(n.op, 1)
	if err != nil {
		return invalidType, err
	}
	if opRHS != unknownType && rhs != opRHS {
		return invalidType, fmt.Errorf("incompatible right operand type: '%s' vs '%s' in expr '%s'", opRHS, rhs, n)
	}
	if lhs != rhs {
		return invalidType, fmt.Errorf("incompatible types: '%s' vs '%s' in expr '%s'", lhs, rhs, n)
	}

	return tp, nil
}

func (n *exprUnOpNode) getType() (exprType, error) {
	tp, err := opTypeFromSpec(n.op, 0)
	if err != nil {
		return invalidType, fmt.Errorf("un op '%s' not it the language: %s", n.op, err.Error())
	}

	valType, err := n.value.getType()
	if err != nil {
		return invalidType, fmt.Errorf("operand '%s' has invalid type: %s", n.String(), err.Error())
	}

	operandType, err := argOpTypeFromSpec(n.op, 0)
	if err != nil {
		return invalidType, err
	}
	if operandType != unknownType && valType != operandType {
		return invalidType, fmt.Errorf("incompatible operand type: '%s' vs %s in expr '%s'", operandType, valType, n)
	}

	if tp != valType {
		return invalidType, fmt.Errorf("up op expects type '%s' but operand is '%s'", tp, valType)
	}
	return tp, nil
}

func (n *ifExprNode) getType() (exprType, error) {
	tp, err := n.condExpr.getType()
	if err != nil {
		return invalidType, fmt.Errorf("cond type evaluation failed: %s", err.Error())
	}

	condType := tp
	if condType != intType {
		return invalidType, fmt.Errorf("cond type is '%s', expected '%s'", condType, tp)
	}

	condTrueExprType, err := n.condTrueExpr.getType()
	if err != nil {
		return invalidType, fmt.Errorf("first block has invalid type: %s", err.Error())
	}
	condFalseExprType, err := n.condFalseExpr.getType()
	if err != nil {
		return invalidType, fmt.Errorf("second block has invalid type: %s", err.Error())
	}
	if condTrueExprType != condFalseExprType {
		return invalidType, fmt.Errorf("if blocks types mismatch '%s' vs '%s'", condTrueExprType, condFalseExprType)
	}

	return condTrueExprType, nil
}

func (n *exprGroupNode) getType() (exprType, error) {
	return n.value.getType()
}

// Scans node's children recursively and find return statements,
// applies type resolution and track conflicts.
// Return expr type or invalidType on error
func determineBlockReturnType(node TreeNodeIf, retTypeSeen []exprType) (exprType, error) {
	var statements []TreeNodeIf
	if node != nil {
		statements = node.children()
	}

	for _, stmt := range statements {
		switch tt := stmt.(type) {
		case *returnNode:
			tp, err := tt.value.getType()
			if err != nil {
				return invalidType, err
			}
			retTypeSeen = append(retTypeSeen, tp)
		case *errorNode:
			retTypeSeen = append(retTypeSeen, intType) // error is ok
		case *ifStatementNode, *blockNode:
			blockType, err := determineBlockReturnType(stmt, retTypeSeen)
			if err != nil {
				return invalidType, err
			}
			retTypeSeen = append(retTypeSeen, blockType)
		}
	}

	if len(retTypeSeen) == 0 {
		return unknownType, nil
	}
	commonType := retTypeSeen[0]
	for _, tp := range retTypeSeen {
		if commonType == unknownType && tp != unknownType {
			commonType = tp
			continue
		}

		if commonType != unknownType && tp != commonType {
			return invalidType, fmt.Errorf("block types mismatch: %s vs %s", commonType, tp)
		}
	}
	return commonType, nil
}

func ensureBlockReturns(node TreeNodeIf) bool {
	chLength := len(node.children())
	if chLength == 0 {
		return false
	}

	lastNode := node.children()[chLength-1]
	switch tt := lastNode.(type) {
	case *returnNode, *errorNode:
		return true
	case *ifStatementNode:
		if len(tt.children()) == 1 {
			// only if-block present
			return false
		}
		// otherwise ensure both if-else and else-block returns
		return ensureBlockReturns(lastNode.children()[0]) && ensureBlockReturns(lastNode.children()[1])
	default:
	}

	return false
}

func (n *funCallNode) getType() (exprType, error) {
	if n.funType != unknownType {
		return n.funType, nil
	}

	var err error
	builtin := false
	_, err = n.ctx.lookup(n.name)
	if err != nil {
		_, builtin = builtinFun[n.name]
		if !builtin {
			return invalidType, fmt.Errorf("function %s lookup failed: %s", n.name, err.Error())
		}
	}

	var tp exprType
	if builtin {
		tp, err = opTypeFromSpec(n.name, 0)
		if tp == unknownType {
			if idx, ok := builtinFunDependantTypes[n.name]; ok {
				tp, err = n.childrenNodes[idx].(ExprNodeIf).getType()
				if err != nil {
					return invalidType, fmt.Errorf("function %s type deduction failed: %s", n.name, err.Error())
				}
			}
		}
	} else {
		tp, err = determineBlockReturnType(n.definition, []exprType{})
	}
	n.funType = tp
	return tp, err
}

func (n *funCallNode) getTypeTuple() (exprType, exprType, error) {
	var err error
	builtin := false
	_, builtin = builtinFun[n.name]
	if !builtin {
		return invalidType, invalidType, fmt.Errorf("function %s lookup failed: %s", n.name, err.Error())
	}

	var tpl exprType = invalidType
	var tph exprType = invalidType
	tph, err = opTypeFromSpec(n.name, 0)
	if err != nil {
		return tph, tpl, err
	}
	tpl, err = opTypeFromSpec(n.name, 1)
	return tph, tpl, err
}

func (n *funCallNode) resolveArgs(definitionNode *funDefNode) error {
	args := n.children()

	if len(definitionNode.args) != len(args) {
		return fmt.Errorf("mismatching parsed argument(s)")
	}

	for i := range args {
		varName := definitionNode.args[i]
		info, err := definitionNode.ctx.lookup(varName)
		if err != nil {
			return err
		}
		info.theType, err = args[i].(ExprNodeIf).getType()
		if err != nil {
			return err
		}
		err = definitionNode.ctx.update(varName, info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *funCallNode) checkBuiltinArgs() (err error) {
	args := n.children()
	for i, arg := range args {
		tp, err := argOpTypeFromSpec(n.name, i)
		if err != nil {
			return err
		}
		argExpr := arg.(ExprNodeIf)
		actualType, err := argExpr.getType()
		if err != nil {
			return err
		}
		if tp != unknownType && actualType != unknownType && actualType != tp {
			return fmt.Errorf("incompatible types: (exp) %s vs %s (actual) in expr '%s'", tp, actualType, n)
		}
	}
	return
}

func (n *funCallNode) resolveFieldArg(field string) (err error) {
	tp, err := runtimeFieldTypeFromSpec(n.name, field)
	if err != nil {
		return
	}
	n.field = field
	n.funType = tp
	return
}

func (n *runtimeFieldNode) getType() (exprType, error) {
	if n.exprType != unknownType {
		return n.exprType, nil
	}

	tp, err := runtimeFieldTypeFromSpec(n.op, n.field)
	if err != nil {
		return invalidType, fmt.Errorf("lookup failed: %s", err.Error())
	}

	n.exprType = tp
	return tp, err
}

func (n *runtimeArgNode) getType() (exprType, error) {
	if n.exprType != unknownType {
		return n.exprType, nil
	}

	tp, err := opTypeFromSpec(n.op, 0)
	if err != nil {
		return invalidType, fmt.Errorf("lookup failed: %s", err.Error())
	}

	n.exprType = tp
	return tp, err
}

func (n *constNode) getType() (exprType, error) {
	return n.exprType, nil
}

//--------------------------------------------------------------------------------------------------
//
// Common node methods
//
//--------------------------------------------------------------------------------------------------

func (n *TreeNode) append(ch TreeNodeIf) {
	n.childrenNodes = append(n.childrenNodes, ch)
}

func (n *TreeNode) children() []TreeNodeIf {
	return n.childrenNodes
}

func (n *TreeNode) String() string {
	return n.nodeName
}

func (n *TreeNode) parent() TreeNodeIf {
	return n.parentNode
}

// Print AST and context
func (n *TreeNode) Print() {
	printImpl(n, 0)

	n.ctx.Print()
}

func printImpl(n TreeNodeIf, offset int) {
	fmt.Printf("%s%s\n", strings.Repeat(" ", offset), n.String())
	for _, ch := range n.children() {
		printImpl(ch, offset+4)
	}
}

func (n *varDeclNode) String() string {
	return fmt.Sprintf("var (%s) %s = %s", n.exprType, n.name, n.value)
}

func (n *varDeclTupleNode) String() string {
	return fmt.Sprintf("var (%s) %s, %s = %s", n.exprType, n.high, n.low, n.value)
}

func (n *constNode) String() string {
	return fmt.Sprintf("const (%s) %s = %s", n.exprType, n.name, n.value)
}

func (n *funDefNode) String() string {
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

func (n *forStatementNode) String() string {
	return fmt.Sprintf("for %s { %s}", n.condExpr, n.condTrueExpr)
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

func (n *runtimeFieldNode) String() string {
	if n.op == "gtxn" {
		return fmt.Sprintf("%s[%s].%s\n", n.op, n.index1, n.field)
	} else if n.op == "gtxna" {
		return fmt.Sprintf("%s[%s].%s[%s]\n", n.op, n.index1, n.field, n.index2)
	} else if n.op == "txna" {
		return fmt.Sprintf("%s.%s[%s]\n", n.op, n.field, n.index1)
	} else {
		return fmt.Sprintf("%s.%s\n", n.op, n.field)
	}
}
