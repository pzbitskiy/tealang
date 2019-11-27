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

func (n *returnNode) Codegen(ostream io.Writer) {
	n.value.Codegen(ostream)
	fmt.Fprintf(ostream, "intc %d\nbnz %s\n", n.ctx.literals.literals[trueConstValue].offset, endProgramLabel)
}

// Codegen runs code generation for a node and returns the program as a string
func Codegen(prog TreeNodeIf) string {
	buf := new(gobytes.Buffer)
	prog.Codegen(buf)
	return buf.String()
}
