package test

import (
	"testing"
)

func TestByteArith(t *testing.T) {
	source := `
function logic() {
	let z = bzero(4)
	assert(len(z) == 4)
	let r = band(z, "\x11");
	assert(r == "\x00\x00\x00\x00")
	assert(badd("\x01\x02\x03\x04", z) == "\x01\x02\x03\x04")
	assert(bsqrt("\x11") == "\x04")
	return 1;
}
`
	performTest(t, source)
}

func TestExtract(t *testing.T) {
	source := `
function logic() {
	let a = extract("\x12\x34\x56\x78\x9a\xbc", 1, 2)
	assert(a == "\x34\x56")

	let s = 5
	let e = 1
	a = extract("\x12\x34\x56\x78\x9a\xbc", s, e)
	assert(a == "\xbc")

	let b = extract(UINT16, "\x12\x34\x56\x78\x9a\xbc", 1)
	assert(b == 0x3456)

	b = extract(UINT32, "\x12\x34\x56\x78\x9a\xbc", 1)
	assert(b == 0x3456789a)

	return 1;
}
`
	performTest(t, source)
}

func TestSubstring(t *testing.T) {
	source := `
function logic() {
	let a = substring("\x12\x34\x56\x78\x9a\xbc", 1, 3)
	assert(a == "\x34\x56")

	let s = 1
	let e = 3
	a = substring("\x12\x34\x56\x78\x9a\xbc", s, e)
	assert(a == "\x34\x56")

	return 1
}
`
	performTest(t, source)
}

func TestDivw(t *testing.T) {
	source := `
function logic() {
	let a = divw(1, 0, 2)
	assert(a == 0x8000000000000000)

	a = divw(0, 90, 30)
	assert(a == 3)
	return 1
}
`
	performTest(t, source)
}
