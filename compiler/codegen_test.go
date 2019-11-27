package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodegenVariables(t *testing.T) {
	a := require.New(t)

	source := `let a = 1; let b = "123"; function logic(txn, gtxn, args) {a = 5; return 6;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 5 6", lines[0]) // 0 and 1 are added internally
	a.Equal("bytecblock 0x313233", lines[1])

	lastLine := len(lines) - 1
	a.Equal("intc 2", lines[lastLine-6])  // a = 5 (a's address is 0, 5's offset is 2)
	a.Equal("store 0", lines[lastLine-5]) //
	a.Equal("intc 3", lines[lastLine-4])  // ret 6 (6's offset is 3)
	a.Equal("intc 1", lines[lastLine-3])
	a.Equal("bnz end_program", lines[lastLine-2])
	a.Equal("end_program:", lines[lastLine-1])
	a.Equal("", lines[lastLine])
}

func TestCodegenErr(t *testing.T) {
	a := require.New(t)

	source := `function logic(txn, gtxn, args) {error;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0]) // 0 and 1 are added internally
	a.Equal("err", lines[1])
}

func TestCodegenBinOp(t *testing.T) {
	a := require.New(t)

	source := `const c = 10; function logic(txn, gtxn, args) {let a = 1 + c; let b = !a;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10", lines[0]) // 0 and 1 are added internally
	a.Equal("// const", lines[1])
	a.Equal("intc 1", lines[2])
	a.Equal("intc 2", lines[3])
	a.Equal("+", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("load 0", lines[6])
	a.Equal("!", lines[7])
	a.Equal("store 1", lines[8])
}

func TestCodegenIfExpr(t *testing.T) {
	a := require.New(t)

	source := `let x = if 1 { 2 } else { 3 }; function logic(txn, gtxn, args) {}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_expr_false_", lines[3][:len("bnz if_expr_false_")])
	a.Equal("intc 2", lines[4])
	a.Equal("intc 1", lines[5])
	a.Equal("bnz if_expr_end_", lines[6][:len("bnz if_expr_end_")])
	a.Equal("if_expr_false_", lines[7][:len("if_expr_false_")])
	a.Equal("intc 3", lines[8])
	a.Equal("if_expr_end_", lines[9][:len("if_expr_end_")])
	a.Equal("store 0", lines[10])
}

func TestCodegenIfStmt(t *testing.T) {
	a := require.New(t)

	source := `function logic(txn, gtxn, args) { if 1 {let x=10;}}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_stmt_end_", lines[3][:len("bnz if_stmt_end_")])
	a.Equal("intc 2", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("if_stmt_end_", lines[6][:len("if_stmt_end_")])

	source = `function logic(txn, gtxn, args) { if 1 {let x=10;} else {let y=11;}}`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog = Codegen(result)
	lines = strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10 11", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_stmt_false_", lines[3][:len("bnz if_stmt_false_")])
	a.Equal("intc 2", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("intc 1", lines[6])
	a.Equal("bnz if_stmt_end_", lines[7][:len("bnz if_stmt_end_")])
	a.Equal("if_stmt_false_", lines[8][:len("if_stmt_false_")])
	a.Equal("intc 3", lines[9])
	a.Equal("store 1", lines[10])
	a.Equal("if_stmt_end_", lines[11][:len("if_stmt_end_")])
}

func TestCodegenGlobals(t *testing.T) {
	a := require.New(t)

	source := `function logic(txn, gtxn, args) {let glob = global.MinTxnFee; let g = gtxn[1].Sender; let a = args[0];}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("global MinTxnFee", lines[1])
	a.Equal("store 0", lines[2])
	a.Equal("gtxn 1 Sender", lines[3])
	a.Equal("store 1", lines[4])
	a.Equal("arg 0", lines[5])
	a.Equal("store 2", lines[6])
}

func TestCodegenGeneric(t *testing.T) {
	a := require.New(t)

	source := `let a = 1; let b = "123"; function logic(txn, gtxn, args) {}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0]) // 0 and 1 are added internally
	a.Equal("bytecblock 0x313233", lines[1])

	// lastLine := len(lines) - 1
	fmt.Printf(prog)
}
