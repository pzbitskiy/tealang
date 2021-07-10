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

function logic() {
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

	x = shr(shl(9, 2), 2)
	x = bitlen(sqrt(x))
	let zero = bzero(5)
	let borres = bor(zero, "\x10")
	let p = gaid(0)
	let q = gaids(0)
	return 1
}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(5, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible right operand type: 'uint64' vs 'byte[]'`)
	a.Contains(parserErrors[1].msg, `if blocks types mismatch 'uint64' vs 'byte[]'`)
	a.Contains(parserErrors[2].msg, `const 'b' already declared`)
	a.Contains(parserErrors[3].msg, `function 'test' already defined`)
	a.Contains(parserErrors[4].msg, `incompatible types: (var) byte[] vs uint64 (expr)`)
}

func TestOneLinerLogic(t *testing.T) {
	a := require.New(t)
	source := "function logic() {return 1;}"
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "let a=1; function logic() {return 1;}"
	result, errors = Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)
}

// TODO: Add this heuristic back
// func TestMissedLogicFunc(t *testing.T) {
// 	a := require.New(t)
// 	source := "let a = 1;"
// 	a.NotPanics(func() { Parse(source) })
// 	result, errors := Parse(source)
// 	a.Empty(result)
// 	a.NotEmpty(errors)
// 	a.Contains(errors[0].String(), "Missing logic function")
// }

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

	source := "function logic() {a=2; return 1;}"
	result, errors := Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident 'a' not defined")

	source = "function logic() {const a=1; a=2; return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "assign to a constant")

	source = "const a=1; function logic() {a=2; return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "assign to a constant")
}

func TestLookup(t *testing.T) {
	a := require.New(t)

	source := "let a=1; function logic() {a=2; return 1;}"
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "function logic() {let a = test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident 'test' not defined")
}

func TestFunctionLookup(t *testing.T) {
	a := require.New(t)

	source := `
function test(x, y) {return x + y;}
function logic() {let a = test(1, 2); return 1;}
`
	result, errors := Parse(source)
	a.NotEmpty(result)
	a.Empty(errors)

	source = "function logic() {let a = test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident 'test' not defined")

	source = `
function logic() {let a = test(1); return 1;}
`
	result, errors = Parse(source)
	a.Empty(result, errors)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "ident 'test' not defined")

	source = "let test = 1; function logic() {let a = test(); return 1;}"
	result, errors = Parse(source)
	a.Empty(result)
	a.NotEmpty(errors)
	a.Contains(errors[0].String(), "not a function")

	source = `
function test(x) {return x;}
function logic() {let a = test(); return 1;}
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
function logic() {let x = test(1, 2); return 1;}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function test(x, y) {
	if (x) {return x + y;}
	else {return "a";}
}
function logic() {let x = test(1, 2); return 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `block types mismatch: uint64 vs byte[]`)

	source = `
function test(x, y) {return x + y;}
function logic() {let x = "abc"; x = test(1, 2); return 1;}
`

	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: (var) byte[] vs uint64 (expr)`)
}

func TestFunctionArgsType(t *testing.T) {
	a := require.New(t)
	source := `
function condition(a) {
	let b = if a == 1 { 10 } else { 0 }

    if b == 0 {
        return a
    }
    return 1
}

function logic() {
	let a = condition(1)
    return 1
}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)
}

func TestFunctionGlobals(t *testing.T) {
	a := require.New(t)
	source := `
function test(a) {
	if a == txn.Sender {
		return 0
	}
	if a == txn.ApplicationArgs[0] {
		return 0
	}
	const idx = 1
	if a == txn.ApplicationArgs[idx] {
		return 0
	}
	if a == gtxn[0].Sender {
		return 0
	}
	if a == gtxn[idx].Sender {
		return 0
	}
	if a == gtxn[0].ApplicationArgs[0] {
		return 0
	}
	if a == gtxn[0].ApplicationArgs[idx] {
		return 0
	}
	if global.MinTxnFee == 100 {
		return 0
	}
	let b = gtxn[idx].Sender

	const reimburseTxIndex = 0
	let reimburseReceiver = gtxn[reimburseTxIndex].Receiver
    return 1
}

function logic() {
	let a = test("abc")
    return 1
}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)
}

func TestBuiltinFunction(t *testing.T) {
	a := require.New(t)
	source := `
function logic() {let x = sha256(1) ; return 1;}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `incompatible types: (exp) byte[] vs uint64 (actual) in expr 'sha256 ([1])'`)

	source = `
function logic() {let x = 1; x = sha256("abc") ; return 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, fmt.Sprintf(`incompatible types: (var) uint64 vs byte[] (expr)`))

}

func TestMainReturn(t *testing.T) {
	a := require.New(t)
	source := `
function logic() {let x = 1;}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `main function does not return`)

	source = `
function approval() {let x = 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `main function does not return`)

	source = `
function clearstate() {let x = 1;}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `main function does not return`)

	source = `
function logic() {return "test";}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `main function must return int but got byte[]`)

	source = `
function logic() {
	let a = 1;
	if a == 1 {
		return 1;
	} else {
		return 0;
	}
}
`
	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function logic() {
	let a = 1;
	if a == 1 {
		return 1;
	}
	return 0;
}
`
	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

}

func TestDoubleVariable(t *testing.T) {
	a := require.New(t)

	source := "function logic() {let x = 1; let x = 2; return 1;}"
	result, errors := Parse(source)
	a.Empty(result, errors)
	a.NotEmpty(errors)

	source = "let x = 1; function logic() {let x = 2; return 1;}"
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
}

func TestDoubleScopeVariable(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let x = 2;
	if 1 {
		let x = 3;
	} else {
		let x = 4;
	}
	let y = 5;
	return 1;
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

func TestImportsDefault(t *testing.T) {
	a := require.New(t)
	source := `
import test
function logic() {return 1;}
`
	result, parserErrors := Parse(source)
	a.Empty(result, parserErrors)
	a.NotEmpty(parserErrors, parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `module test not found`)

	source = `
import stdlib.const
import stdlib.noop
function logic() { let type = TxTypePayment; type = NoOp(); return 1;}
`
	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
}

func TestImportsTemplate(t *testing.T) {
	a := require.New(t)
	source := `
import stdlib.const
import stdlib.templates
function logic() {
	let type = TxTypePayment
	let result = DynamicFee("abc", 10, "xyz", 1, 1000, "mylease")
	return result
}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
}

func TestImports(t *testing.T) {
	a := require.New(t)
	source := `
import test
function logic() {let x = test(); return 1;}
`
	module := `
function test() {
	if txn.Sender == "abc" {
		return global.MinBalance
	}
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	}
	return 0
}
`
	result, parserErrors := parseTestProgModule(source, module)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
}

func TestOneLineCond(t *testing.T) {
	a := require.New(t)
	source := `(1+2) >= 3 && txn.Sender == "abc"`
	result, parserErrors := ParseOneLineCond(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
}

func TestBinOpArgType(t *testing.T) {
	a := require.New(t)

	source := `1 == "abc"`
	result, parserErrors := ParseOneLineCond(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Contains(parserErrors[0].String(), "incompatible types: 'uint64' vs 'byte[]' in expr '1 == \"abc\"'")
}

func TestBuiltinDeclaration(t *testing.T) {
	a := require.New(t)

	source := `let global = 1
const gtxn = 2
function txn() { return 1; }
function sha256(x) { return x; }
function logic() {
	const sha512_256 = 1
	let args = 2
}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(3, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `no viable alternative at input 'let global'`)
	a.Contains(parserErrors[1].msg, `no viable alternative at input 'const gtxn'`)
	a.Contains(parserErrors[2].msg, `no viable alternative at input 'function txn'`)

	source = `function sha256(x) { return x; }
function logic() {
	const sha512_256 = 1
	let args = 2
}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `no viable alternative at input 'function sha256'`)

	source = `function logic() {
	const sha512_256 = 1
	let args = 2
}
`
	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(2, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `no viable alternative at input 'const sha512_256'`)
	a.Contains(parserErrors[1].msg, `no viable alternative at input 'let args'`)
}

func TestBuiltinFuncArgsNumber(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let a = sha256("test", 1)
}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `can't get type for sha256 arg #2`)
}

func TestBuiltinMulw(t *testing.T) {
	a := require.New(t)

	source := `
let a, b = mulw(1, 2)
function logic() {
	a, b = mulw(3, 4)
	if a == b {
		return 0
	}
	a, b = addw(5, 6)
	a, b = expw(3, 2)
	return 1
}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)
}

func TestFunctionReturn(t *testing.T) {
	a := require.New(t)
	source := `
function logic() {let x = 1;}
`
	result, parserErrors := Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `main function does not return`)

	source = `
function test() {
	if txn.Sender == "abc" {
		return global.MinBalance
	}
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	}
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `test function does not return`)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	} else {
		let x = 2;
	}
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `test function does not return`)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		let x = 1
	} else {
		return 0
	}
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.Empty(result)
	a.NotEmpty(parserErrors)
	a.Equal(1, len(parserErrors), parserErrors)
	a.Contains(parserErrors[0].msg, `test function does not return`)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	} else {
		return 0
	}
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	} else {
		return 0
	}
	return 1
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		return txn.FirstValid
	}
	return 1
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		error
	}
	return 1
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)

	source = `
function test() {
	if gtxn[1].Sender == "abc" {
		return 1
	}
	error
}
function logic() {return test();}
`

	result, parserErrors = Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)
}

func TestBuiltinApp(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let b = accounts[1].Balance;
	let exist = accounts[0].optedIn(1);
	let val = accounts[1].get("key")
	val, exist = accounts[1].getEx(0, "key")
	if exist == 0 {
		return 0;
	}

	val = apps[0].get("key")
	val, exist = apps[1].getEx("key");
	if exist == 0 {
		return 0;
	}

	let senderIdx = 0;
	accounts[senderIdx].put("key", 1)
	let value = "value";
	let key = "key";
	apps[0].put(key, value)

	const acc = 1;
	accounts[acc].del(key)
	apps[0].del(key)

	let asset = 100;
	let account = 1;
	let amount, isok = accounts[account].assetBalance(asset)
	let frozen, isok2 = accounts[0].assetIsFrozen(1)
	let assetIdx = 0;
	amount, exist = assets[assetIdx].AssetTotal
	return 1;
}
`
	result, parserErrors := Parse(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors)
}
