package compiler

import (
	// "fmt"
	"testing"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserAllFeatures(t *testing.T) {
	a := require.New(t)

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

function logic(txn, gtxn, args) {
	let x = 1 + 1;
	if x == 2 {
		x = 0
		return 0
	}
	let f = global.GroupSize
	let t = txn.Note
	let g = gtxn[0].Sender
	let r = args[0]
	r = t
	t = f

	let f = test(20+2, 30)
	return 1
}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(3, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: uint64 vs byte[]`)
	a.Contains(parserErrors[1].msg, `if blocks types mismatch uint64 vs byte[]`)
	a.Contains(parserErrors[2].msg, `Incompatible types: (var) byte[] vs uint64 (expr)`)
}

func TestOneLinerLogic(t *testing.T) {
	a := require.New(t)
	source := "function logic(txn, gtxn, args) {return 1;}"
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "let a=1; function logic(txn, gtxn, args) {return 1;}"
	result, errors = Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)
}

func TestMissedLogicFunc(t *testing.T) {
	a := require.New(t)
	source := "let a = 1;"
	a.NotPanics(func() { Parse(source) })
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "Missing logic function")
}

func TestInvalidLogicFunc(t *testing.T) {
	a := require.New(t)
	source := "function logic(txn, gtxn, account) {}"
	a.NotPanics(func() { Parse(source) })
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
}

func TestAssignment(t *testing.T) {
	a := require.New(t)

	source := "function logic(txn, gtxn, args) {a=2; return 1;}"
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident a not defined")

	source = "function logic(txn, gtxn, args) {const a=1; a=2; return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "assign to a constant")

	source = "const a=1; function logic(txn, gtxn, args) {a=2; return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "assign to a constant")
}

func TestLookup(t *testing.T) {
	a := require.New(t)

	source := "let a=1; function logic(txn, gtxn, args) {a=2; return 1;}"
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "function logic(txn, gtxn, args) {test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident test not defined")
}

func TestFunctionLookup(t *testing.T) {
	a := require.New(t)

	source := `
function test(x, y) {return x + y;}
function logic(txn, gtxn, args) {test(1, 2); return 1;}
`
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "function logic(txn, gtxn, args) {test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident test not defined")

	source = "let test = 1; function logic(txn, gtxn, args) {test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "Not a function")

	source = `
function test(x) {return x;}
function logic(txn, gtxn, args) {test(); return 1;}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "Mismatching argument")
}

func TestFunctionType(t *testing.T) {
	a := require.New(t)

	source := `
function test(x, y) {return x + y;}
function logic(txn, gtxn, args) {let x = test(1, 2); return 1;}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result)
	a.Empty(parserErrors)

	source = `
function test(x, y) {
	if (x) {return x + y;}
	else {return "a";}
}
function logic(txn, gtxn, args) {let x = test(1, 2); return 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `block types mismatch: uint64 vs byte[]`)
}

func TestFunctionType2(t *testing.T) {
	a := require.New(t)
	source := `
function test(x, y) {return x + y;}
function logic(txn, gtxn, args) {let x = "abc"; x = test(1, 2); return 1;}
`

	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `Incompatible types: (var) byte[] vs uint64 (expr)`)
}

func T1estParser(t *testing.T) {
	source := `
let a = 456;
const b = "123";
let c = 1 + 2;
let d = 1 + "a"

function logic(txn, gtxn, args) {
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
