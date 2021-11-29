package test

import (
	"testing"
)

func TestLog(t *testing.T) {
	source := `
function logic() {
	log("Hi")
	return 1
}
`
  teal := `#pragma version 5
intcblock 0 1
bytecblock 0x4869
fun_main:
bytec 0
log
intc 1
return
end_main:
`
	compileTest(t, source, teal)
}
