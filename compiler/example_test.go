package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExampleCompile(t *testing.T) {
	a := require.New(t)
	source := `
let variable1 = 1
const myaddr = "XYZ"
let a = (1 + 2) / 3
let b = ~a
function sample(a) {
    return a - 1
}
function condition(a) {
    let b = if a == 1 { 10 } else { 0 }

    if b == 0 {
        return a
    }
    return 1
}
function get_string() {
    return "\x32\x33\x34"
}

function logic() {
	let a = condition(1)
	let b = get_string()
    return sample(2)
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	a.NotEmpty(prog)
}

func TestGuideCompile(t *testing.T) {
	a := require.New(t)
	source := `
import stdlib.const

const a = 1
const b = "abc\x01"
let x = b
function test(x) { return x; }
let sender = if global.GroupSize > 1 { txn.Sender } else { gtxn[1].Sender }
const zeroAddress = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
const someval = b64"MTIz"

function inc(x) { return x+1; }
const myconst = 1
function myfunction() { return 0; }

function logic() {
    if txn.Sender == "ABC" {
		return 1
	}

    let x = 2       // shadows 1 in logic block
    if 1 {
        let x = 3   // shadows 2 in if-block
    }
    return x        // 2

	if x == 1 {
		return 1
	} else {
		let x = txn.Receiver
	}

    if txn.Receiver == zeroAddress {
        return 1
    }

	return inc(0);

	let ret = TxTypePayment
	return ret
}
`
	result, errors := Parse(source)
	a.NotEmpty(result, errors)
	a.Empty(errors)
	prog := Codegen(result)
	a.NotEmpty(prog)
}
