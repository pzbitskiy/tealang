package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func CompareTEAL(a *require.Assertions, expected string, actual string) {
	exp := strings.Split(expected, "\n")
	act := strings.Split(actual, "\n")
	n := len(exp)
	if len(act) < n {
		a.Fail(fmt.Sprintf("program expected to be at least %d lines long but got %d", n, len(act)))
	}
	for i := 0; i < n; i++ {
		if len(exp[i]) == 0 {
			a.Empty(act[i], fmt.Sprintf("line %d not empty: %s", i+1, act[i]))
			continue
		} else if exp[i][len(exp[i])-1] == '*' {
			a.Equal(exp[i][:len(exp[i])-1], act[i][:len(exp[i])-1])
		} else {
			a.Equal(exp[i], act[i])
		}
	}
	a.Equal("", act[len(act)-1])
}

func TestCodegenVariables(t *testing.T) {
	a := require.New(t)

	source := `let a = 1; let b = "123"; function logic() {a = 5; return 6;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	// 0 and 1 are added internally
	// a = 5 (a's address is 0, 5's offset is 2)
	// ret 6 (6's offset is 3)
	expected := `#pragma version *
intcblock 0 1 5 6
bytecblock 0x313233
intc 1
store 0
bytec 0
store 1
intc 2
store 0
intc 3
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenErr(t *testing.T) {
	a := require.New(t)

	source := `function logic() {error;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
err`
	CompareTEAL(a, expected, actual)
}

func TestCodegenBinOp(t *testing.T) {
	a := require.New(t)

	source := `const c = 10; function logic() {let a = 1 + c; let b = !a; return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 10
// const
intc 1
intc 2
+
store 0
load 0
!
store 1`
	CompareTEAL(a, expected, actual)
}

func TestCodegenIfExpr(t *testing.T) {
	a := require.New(t)

	source := `let x = if 1 { 2 } else { 3 }; function logic() {return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
intc 1
!
bnz if_expr_false_*
intc 2
intc 1
bnz if_expr_end_*
if_expr_false_*
intc 3
if_expr_end_*
store 0`

	CompareTEAL(a, expected, actual)
}

func TestCodegenIfStmt(t *testing.T) {
	a := require.New(t)

	source := `function logic() { if 1 {let x=10;} return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 10
intc 1
!
bnz if_stmt_end_*
intc 2
store 0
if_stmt_end_*`

	CompareTEAL(a, expected, actual)

	source = `function logic() { if 1 {let x=10;} else {let y=11;} return 1;}`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 10 11
intc 1
!
bnz if_stmt_false_*
intc 2
store 0
intc 1
bnz if_stmt_end_*
if_stmt_false_*
intc 3
store 0
if_stmt_end_*`

	CompareTEAL(a, expected, actual)
}

func TestCodegenGlobals(t *testing.T) {
	a := require.New(t)

	source := `function logic() {let glob = global.MinTxnFee; let g = gtxn[1].Sender; let a = args[0]; return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
*
global MinTxnFee
store 0
gtxn 1 Sender
store 1
arg 0
store 2`
	CompareTEAL(a, expected, actual)
}

func TestCodegenFunCall(t *testing.T) {
	a := require.New(t)

	source := `
function sum(x, y) { return x + y; }
function logic() {
	let a = 1
	let b = sum (a, 2)
	let x = 3
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
intc 1
store 0
load 0
store 1
intc 2
store 2
load 1
load 2
+
intc 1
bnz end_sum
end_sum:
store 1
intc 3
store 2
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenGeneric(t *testing.T) {
	a := require.New(t)

	source := `
let a = 456;
const b = "123";
let c = 1 + 2 * (2 + 3);
let d = 1 + 2
let e = if c == 1 {1} else {2}

function test(x, y) {
	return x + y
}

function test1(x) {
	return !x
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

	let z = sha256("test")

	let f = test(20+2, 30)
	if f + 2 < 10 {
		error
	}
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	if len(errors) != 0 {
		result.Print()
	}
	a.Empty(errors)
	prog := Codegen(result)
	a.Greater(len(prog), 0)
}

func TestCodegenOpsPriority(t *testing.T) {
	a := require.New(t)

	source := `
let a = (1 + 2) / (3 - 4)
function logic() { return a; }
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3 4
intc 1
intc 2
+
intc 3
intc 4
-
/
store 0
load 0
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenImportStdlib(t *testing.T) {
	a := require.New(t)

	source := `
import stdlib.const
import stdlib.noop
function logic() { let type = TxTypePayment; type = NoOp(); return 1;}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3 4 5 6
intc 1
store 0
intc 0
intc 1
bnz end_NoOp
end_NoOp:
store 0
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenOneLineCond(t *testing.T) {
	a := require.New(t)
	source := `(1+2) >= 3 && txn.Sender == "123"`
	result, parserErrors := ParseOneLineCond(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
bytecblock 0x313233
intc 1
intc 2
+
intc 3
>=
txn Sender
bytec 0
==
&&
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenShadow(t *testing.T) {
	a := require.New(t)

	source := `
let x = 1
function logic() {
	let x = 2       // shadows 1 in logic block
	if 1 {
		let x = 3   // shadows 2 in if-block
	}
	return x        // 2
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
intc 1
store 0
intc 2
store 1
intc 1
!
bnz if_stmt_end_*
intc 3
store 2
if_stmt_end_*
load 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenNestedFun(t *testing.T) {
	a := require.New(t)

	source := `
function test1() { return 1; }
function test2() { return test1(); }
function logic() {
	return test2()
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
intc 1
intc 1
bnz end_test1
end_test1:
intc 1
bnz end_test2
end_test2:
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestAddressStringLiteralDecoding(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let a = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
	return 0
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := fmt.Sprintf(`#pragma version *
intcblock 0 1
bytecblock 0x%s
bytec 0
store 0
intc 0
return
end_main:
`, strings.Repeat("00", 32))
	CompareTEAL(a, expected, actual)
}

func TestCodegenMulw(t *testing.T) {
	a := require.New(t)

	source := `
let h, l = mulw(1, 2)
function logic() {
	h, l = mulw(3, 4)
	let a, b = addw(5, 6)
	return l
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3 4 5 6
intc 1
intc 2
mulw
store 0
store 1
intc 3
intc 4
mulw
store 0
store 1
intc 5
intc 6
addw
store 2
store 3
load 0
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenApp(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let val, exist = app_local_get_ex(1, 0, "key");
	return exist;
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
bytecblock 0x6b6579
intc 1
intc 0
bytec 0
app_local_get_ex
store 0
store 1
load 0
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	app_local_put(0, "key", 1);
	return 1;
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1
bytecblock 0x6b6579
intc 0
bytec 0
intc 1
app_local_put
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenAsset(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let asset = 100;
	let acc = 1;
	let amount, exist = asset_holding_get(AssetBalance, asset, acc);
	return exist;
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 100
intc 2
store 0
intc 1
store 1
load 0
load 1
asset_holding_get AssetBalance
store 2
store 3
load 2
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenConcat(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let a1 = "abc"
	let a2 = "def"
	let result = concat(a1, a2)
	return len(result)
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
bytecblock 0x616263 0x646566
bytec 0
store 0
bytec 1
store 1
load 0
load 1
concat
store 2
load 2
len
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenSubstring(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let result = substring3("abc", 1, 2)
	return len(result)
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2
bytecblock 0x616263
bytec 0
intc 1
intc 2
substring3
store 0
load 0
len
return
end_main:
`
	CompareTEAL(a, expected, actual)
}
