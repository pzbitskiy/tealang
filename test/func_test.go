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

func TestFuncSlotsAlloc1(t *testing.T) {
	source := `
let g = ""

function getch(digit) {
	return itob(digit + 48)
}

function mod10(nn) {
	g = getch(nn / 10)
	return nn % 10
}

function logic() {
	let a = mod10(74)
	assert(a == 4)
	let b = mod10(a+1)
	assert(b == 5)
	// assert(g != "")
	return 1
}
`
	performTest(t, source)
}

func TestFuncSlotsAlloc2(t *testing.T) {
	source := `
let g = ""

function sum(x, y) {
	return x + y
}

function mod10(nn) {
	let x = 10
	g = itob(sum(nn, nn / x))
	return nn % 10
}

function logic() {
	let a = mod10(74)
	assert(a == 4)
	assert(g != "")
	let x = 1
	let y = 2
	let c = sum(x, y)
	assert(c == 3)
	return 1
}
`
	performTest(t, source)
}

func TestFuncSlotsAlloc3(t *testing.T) {
	source := `
let g = ""

function sum(x, y) {
	return x + y
}

function mod10(nn) {
	let x = 10
	g = itob(sum(nn, nn / x))
	return nn % 10
}

// non-commutative op to ensure args order
function modsum(a, b) {
	let r = a % b
	let p = r + sum(a, b)
	return p
}

function logic() {
	let a = mod10(74)
	assert(a == 4)
	assert(g != "")
	let x = 1
	let y = 2
	let c = sum(x, y)
	assert(c == 3)
	let z = modsum(x, y)
	assert(z == 4)
	let yy = 5
	let zz = modsum(z, yy)
	assert(zz == 13)
	return 1
}
`
	performTest(t, source)
}
