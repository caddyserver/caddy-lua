package lua

import (
	"bytes"
	"testing"
)

var luaMap = map[string]string{
	`Hello <?lua print("world");?>`:                                               "Hello world",
	`<html><?lua print("First");?><br><?lua print("Second")?><?lua print "Third"`: `<html>First<br>SecondThird`,
}

func TestInterpret(t *testing.T) {
	for script, expect := range luaMap {
		var out bytes.Buffer
		in := []byte(script)

		if err := Interpret(&out, in); err != nil {
			t.Errorf("Error on interpret: %s", err)
		}

		if out.String() != expect {
			t.Errorf("Expected '%s', got '%s'", expect, out.String())
		}
	}
}
