package compiler

import (
	"fmt"
	"testing"

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
const b = 1;

function test(x, y) {
	return x + y
}

function test(x, y) {
	return x - y
}

function logic(txn, gtxn, args) {
	let x = 1 + 1;
	if x == 2 {
		x = 0
		return 0
	}
	let s = global.GroupSize
	let t = txn.Note
	let g = gtxn[0].Sender
	let r = args[0]
	r = t
	t = s

	let z = sha256("test")

	let f = test(20+2, 30)
	if f + 2 < 10 {
		error
	}
	return 1
}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(5, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: uint64 vs byte[]`)
	a.Contains(parserErrors[1].msg, `if blocks types mismatch uint64 vs byte[]`)
	a.Contains(parserErrors[2].msg, `const 'b' already declared`)
	a.Contains(parserErrors[3].msg, `function 'test' already defined`)
	a.Contains(parserErrors[4].msg, `incompatible types: (var) byte[] vs uint64 (expr)`)
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
	a.Contains(errors[0].String(), "ident 'a' not defined")

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
	a.Contains(errors[0].String(), "ident 'test' not defined")
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
	a.Contains(errors[0].String(), "ident 'test' not defined")

	source = `
function logic(txn, gtxn, args) {test(1); return 1;}
`
	result, errors = Parse(source)
	a.Empty(result, errors)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident 'test' not defined")

	source = "let test = 1; function logic(txn, gtxn, args) {test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "not a function")

	source = `
function test(x) {return x;}
function logic(txn, gtxn, args) {test(); return 1;}
`
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "mismatching argument")
}

func TestFunctionType(t *testing.T) {
	a := require.New(t)

	source := `
function test(x, y) {return x + y;}
function logic(txn, gtxn, args) {let x = test(1, 2); return 1;}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
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

	source = `
function test(x, y) {return x + y;}
function logic(txn, gtxn, args) {let x = "abc"; x = test(1, 2); return 1;}
`

	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: (var) byte[] vs uint64 (expr)`)
}

func TestBuiltinFunction(t *testing.T) {
	a := require.New(t)
	source := `
function logic(txn, gtxn, args) {let x = sha256(1) ; return 1;}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: (exp) byte[] vs uint64 (actual) in expr 'sha256 ([1])'`)

	source = `
function logic(txn, gtxn, args) {let x = 1; x = sha256("abc") ; return 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, fmt.Sprintf(`incompatible types: (var) uint64 vs byte[] (expr)`))

}

func TestDoubleVariable(t *testing.T) {
	a := require.New(t)

	source := "function logic(txn, gtxn, args) {let x = 1; let x = 2;}"
	result, errors := Parse(source)
	a.Empty(result, errors)
	a.NotEmpty(errors)

	source = "let x = 1; function logic(txn, gtxn, args) {let x = 2;}"
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
}

func TestDoubleScopeVariable(t *testing.T) {
	a := require.New(t)

	source := `
function logic(txn, gtxn, args) {
	let x = 2;
	if 1 {
		let x = 3;
	} else {
		let x = 4;
	}
	let y = 5;
}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)

	pgNode := result.(*programNode)
	a.Equal(0, len(pgNode.ctx.vars))
	pgNode.ctx.Print()

	logicNode := pgNode.children()[0].(*funDefNode)
	a.Equal(2, len(logicNode.ctx.vars))
	info, _ := logicNode.ctx.vars["x"]
	a.Equal(uint(0), info.address)
	info, _ = logicNode.ctx.vars["y"]
	a.Equal(uint(1), info.address)

	ifStmtNode := logicNode.children()[1].(*ifStatementNode)

	ifStmtTrueNode := ifStmtNode.children()[0].(*blockNode)
	a.Equal(1, len(ifStmtTrueNode.ctx.vars))
	info, _ = ifStmtTrueNode.ctx.vars["x"]
	a.Equal(uint(1), info.address)

	ifStmtFalseNode := ifStmtNode.children()[1].(*blockNode)
	a.Equal(1, len(ifStmtFalseNode.ctx.vars))
	info, _ = ifStmtFalseNode.ctx.vars["x"]
	a.Equal(uint(1), info.address)
}
