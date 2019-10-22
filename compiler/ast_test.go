package compiler

import (
	// "fmt"
	"testing"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	source := `
let a = 456;
const b = "123";
let c = 1 + 2 * (2 + 3);
let d = 1 + "a"

function logic(txn, gtxn, account) {
	return 1
}
`
	result, _ := Parse(source)
	require.NotEmpty(t, result)

	result.Print()

	errors := result.TypeCheck()
	require.NotEmpty(t, errors)
	require.Contains(t, errors, TypeError{`mismatching types at '1 + "a"' expr`})
}

func T1estParser(t *testing.T) {
	source := `
let a = 456;
const b = "123";
let c = 1 + 2;
let d = 1 + "a"

function logic(txn, gtxn, account) {
	if e == 1 {
		let x = a + b;
		error
	}

	if a == 1 {
		return 0
	}

	return 1
}
`
	result, _ := Parse(source)
	require.NotEmpty(t, result)

	result.Print()
}
