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
	return 1;
}
`
	performTest(t, source)
}
