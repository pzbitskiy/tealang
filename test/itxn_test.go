package test

import (
	"testing"
)

func TestInnerTxn(t *testing.T) {
	source := `
function logic() {
  itxn.begin()
  itxn.TypeEnum = 1
  itxn.Receiver = txn.Sender
  itxn.submit()
  return 1
}`
  teal := `#pragma version 5
intcblock 0 1
fun_main:
itxn_begin
intc 1
itxn_field TypeEnum
txn Sender
itxn_field Receiver
itxn_submit
intc 1
return
end_main:`
	compileTest(t, source, teal)
}
