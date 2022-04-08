package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidProgram(t *testing.T) {
	a := require.New(t)
	source := `
let a = 456; const b = 123; const c = "1234567890123";
let d = 1 + 2 ;
let e = if a > 0 {1} else {2}

function logic() {
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

	let x = 2;
	x = global.GroupSize
	let y = args[0]
	y = sha256(c)
	x = ed25519verify("\x01\x02", c, "test")
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
}

func TestParserValidProgram(t *testing.T) {
	a := require.New(t)
	source := `
let a = 1
let e = if a > 0 {1} else {2}

function logic() {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
let e = 1

function approval() {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
let e = 2;

function clearstate() {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
}

func TestParserErrorReporting(t *testing.T) {
	a := require.New(t)

	source := `
let e = if a > 0 {1} else {}

function logic() {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("let e = if a > 0 {1} else {}", errors[0].excerpt[0])
	a.Equal("                      -----^-----", errors[0].excerpt[1])
	msg := `syntax error at line 2, col 27 near token '}'
let e = if a > 0 {1} else {}
                      -----^-----`
	a.Equal(msg, errors[0].String())

	source = "a = 33"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("a = 33", errors[0].excerpt[0])
	a.Equal("^-----", errors[0].excerpt[1])
	msg = `syntax error at line 1, col 0 near token 'a'
a = 33
^-----`
	a.Equal(msg, errors[0].String())

	source = "let a = 33bbb"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("let a = 33bbb", errors[0].excerpt[0])
	a.Equal("     -----^-----", errors[0].excerpt[1])
	msg = `syntax error at line 1, col 10 near token 'bbb'
let a = 33bbb
     -----^-----`
	a.Equal(msg, errors[0].String())

	source = `
function logic() {
	if e == 1 {
		let x = 2;
		error
	}
	return 1
}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("    if e == 1 {", errors[0].excerpt[0])
	a.Equal("  -----^-----", errors[0].excerpt[1])
	msg = `error at line 3, col 4 near token 'e'
    if e == 1 {
  -----^-----
ident not found`
	a.Equal(msg, errors[0].String())

	source = `
function logic() {
	let x = "123"
	let e = 1
	if e == 1 {
		x = 2
		error
	}
	return 1
}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("        x = 2", errors[0].excerpt[0])
	a.Equal("   -----^-----", errors[0].excerpt[1])
	msg = `error at line 6, col 2 near token 'x'
        x = 2
   -----^-----
incompatible types: (var) byte[] vs uint64 (expr)`
	a.Equal(msg, errors[0].String())

	source = `
function logic() {
	let e = 2
	e = "123"
	return 1
}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal(`    e = "123"`, errors[0].excerpt[0])
	a.Equal("----^-----", errors[0].excerpt[1])
	msg = `error at line 4, col 1 near token 'e'
    e = "123"
----^-----
incompatible types: (var) uint64 vs byte[] (expr)`
	a.Equal(msg, errors[0].String())
}

func TestIfElseProgram(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let e = 2
	if e == 1 {return 1;}
	else {return 0;}
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1 { return 1; }
	else { return 0; }
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1 {
		return 1;
	} else {
		return 0;
	}
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1 {
		return 1;
	}
	else {
		return 0;
	}
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1
	{
		return 1;
	}
}
`
	result, errors = Parse(source)
	a.Empty(result, errors)
	a.NotEmpty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1 {
		return 1;
	}
	return 0
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let e = 2
	if e == 1 { return 1; }
	return 0
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

}

func TestStringLiteralPrefixes(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let a = b32"GEZDGCQ="
	return 0
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	source = `
function logic() {
	let a = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
	return 0
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
}

func TestBreakError1(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	break
	return 0
}
`
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("    break", errors[0].excerpt[0])
	a.Equal("----^-----", errors[0].excerpt[1])
	msg := `error at line 3, col 1 near token 'break'
    break
----^-----
break is not inside for block`
	a.Equal(msg, errors[0].String())

	source = `
function logic() {
	if 1 {
		break
	}
	return 0
}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("        break", errors[0].excerpt[0])
	a.Equal("   -----^-----", errors[0].excerpt[1])
	msg = `error at line 4, col 2 near token 'break'
        break
   -----^-----
break is not inside for block`
	a.Equal(msg, errors[0].String())
}

func TestBreakError2(t *testing.T) {
	a := require.New(t)

	source := `
function test() {
	break
	return 1
}
function logic() {
	let res = 0
	for 1 {
		res = res + test()
	}
	return 0
}
`
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Equal(1, len(errors))
	a.Equal("    break", errors[0].excerpt[0])
	a.Equal("----^-----", errors[0].excerpt[1])
	msg := `error at line 3, col 1 near token 'break'
    break
----^-----
break is not inside for block`
	a.Equal(msg, errors[0].String())
}
