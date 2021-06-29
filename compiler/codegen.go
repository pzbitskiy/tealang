//--------------------------------------------------------------------------------------------------
//
// Code generation
//
//--------------------------------------------------------------------------------------------------

package compiler

import (
	gobytes "bytes"
	"encoding/hex"
	"fmt"
	"io"
)

const trueConstValue = "1"
const falseConstValue = "0"
const tealVersion = 4

var ids []interface{}

// Codegen by default emits AST node as a comment
func (n *TreeNode) Codegen(ostream io.Writer) {
	fmt.Fprintf(ostream, "// %s\n", n.String())
}

// Codegen of program node generates literals and runs code generation for children nodes
func (n *programNode) Codegen(ostream io.Writer) {
	ctx := n.ctx

	fmt.Fprintf(ostream, "#pragma version %d\n", tealVersion)

	// emit literals
	if len(ctx.literals.intc) > 0 {
		fmt.Fprintf(ostream, "intcblock ")
		sep := " "
		for idx, value := range ctx.literals.intc {
			if idx == len(ctx.literals.intc)-1 {
				sep = ""
			}
			fmt.Fprintf(ostream, "%s%s", value, sep)
		}
		fmt.Fprintf(ostream, "\n")
	}

	if len(ctx.literals.bytec) > 0 {
		fmt.Fprintf(ostream, "bytecblock ")
		sep := " "
		for idx, value := range ctx.literals.bytec {
			if idx == len(ctx.literals.bytec)-1 {
				sep = ""
			}
			fmt.Fprintf(ostream, "0x%s%s", hex.EncodeToString(value), sep)
		}
		fmt.Fprintf(ostream, "\n")
	}

	for _, ch := range n.children() {
		ch.Codegen(ostream)
	}
}

func (n *funDefNode) Codegen(ostream io.Writer) {
	if n.name == mainFuncName {
		for _, ch := range n.children() {
			ch.Codegen(ostream)
		}
		fmt.Fprintf(ostream, "end_%s:\n", n.name)
	}
}

func literalTypeToOpcode(theType exprType) string {
	op := "intc"
	if theType == bytesType {
		op = "bytec"
	}
	return op
}

func (n *exprLiteralNode) Codegen(ostream io.Writer) {
	op := literalTypeToOpcode(n.exprType)
	fmt.Fprintf(ostream, "%s %d\n", op, n.ctx.literals.literals[n.value].offset)
}

func (n *exprIdentNode) Codegen(ostream io.Writer) {
	info, _ := n.ctx.lookup(n.name)
	op := "load"
	if info.constant {
		op = literalTypeToOpcode(info.theType)
	}
	fmt.Fprintf(ostream, "%s %d\n", op, info.address)
}

func (n *assignNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)

	info, _ := n.ctx.lookup(n.name)
	fmt.Fprintf(ostream, "store %d\n", info.address)
}

func (n *assignTupleNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)

	info, _ := n.ctx.lookup(n.low)
	fmt.Fprintf(ostream, "store %d\n", info.address)
	info, _ = n.ctx.lookup(n.high)
	fmt.Fprintf(ostream, "store %d\n", info.address)
}

func (n *returnNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)
	if n.definition.name == mainFuncName {
		fmt.Fprintf(ostream, "return\n")
	} else {
		fmt.Fprintf(
			ostream,
			"intc %d\nbnz end_%s_%d\n",
			n.ctx.literals.literals[trueConstValue].offset, n.definition.name,
			&n.definition.name,
		)
	}
}

func (n *errorNode) Codegen(ostream io.Writer) {
	fmt.Fprintf(ostream, "err\n")
}

func (n *exprGroupNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)
}

func (n *exprBinOpNode) Codegen(ostream io.Writer) {
	n.lhs.Codegen(ostream)
	n.rhs.Codegen(ostream)

	fmt.Fprintf(ostream, "%s\n", n.op)
}

func (n *exprUnOpNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)

	fmt.Fprintf(ostream, "%s\n", n.op)
}

func (n *varDeclNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)

	info, _ := n.ctx.lookup(n.name)
	fmt.Fprintf(ostream, "store %d\n", info.address)
}

func (n *varDeclTupleNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)

	info, _ := n.ctx.lookup(n.low)
	fmt.Fprintf(ostream, "store %d\n", info.address)
	info, _ = n.ctx.lookup(n.high)
	fmt.Fprintf(ostream, "store %d\n", info.address)
}

func (n *runtimeFieldNode) Codegen(ostream io.Writer) {
	switch n.op {
	case "gtxn":
		fmt.Fprintf(ostream, "%s %s %s\n", n.op, n.index1, n.field)
	case "gtxns":
		for i := 0; i < len(n.childrenNodes); i++ {
			n.childrenNodes[i].Codegen(ostream)
		}
		fmt.Fprintf(ostream, "%s %s\n", n.op, n.field)
	case "gtxna":
		fmt.Fprintf(ostream, "%s %s %s %s\n", n.op, n.index1, n.field, n.index2)
	case "gtxnsa":
		for i := 0; i < len(n.childrenNodes); i++ {
			n.childrenNodes[i].Codegen(ostream)
		}
		fmt.Fprintf(ostream, "%s %s %s\n", n.op, n.field, n.index2)
	case "txna":
		fmt.Fprintf(ostream, "%s %s %s\n", n.op, n.field, n.index1)
	default:
		fmt.Fprintf(ostream, "%s %s\n", n.op, n.field)
	}
}

func (n *runtimeArgNode) Codegen(ostream io.Writer) {
	fmt.Fprintf(ostream, "%s %s\n", n.op, n.number)
}

func (n *ifExprNode) Codegen(ostream io.Writer) {
	n.condExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "bz if_expr_false_%d\n", &n)
	n.condTrueExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "b if_expr_end_%d\n", &n)
	fmt.Fprintf(ostream, "if_expr_false_%d:\n", &n)
	n.condFalseExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "if_expr_end_%d:\n", &n)
}

func (n *ifStatementNode) Codegen(ostream io.Writer) {
	n.condExpr.Codegen(ostream)
	ch := n.children()
	hasFalse := false
	if len(ch) == 2 {
		hasFalse = true
	}

	if hasFalse {
		fmt.Fprintf(ostream, "bz if_stmt_false_%d\n", &n)
	} else {
		fmt.Fprintf(ostream, "bz if_stmt_end_%d\n", &n)
	}

	ch[0].Codegen(ostream)

	if hasFalse {
		fmt.Fprintf(ostream, "b if_stmt_end_%d\n", &n)
		fmt.Fprintf(ostream, "if_stmt_false_%d:\n", &n)
		ch[1].Codegen(ostream)
	}

	fmt.Fprintf(ostream, "if_stmt_end_%d:\n", &n)
}

func (n *forStatementNode) Codegen(ostream io.Writer) {
	if ids == nil {
		ids = make([]interface{}, 0)
	}
	ids = append(ids, &n)

	fmt.Fprintf(ostream, "loop_start_%d:\n", &n)
	n.condExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "bz loop_end_%d\n", &n)
	ch := n.children()
	ch[0].Codegen(ostream)
	fmt.Fprintf(ostream, "b loop_start_%d\n", &n)
	fmt.Fprintf(ostream, "loop_end_%d:\n", &n)
}

func (n *breakNode) Codegen(ostream io.Writer) {

	id := ids[len(ids)-1]
	ids = ids[:len(ids)-1]

	fmt.Fprintf(ostream, "bz loop_end_%d\n", id)

}

func (n *blockNode) Codegen(ostream io.Writer) {
	for _, ch := range n.children() {
		ch.Codegen(ostream)
	}
}

func (n *funCallNode) Codegen(ostream io.Writer) {
	_, builtin := builtinFun[n.name]
	if builtin {
		// push args
		for _, ch := range n.children() {
			ch.Codegen(ostream)
		}
		field := ""
		if len(n.field) > 0 {
			field = fmt.Sprintf(" %s", n.field)
		} else if n.name == "substring" {
			field = fmt.Sprintf(" %s %s", n.index1, n.index2)
		}
		fmt.Fprintf(ostream, "%s%s\n", n.name, field)
	} else {
		definitionNode := n.definition

		// for each arg evaluate and store as appropriate named var
		for idx, ch := range n.children() {
			ch.Codegen(ostream)
			argName := definitionNode.args[idx]
			i, _ := definitionNode.ctx.lookup(argName)
			fmt.Fprintf(ostream, "store %d\n", i.address)
		}
		// and now generate statements
		for _, ch := range definitionNode.children() {
			ch.Codegen(ostream)
		}
		fmt.Fprintf(ostream, "end_%s_%d:\n", n.name, &n.definition.name)
	}
}

// Codegen runs code generation for a node and returns the program as a string
func Codegen(prog TreeNodeIf) string {
	buf := new(gobytes.Buffer)
	prog.Codegen(buf)
	return buf.String()
}
