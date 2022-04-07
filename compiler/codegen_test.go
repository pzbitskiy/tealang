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
			a.Equal(exp[i][:len(exp[i])-1], act[i][:len(exp[i])-1], fmt.Sprintf("line %d: %s != %s", i+1, exp[i], act[i]))
		} else {
			a.Equal(exp[i], act[i], fmt.Sprintf("line %d: %s != %s", i+1, exp[i], act[i]))
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
fun_main:
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
fun_main:
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
fun_main:
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
bz if_expr_false_*
intc 2
b if_expr_end_*
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
fun_main:
intc 1
bz if_stmt_end_*
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
fun_main:
intc 1
bz if_stmt_false_*
intc 2
store 0
b if_stmt_end_*
if_stmt_false_*
intc 3
store 0
if_stmt_end_*`

	CompareTEAL(a, expected, actual)
}

func TestCodegenGlobals(t *testing.T) {
	a := require.New(t)

	source := `function logic() {
let glob = global.MinTxnFee;
let g = gtxn[1].Sender;
let a = args[0];
let b = txn.ApplicationArgs[0]
let c = gtxn[1].Assets[0]
a = args[toint(a)+1];
return 1;
}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
*
fun_main:
global MinTxnFee
store 0
gtxn 1 Sender
store 1
arg 0
store 2
txna ApplicationArgs 0
store 3
gtxna 1 Assets 0
store 4
load 2
intc 1
+
args
store 2
intc 1`
	CompareTEAL(a, expected, actual)
}

func TestCodegenTxn(t *testing.T) {
	a := require.New(t)

	source := `function logic() {
let a = txn.ApplicationArgs[0]
let idx = 1
let b = txn.ApplicationArgs[idx+1]
return 1;
}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
*
fun_main:
txna ApplicationArgs 0
store 0
intc 1
store 1
load 1
intc 1
+
txnas ApplicationArgs
store 2
intc 1`
	CompareTEAL(a, expected, actual)
}

func TestCodegenGtxn(t *testing.T) {
	a := require.New(t)

	source := `function logic() {
let a = gtxn[0].Sender;
let idx = 1;
let b = gtxn[idx].Sender;
let c = gtxn[idx+1].Sender;
return 1;
}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
*
fun_main:
gtxn 0 Sender
store 0
intc 1
store 1
load 1
gtxns Sender
store 2
load 1
intc 1
+
gtxns Sender
store 3
intc 1`
	CompareTEAL(a, expected, actual)

	source = `function logic() {
let idx = 1;
let a = gtxn[0].ApplicationArgs[1];
let b = gtxn[0].ApplicationArgs[idx];
let c = gtxn[idx].ApplicationArgs[1];
let d = gtxn[idx].ApplicationArgs[idx+2];
return 1;
}`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
*
fun_main:
intc 1
store 0
gtxna 0 ApplicationArgs 1
store 1
load 0
gtxnas 0 ApplicationArgs
store 2
load 0
gtxnsa ApplicationArgs 1
store 3
load 0
load 0
intc 2
+
gtxnsas ApplicationArgs
store 4
intc 1`
	CompareTEAL(a, expected, actual)

}

func TestCodegenFunCallInline(t *testing.T) {
	a := require.New(t)

	source := `
inline function sum(x, y) { return x + y; }
function logic() {
	let a = 1
	let b = sum (a, 2)
	let x = 3
	let c = sum (x, 1)
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
fun_main:
intc 1
store 0
load 0
store 1
intc 2
store 2
load 1
load 2
+
b end_sum_*
end_sum_*
store 1
intc 3
store 2
load 2
store 3
intc 1
store 4
load 3
load 4
+
b end_sum_*
end_sum_*
store 3
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
	lines := strings.Split(actual, "\n")
	a.Contains(lines[12], "b end_sum_")
	a.Contains(lines[13], "end_sum_")
	a.Contains(lines[24], "b end_sum_")
	a.Contains(lines[25], "end_sum_")
	a.NotEqual(lines[12], lines[24])
	a.NotEqual(lines[13], lines[25])
	a.True(lines[13][len(lines[13])-1] == ':')
	a.True(lines[25][len(lines[25])-1] == ':')
}

func TestCodegenFunCall(t *testing.T) {
	a := require.New(t)

	source := `
function sum(x, y) { return x + y; }
function logic() {
	let a = 1
	let b = sum (a, 2)
	let x = 3
	let c = sum (x, 1)
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2 3
fun_main:
intc 1
store 0
load 0
intc 2
callsub fun_sum
store 1
intc 3
store 2
load 2
intc 1
callsub fun_sum
store 3
intc 1
return
end_main:
fun_sum:
store 4
store 3
load 3
load 4
+
retsub
end_sum:
`
	CompareTEAL(a, expected, actual)
	lines := strings.Split(actual, "\n")
	a.Equal(lines[7], lines[13])               // callsub func_sum_*
	a.True(lines[18][len(lines[18])-1] == ':') // func_sum_*:
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
	let p = txn.ExtraProgramPages
	r = t

	let z = sha256("test")

	let f = test(20+2, 30)
	if f + 2 < 10 {
		error
	} else {
		f = exp(2, 3)
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
fun_main:
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
fun_main:
intc 1
store 0
intc 0
b end_NoOp_*
end_NoOp_*
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
fun_main:
intc 2
store 1
intc 1
bz if_stmt_end_*
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
inline function test1() { return 1; }
inline function test2() { return test1(); }
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
fun_main:
intc 1
b end_test1_*
end_test1_*
b end_test2_*
end_test2_*
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
fun_main:
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
	let c, d = expw(3, 4)
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
fun_main:
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
intc 3
intc 4
expw
store 4
store 5
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
	let val, exist = accounts[1].getEx(0, "key");
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
fun_main:
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
	return accounts[2].get("key");
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2
bytecblock 0x6b6579
fun_main:
intc 2
bytec 0
app_local_get
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	accounts[0].put("key", 1);
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
fun_main:
intc 0
bytec 0
intc 1
app_local_put
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	return apps[0].get("key")
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1
bytecblock 0x6b6579
fun_main:
bytec 0
app_global_get
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	apps[0].put("key", 2)
	return 1
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2
bytecblock 0x6b6579
fun_main:
bytec 0
intc 2
app_global_put
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	let val, exist = apps[2].getEx("key")
	return val
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2
bytecblock 0x6b6579
fun_main:
intc 2
bytec 0
app_global_get_ex
store 0
store 1
load 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenAppParams(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let app = 1;
	let amount, exist = apps[app].AppExtraProgramPages;
	return exist;
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
fun_main:
intc 1
store 0
load 0
app_params_get AppExtraProgramPages
store 1
store 2
load 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenAppAccounts(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let b = accounts[1].Balance
	let m = accounts[0].MinimumBalance
	let ok = 0
	b, ok = accounts[1].acctBalance()
	m, ok = accounts[m].acctMinBalance()
	let a = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
	let u = ""
	u, ok = accounts[a].acctAuthAddr()
	return b + m
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	fmt.Println(actual)
	expected := `#pragma version *
intcblock 0 1
*
fun_main:
intc 1
balance
store 0
intc 0
min_balance
store 1
intc 0
store 2
intc 1
acct_params_get AcctBalance
store 2
store 0
load 1
acct_params_get AcctMinBalance
store 2
store 1
bytec 0
store 3
bytec 1
store 4
load 3
acct_params_get AcctAuthAddr
store 2
store 4
load 0
load 1
+
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	return accounts[2].optedIn(101)
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2 101
fun_main:
intc 2
intc 3
app_opted_in
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
	let amount, exist = accounts[acc].assetBalance(asset);
	return exist;
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 100
fun_main:
intc 2
store 0
intc 1
store 1
load 1
load 0
asset_holding_get AssetBalance
store 2
store 3
load 2
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	let amount, exist = accounts[2].assetIsFrozen(101);
	return exist;
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2 101
fun_main:
intc 2
intc 3
asset_holding_get AssetFrozen
store 0
store 1
load 0
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	let amount, exist = assets[0].AssetTotal
	return amount;
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1
fun_main:
intc 0
asset_params_get AssetTotal
store 0
store 1
load 1
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
fun_main:
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
	let start = 1
	let result = substring("abc", start, 2)
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
fun_main:
intc 1
store 0
bytec 0
load 0
intc 2
substring3
store 1
load 1
len
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function logic() {
	let result = substring("abc", 1, 2)
	return len(result)
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 2
bytecblock 0x616263
fun_main:
bytec 0
substring 1 2
store 0
load 0
len
return
end_main:
`
	CompareTEAL(a, expected, actual)

}

func TestLoop(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let y= 2;
	for y>0 { y=y-1 }
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 2
fun_main:
intc 2
store 0
loop_start_*
load 0
intc 0
>
bz loop_end_*
load 0
intc 1
-
store 0
b loop_start_*
loop_end_*
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestBreak(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let y= 0;
	for 1 {
		if y==10 {break;}
		y=y+1
	}
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 10
fun_main:
intc 0
store 0
loop_start_*
intc 1
bz loop_end_*
load 0
intc 2
==
bz if_stmt_end_*
bz loop_end_*
if_stmt_end_*
load 0
intc 1
+
store 0
b loop_start_*
loop_end_*
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenGetSetBitByte(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let a = getbit(255, 1)
	let b = getbit("\xFF", 2)
	let c = getbyte("test", 0)
	return a + b + c
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 255 2
bytecblock 0xff 0x74657374
fun_main:
intc 2
intc 1
getbit
store 0
bytec 0
intc 3
getbit
store 1
bytec 1
intc 0
getbyte
store 2
load 0
load 1
+
load 2
+
return
end_main:
`
	CompareTEAL(a, expected, actual)

	source = `
function approval() {
	let a = setbit(0, 1, 1)
	let b = setbit("\xFF", 1, 0)
	let c = setbyte("test", 0, 32)
	let d = btoi(b)
	let e = btoi(c)
	return a + d + e
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual = Codegen(result)
	expected = `#pragma version *
intcblock 0 1 32
bytecblock 0xff 0x74657374
fun_main:
intc 0
intc 1
intc 1
setbit
store 0
bytec 0
intc 1
intc 0
setbit
store 1
bytec 1
intc 0
intc 2
setbyte
store 2
load 1
btoi
store 3
load 2
btoi
store 4
load 0
load 3
+
load 4
+
return
end_main:
`
	CompareTEAL(a, expected, actual)

}

func TestCodegenByteArith(t *testing.T) {
	a := require.New(t)

	source := `
function logic() { let z = bzero(4); let r = band(z, "\x11"); return 1;}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1 4
bytecblock 0x11
fun_main:
intc 2
bzero
store 0
load 0
bytec 0
b&
store 1
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenGaid(t *testing.T) {
	a := require.New(t)

	source := `
function logic() {
	let a = gaid(0)
	let h = gaid(a+1)
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
fun_main:
gaid 0
store 0
load 0
intc 1
+
gaids
store 1
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestToInt(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	let a = accounts[0].get("key")
	let b = toint(a) + 1
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)
	expected := `#pragma version *
intcblock 0 1
bytecblock 0x6b6579
fun_main:
intc 0
bytec 0
app_local_get
store 0
load 0
intc 1
+
store 1
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestInnerTxn(t *testing.T) {
	a := require.New(t)

	source := `
function approval() {
	itxn.begin()
	itxn.TypeEnum = 1
	itxn.Receiver = txn.Sender
	itxn.next()
	itxn.TypeEnum = 1
	itxn.Receiver = txn.Sender
	itxn.submit()
	let a = concat(itxn.Sender, "0")
	let c = itxn.ApplicationArgs[0]
	let b = itxn.Logs[c]
	let d = concat(gitxn[0].Sender, "1")
	let e = gitxn[1].ApplicationArgs[0]
	let f = gitxn[0].Logs[e]
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)

	fmt.Println(actual)

	expected := `#pragma version *
intcblock 0 1
bytecblock 0x30 0x31
fun_main:
itxn_begin
intc 1
itxn_field TypeEnum
txn Sender
itxn_field Receiver
itxn_next
intc 1
itxn_field TypeEnum
txn Sender
itxn_field Receiver
itxn_submit
itxn Sender
bytec 0
concat
store 0
itxna ApplicationArgs 0
store 1
load 1
itxnas Logs
store 2
gitxn 0 Sender
bytec 1
concat
store 3
gitxna 1 ApplicationArgs 0
store 4
load 4
gitxnas 0 Logs
store 5
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenLog(t *testing.T) {
	a := require.New(t)
	source := `
function logic() {
	log("Hi")
	return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)

	expected := `#pragma version *
intcblock 0 1
bytecblock 0x4869
fun_main:
bytec 0
log
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestCodegenEcdsa(t *testing.T) {
	a := require.New(t)
	source := `
function logic() {
	let res = ecdsa_verify(Secp256k1, "a", "b", "c", "d", "e")
	let d1, d2 = ecdsa_pk_decompress(Secp256k1, "a")
	d1, d2 = ecdsa_pk_recover(Secp256k1, "a", 1, "b", "c")
	return res
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)

	expected := `#pragma version *
intcblock 0 1
*
fun_main:
bytec 0
bytec 1
bytec 2
bytec 3
bytec 4
ecdsa_verify Secp256k1
store 0
bytec 0
ecdsa_pk_decompress Secp256k1
store 1
store 2
bytec 0
intc 1
bytec 1
bytec 2
ecdsa_pk_recover Secp256k1
store 1
store 2
load 0
return
end_main:
`
	CompareTEAL(a, expected, actual)
}

func TestExtract(t *testing.T) {
	a := require.New(t)

	source := `function logic() {
let a = extract("\x12\x34\x56\x78\x9a\xbc", 1, 2)

let s = 1
let e = 5
a = extract("\x12\x34\x56\x78\x9a\xbc", s, e)

let b = extract(UINT16, "\x12\x34\x56\x78\x9a\xbc", 1)
assert(b == 0x3456)
return 1
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	actual := Codegen(result)

	expected := `#pragma version *
intcblock 0 1 2 5 0x3456
bytecblock 0x123456789abc
fun_main:
bytec 0
extract 1 2
store 0
intc 1
store 1
intc 3
store 2
bytec 0
load 1
load 2
extract3
store 0
bytec 0
intc 1
extract_uint16
store 3
load 3
intc 4
==
assert
intc 1
return
end_main:
`
	CompareTEAL(a, expected, actual)
}
