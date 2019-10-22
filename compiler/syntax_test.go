package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidProgram(t *testing.T) {
	source := `
let a = 456; const b = 123; const c = "1234567890123";
let d = 1 + 2 ;
let e = if a > 0 {1} else {2}

function logic(txn, gtxn, account) {
	if e == 1 {
		let x = a + b;
		error
	}

	if a == 1 {
		return 0
	}

	if a == 1 {
		return 1
	} else {
		a = 2
	}

	x = 2;
	x = global.GroupSize
	x = gtxn[1].Sender
	sha256(x)
	ed25519verify("\x01\x02", c, x)
	return 1
}
`
	result := Compile(source)
	require.NotEmpty(t, result)
}

func TestInvalidProgram(t *testing.T) {
	source := "a = 33"
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")

	source = "let a = 33bbb"
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")

	source = `
let e = if a > 0 {1} else {2}

if e == 1 {
	let x = a + b;
	error
}
`
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")

	source = `
let e = if a > 0 {1} else {2}

function test() {
	if e == 1 {
		let x = a + b;
		error
	}
}
`
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")

	source = `
let e = if a > 0 {1} else {2}

function logic() {
	if e == 1 {
		let x = a + b;
		error
	}
}
`
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")
}

func TestParserValidProgram(t *testing.T) {
	source := `
let e = if a > 0 {1} else {2}

function logic(txn, gtxn, account) {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors := Parse(source)
	require.NotEmpty(t, result)
	require.Empty(t, errors)
}

func TestParserErrorReporting(t *testing.T) {
	source := `
let e = if a > 0 {1} else {}

function logic(txn, gtxn, account) {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors := Parse(source)
	require.Empty(t, result)
	require.NotEmpty(t, errors)
	require.Equal(t, 1, len(errors))
	require.Contains(t, errors[0].excerpt, "if a > 0 {1} else { ==> } <==")
	require.Contains(t, errors[0].String(), "syntax error at token \"}\" at line 2, col 27:  if a > 0 {1} else { ==> } <==")

	source = "a = 33"
	result, errors = Parse(source)
	require.Empty(t, result)
	require.NotEmpty(t, errors)
	require.Equal(t, 1, len(errors))
	require.Contains(t, errors[0].excerpt, "==> a <==  = 33")
	require.Contains(t, errors[0].String(), "syntax error at token \"a\" at line 1, col 0:")

	source = "let a = 33bbb"
	result, errors = Parse(source)
	require.Empty(t, result)
	require.NotEmpty(t, errors)
	require.Equal(t, 1, len(errors))
	require.Contains(t, errors[0].excerpt, "let a = 33 ==> bbb <==")
	require.Contains(t, errors[0].String(), "syntax error at token \"bbb\" at line 1, col 10: let a = 33 ==> bbb <==")

}
