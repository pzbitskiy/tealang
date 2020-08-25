package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodegenVariables(t *testing.T) {
	a := require.New(t)

	source := `let a = 1; let b = "123"; function logic() {a = 5; return 6;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 5 6", lines[0]) // 0 and 1 are added internally
	a.Equal("bytecblock 0x313233", lines[1])

	lastLine := len(lines) - 1
	a.Equal("intc 2", lines[lastLine-8])  // a = 5 (a's address is 0, 5's offset is 2)
	a.Equal("store 0", lines[lastLine-7]) //
	a.Equal("intc 3", lines[lastLine-6])  // ret 6 (6's offset is 3)
	a.Equal("intc 1", lines[lastLine-5])
	a.Equal("bnz end_main", lines[lastLine-4])
	a.Equal("end_main:", lines[lastLine-3])
	a.Equal("dup", lines[lastLine-2])
	a.Equal("pop", lines[lastLine-1])
	a.Equal(fmt.Sprintf(""), lines[lastLine]) // import fmt
}

func TestCodegenErr(t *testing.T) {
	a := require.New(t)

	source := `function logic() {error;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0]) // 0 and 1 are added internally
	a.Equal("err", lines[1])
}

func TestCodegenBinOp(t *testing.T) {
	a := require.New(t)

	source := `const c = 10; function logic() {let a = 1 + c; let b = !a; return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10", lines[0]) // 0 and 1 are added internally
	a.Equal("// const", lines[1])
	a.Equal("intc 1", lines[2])
	a.Equal("intc 2", lines[3])
	a.Equal("+", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("load 0", lines[6])
	a.Equal("!", lines[7])
	a.Equal("store 1", lines[8])
}

func TestCodegenIfExpr(t *testing.T) {
	a := require.New(t)

	source := `let x = if 1 { 2 } else { 3 }; function logic() {return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_expr_false_", lines[3][:len("bnz if_expr_false_")])
	a.Equal("intc 2", lines[4])
	a.Equal("intc 1", lines[5])
	a.Equal("bnz if_expr_end_", lines[6][:len("bnz if_expr_end_")])
	a.Equal("if_expr_false_", lines[7][:len("if_expr_false_")])
	a.Equal("intc 3", lines[8])
	a.Equal("if_expr_end_", lines[9][:len("if_expr_end_")])
	a.Equal("store 0", lines[10])
}

func TestCodegenIfStmt(t *testing.T) {
	a := require.New(t)

	source := `function logic() { if 1 {let x=10;} return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_stmt_end_", lines[3][:len("bnz if_stmt_end_")])
	a.Equal("intc 2", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("if_stmt_end_", lines[6][:len("if_stmt_end_")])

	source = `function logic() { if 1 {let x=10;} else {let y=11;} return 1;}`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog = Codegen(result)
	lines = strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 10 11", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("!", lines[2])
	a.Equal("bnz if_stmt_false_", lines[3][:len("bnz if_stmt_false_")])
	a.Equal("intc 2", lines[4])
	a.Equal("store 0", lines[5])
	a.Equal("intc 1", lines[6])
	a.Equal("bnz if_stmt_end_", lines[7][:len("bnz if_stmt_end_")])
	a.Equal("if_stmt_false_", lines[8][:len("if_stmt_false_")])
	a.Equal("intc 3", lines[9])
	a.Equal("store 0", lines[10])
	a.Equal("if_stmt_end_", lines[11][:len("if_stmt_end_")])
}

func TestCodegenGlobals(t *testing.T) {
	a := require.New(t)

	source := `function logic() {let glob = global.MinTxnFee; let g = gtxn[1].Sender; let a = args[0]; return 1;}`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("global MinTxnFee", lines[1])
	a.Equal("store 0", lines[2])
	a.Equal("gtxn 1 Sender", lines[3])
	a.Equal("store 1", lines[4])
	a.Equal("arg 0", lines[5])
	a.Equal("store 2", lines[6])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("store 0", lines[2])
	a.Equal("load 0", lines[3])
	a.Equal("store 1", lines[4])
	a.Equal("intc 2", lines[5])
	a.Equal("store 2", lines[6])
	a.Equal("load 1", lines[7])
	a.Equal("load 2", lines[8])
	a.Equal("+", lines[9])
	a.Equal("intc 1", lines[10])
	a.Equal("bnz end_sum", lines[11])
	a.Equal("end_sum:", lines[12])
	a.Equal("store 1", lines[13])
	a.Equal("intc 3", lines[14])
	a.Equal("store 2", lines[15])
	a.Equal("intc 1", lines[16])
	a.Equal("intc 1", lines[17])
	a.Equal("bnz end_main", lines[18])
	a.Equal("end_main:", lines[19])
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
	a.Empty(errors)
	result.Print()
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3 4", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("intc 2", lines[2])
	a.Equal("+", lines[3])
	a.Equal("intc 3", lines[4])
	a.Equal("intc 4", lines[5])
	a.Equal("-", lines[6])
	a.Equal("/", lines[7])
	a.Equal("store 0", lines[8])
	a.Equal("load 0", lines[9])
	a.Equal("intc 1", lines[10])
	a.Equal("bnz end_main", lines[11])
	a.Equal("end_main:", lines[12])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3 4 5 6", lines[0])
	a.Equal("intc 1", lines[1]) // TxTypePayment
	a.Equal("store 0", lines[2])
	a.Equal("intc 0", lines[3]) // NoOp -> ret 0
	a.Equal("intc 1", lines[4])
	a.Equal("bnz end_NoOp", lines[5])
	a.Equal("end_NoOp:", lines[6])
	a.Equal("store 0", lines[7])
	a.Equal("intc 1", lines[8])
	a.Equal("intc 1", lines[9])
	a.Equal("bnz end_main", lines[10])
	a.Equal("end_main:", lines[11])
}

func TestCodegenOneLineCond(t *testing.T) {
	a := require.New(t)
	source := `(1+2) >= 3 && txn.Sender == "123"`
	result, parserErrors := ParseOneLineCond(source)
	a.NotEmpty(result, parserErrors)
	a.Empty(parserErrors, parserErrors)
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3", lines[0])
	a.Equal("bytecblock 0x313233", lines[1])
	a.Equal("intc 1", lines[2])
	a.Equal("intc 2", lines[3]) // NoOp -> ret 0
	a.Equal("+", lines[4])
	a.Equal("intc 3", lines[5])
	a.Equal(">=", lines[6])
	a.Equal("txn Sender", lines[7])
	a.Equal("bytec 0", lines[8])
	a.Equal("==", lines[9])
	a.Equal("&&", lines[10])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("store 0", lines[2])
	a.Equal("intc 2", lines[3])
	a.Equal("store 1", lines[4])
	a.Equal("intc 1", lines[5])
	a.Equal("!", lines[6])
	a.Equal("bnz if_stmt_end_", lines[7][:len("bnz if_stmt_end_")])
	a.Equal("intc 3", lines[8])
	a.Equal("store 2", lines[9])
	a.Equal("if_stmt_end_", lines[10][:len("if_stmt_end_")])
	a.Equal("load 1", lines[11])
	a.Equal("intc 1", lines[12])
	a.Equal("bnz end_main", lines[13])
	a.Equal("end_main:", lines[14])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("intc 1", lines[2])
	a.Equal("bnz end_test1", lines[3])
	a.Equal("end_test1:", lines[4])
	a.Equal("intc 1", lines[5])
	a.Equal("bnz end_test2", lines[6])
	a.Equal("end_test2:", lines[7])
	a.Equal("intc 1", lines[8])
	a.Equal("bnz end_main", lines[9])
	a.Equal("end_main:", lines[10])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0])
	a.Equal("bytecblock 0x"+strings.Repeat("00", 32), lines[1])
	a.Equal("bytec 0", lines[2])
	a.Equal("store 0", lines[3])
	a.Equal("intc 0", lines[4])
	a.Equal("intc 1", lines[5])
	a.Equal("bnz end_main", lines[6])
	a.Equal("end_main:", lines[7])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2 3 4 5 6", lines[0])
	a.Equal("intc 1", lines[1])
	a.Equal("intc 2", lines[2])
	a.Equal("mulw", lines[3])
	a.Equal("store 0", lines[4]) // store low
	a.Equal("store 1", lines[5]) // store high
	a.Equal("intc 3", lines[6])
	a.Equal("intc 4", lines[7])
	a.Equal("mulw", lines[8])
	a.Equal("store 0", lines[9])
	a.Equal("store 1", lines[10])
	a.Equal("intc 5", lines[11])
	a.Equal("intc 6", lines[12])
	a.Equal("addw", lines[13])
	a.Equal("store 2", lines[14])
	a.Equal("store 3", lines[15])
	a.Equal("load 0", lines[16])
	a.Equal("intc 1", lines[17])
	a.Equal("bnz end_main", lines[18])
	a.Equal("end_main:", lines[19])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0])
	a.Equal("bytecblock 0x6b6579", lines[1])
	a.Equal("intc 1", lines[2])
	a.Equal("intc 0", lines[3])
	a.Equal("bytec 0", lines[4])
	a.Equal("app_local_get_ex", lines[5])
	a.Equal("store 0", lines[6])
	a.Equal("store 1", lines[7])
	a.Equal("load 0", lines[8])
	a.Equal("intc 1", lines[9])
	a.Equal("bnz end_main", lines[10])
	a.Equal("end_main:", lines[11])

	source = `
function approval() {
	app_local_put(0, "key", 1);
	return 1;
}
`
	result, errors = Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog = Codegen(result)
	fmt.Print(prog)
	lines = strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0])
	a.Equal("bytecblock 0x6b6579", lines[1])
	a.Equal("intc 0", lines[2])
	a.Equal("bytec 0", lines[3])
	a.Equal("intc 1", lines[4])
	a.Equal("app_local_put", lines[5])
	a.Equal("intc 1", lines[6])
	a.Equal("intc 1", lines[7])
	a.Equal("bnz end_main", lines[8])
	a.Equal("end_main:", lines[9])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 100", lines[0])
	a.Equal("intc 2", lines[1])
	a.Equal("store 0", lines[2])
	a.Equal("intc 1", lines[3])
	a.Equal("store 1", lines[4])
	a.Equal("load 0", lines[5])
	a.Equal("load 1", lines[6])
	a.Equal("asset_holding_get AssetBalance", lines[7])
	a.Equal("store 2", lines[8])
	a.Equal("store 3", lines[9])
	a.Equal("load 2", lines[10])
	a.Equal("intc 1", lines[11])
	a.Equal("bnz end_main", lines[12])
	a.Equal("end_main:", lines[13])
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
	prog := Codegen(result)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1", lines[0])
	a.Equal("bytecblock 0x616263 0x646566", lines[1])
	a.Equal("bytec 0", lines[2])
	a.Equal("store 0", lines[3])
	a.Equal("bytec 1", lines[4])
	a.Equal("store 1", lines[5])
	a.Equal("load 0", lines[6])
	a.Equal("load 1", lines[7])
	a.Equal("concat", lines[8])
	a.Equal("store 2", lines[9])
	a.Equal("load 2", lines[10])
	a.Equal("len", lines[11])
	a.Equal("intc 1", lines[12])
	a.Equal("bnz end_main", lines[13])
	a.Equal("end_main:", lines[14])
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
	prog := Codegen(result)
	fmt.Print(prog)
	lines := strings.Split(prog, "\n")
	a.Equal("intcblock 0 1 2", lines[0])
	a.Equal("bytecblock 0x616263", lines[1])
	a.Equal("bytec 0", lines[2])
	a.Equal("intc 1", lines[3])
	a.Equal("intc 2", lines[4])
	a.Equal("substring3", lines[5])
	a.Equal("store 0", lines[6])
	a.Equal("load 0", lines[7])
	a.Equal("len", lines[8])
	a.Equal("intc 1", lines[9])
	a.Equal("bnz end_main", lines[10])
	a.Equal("end_main:", lines[11])
}
