package test

import (
	"testing"
)

func TestFunc(t *testing.T) {
	source := `
function sum(x, y) { return x + y; }
function logic() {
	let a = 1
	let b = sum (a, 2)
	let x = 4
	let c = sum (x, 1)
	assert(b + c == 8)
	return 1
}
`
	performTest(t, source)
}

func TestFuncInline(t *testing.T) {
	source := `
inline function sum(x, y) { return x + y; }
function logic() {
	let a = 1
	let b = sum (a, 2)
	let x = 4
	let c = sum (x, 1)
	assert(b + c == 8)
	return 1
}
`
	performTest(t, source)
}
