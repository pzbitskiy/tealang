package test

import (
	"testing"
)

func TestToInt(t *testing.T) {
	source := `
function logic() {
  let a = accounts[0].get("key")
  let b = toint(a) + 1
  return 1
}`
  teal := `#pragma version 5
intcblock 0 1
bytecblock 0x6b6579
fun_main:
intc 0
bytec 0
app_local_get
store 101
load 101
intc 1
+
store 102
intc 1
return
end_main:
`
	compileTest(t, source, teal)
}



