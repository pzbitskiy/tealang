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
