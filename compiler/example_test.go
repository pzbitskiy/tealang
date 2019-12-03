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

function logic(txn, gtxn, args) {
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
