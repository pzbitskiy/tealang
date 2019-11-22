package compiler

import (
	// "fmt"
	"testing"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserAllFeatures(t *testing.T) {
	source := `
let a = 456;
const b = "123";
let c = 1 + 2 * (2 + 3);
let d = 1 + "a"
let e = if c == 1 {1} else {2}
let e = if c == 1 {1} else {"1"}

function test(x, y) {
	return x + y
}

function logic(txn, gtxn, account) {
	let x = 1 + 1;
	if x == 2 {
		x = 0
		return 0
	}
	return 1
}
`
	result, _ := Parse(source)
	require.NotEmpty(t, result)

	result.Print()

	errors := result.TypeCheck()
	require.NotEmpty(t, errors)
	require.Equal(t, 2, len(errors), errors)
	require.Contains(t, errors, TypeError{`types mismatch: uint64 + byte[] in expr '1 + "a"'`})
	require.Contains(t, errors, TypeError{`if cond: different types: uint64 and byte[]`})
}

func TestOneLinerLogic(t *testing.T) {
	a := require.New(t)
	source := "function logic(txn, gtxn, account) {return 1;}"
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "let a=1; function logic(txn, gtxn, account) {return 1;}"
	result, errors = Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)
}

func TestMissedLogicFunc(t *testing.T) {
	a := require.New(t)
	source := "let a = 1;"
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "Missing logic function")
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
