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

const endProgramLabel = "end_program"
const trueConstName = "TRUE_INTERNAL"
const falseConstName = "FALSE_INTERNAL"
const trueConstValue = "1"
const falseConstValue = "0"

// Codegen by default emits AST node as a comment
func (n *TreeNode) Codegen(ostream io.Writer) {
	fmt.Fprintf(ostream, "// %s\n", n.String())
}

// Codegen of program node generates literals and runs code generation for children nodes
func (n *programNode) Codegen(ostream io.Writer) {
	ctx := n.ctx

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
	fmt.Fprintf(ostream, "%s:\n", endProgramLabel)
}

func (n *funDefNode) Codegen(ostream io.Writer) {
	if n.name == "logic" {
		for _, ch := range n.children() {
			ch.Codegen(ostream)
		}
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

func (n *returnNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)
	fmt.Fprintf(ostream, "intc %d\nbnz %s\n", n.ctx.literals.literals[trueConstValue].offset, endProgramLabel)
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

func (n *runtimeFieldNode) Codegen(ostream io.Writer) {
	if n.op == "gtxn" {
		fmt.Fprintf(ostream, "%s %s %s\n", n.op, n.index, n.field)
	} else {
		fmt.Fprintf(ostream, "%s %s\n", n.op, n.field)
	}
}

func (n *runtimeArgNode) Codegen(ostream io.Writer) {
	fmt.Fprintf(ostream, "%s %s\n", n.op, n.number)
}

func (n *ifExprNode) Codegen(ostream io.Writer) {
	n.condExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "!\nbnz if_expr_false_%d\n", &n)
	n.condTrueExpr.Codegen(ostream)
	fmt.Fprintf(ostream, "intc %d\nbnz if_expr_end_%d\n", n.ctx.literals.literals[trueConstValue].offset, &n)
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
		fmt.Fprintf(ostream, "!\nbnz if_stmt_false_%d\n", &n)
	} else {
		fmt.Fprintf(ostream, "!\nbnz if_stmt_end_%d\n", &n)
	}

	ch[0].Codegen(ostream)

	if hasFalse {
		fmt.Fprintf(ostream, "intc %d\nbnz if_stmt_end_%d\n", n.ctx.literals.literals[trueConstValue].offset, &n)
		fmt.Fprintf(ostream, "if_stmt_false_%d:\n", &n)
		ch[1].Codegen(ostream)
	}

	fmt.Fprintf(ostream, "if_stmt_end_%d:\n", &n)
}

func (n *blockNode) Codegen(ostream io.Writer) {
	for _, ch := range n.children() {
		ch.Codegen(ostream)
	}
}

// Codegen runs code generation for a node and returns the program as a string
func Codegen(prog TreeNodeIf) string {
	buf := new(gobytes.Buffer)
	prog.Codegen(buf)
	return buf.String()
}
