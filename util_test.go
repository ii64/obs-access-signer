package main

import "testing"

func TestUnsafeByteSliceToString(t *testing.T) {
	exp := "foo bar"
	act := unsafeByteSliceToString([]byte(exp))
	if exp != act { // cmp str
		t.Fail()
	}
	println(&exp, &act)
	println(exp, act)
}
