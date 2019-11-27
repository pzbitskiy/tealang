package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicCodegen(t *testing.T) {
	a := require.New(t)

	source := `let a = 1; let b = "123"; function logic(txn, gtxn, args) { a = 2; return 0;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2", lines[0]) // 0 and 1 are added internally
	a.Equal("bytecblock 0x313233", lines[1])
	fmt.Printf(prog)
}
