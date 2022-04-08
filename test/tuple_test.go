package test

import (
	"testing"
)

func TestAddw(t *testing.T) {
	source := `
function logic() {
	let carry, sum = addw(10, 20)
	assert(sum == 30)
	assert(carry == 0)
	return 1
}`
	performTest(t, source)
}

func TestMulw(t *testing.T) {
	source := `
function logic() {
	let high, low = mulw(10, 20)
	assert(low == 200)
	assert(high == 0)
	return 1
}`
	performTest(t, source)
}

func TestExpw(t *testing.T) {
	source := `
function logic() {
	let high, low = expw(2, 3)
	assert(low == 8)
	assert(high == 0)
	return 1
}`
	performTest(t, source)
}

func TestDivmodw(t *testing.T) {
	source := `
function logic() {
	let qhigh, qlow, rhigh, rlow = divmodw(2, 0, 0, 1)
	assert(qhigh == 2)
	assert(qlow == 0)
	assert(rhigh == 0)
	assert(rlow == 0)

return 1
}`
	performTest(t, source)

	source = `
function logic() {
	let qhigh, qlow, rhigh, rlow = divmodw(0, 99, 0, 2)
	assert(qhigh == 0)
	assert(qlow == 49)
	assert(rhigh == 0)
	assert(rlow == 1)

	qhigh, qlow, rhigh, rlow = divmodw(2, 0, 0, 2)
	assert(qhigh == 1)
	assert(qlow == 0)
	assert(rhigh == 0)
	assert(rlow == 0)

return 1
}`
	performTest(t, source)

}
