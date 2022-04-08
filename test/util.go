package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/algorand/go-algorand/data/transactions/logic"
	"github.com/pzbitskiy/tealang/compiler"
	"github.com/pzbitskiy/tealang/dryrun"
	"github.com/stretchr/testify/require"
)

func compileTest(t *testing.T, source string) *logic.OpStream {
	t.Helper()
	a := require.New(t)
	result, errors := compiler.Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	teal := compiler.Codegen(result)
	op, err := logic.AssembleString(teal)
	a.NoError(err)
	return op
}

func performTest(t *testing.T, source string) {
	t.Helper()
	a := require.New(t)
	op := compileTest(t, source)

	sb := strings.Builder{}
	pass, err := dryrun.Run(op.Program, "", &sb)
	fmt.Printf("trace:\n%s\n", sb.String())

	a.NoError(err)
	a.True(pass)
}
