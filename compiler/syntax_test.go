package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TesValidProgram(t *testing.T) {
	source := `
let a = 456; const b = 123; const c = "1234567890123";
let d = 1 + 2 ;
let e = if a > 0 {1} else {2}

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
`
	result := Compile(source)
	require.NotEmpty(t, result)
}

func TestInvalid(t *testing.T) {
	source := "a = 33"
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")

	source = "let a = 33bbb"
	assert.Panics(t, func() { Compile(source) }, "Code did nit panic")
}
