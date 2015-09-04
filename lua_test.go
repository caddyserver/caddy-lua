package lua

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

var luaMap = map[string]string{
	`Hello <?lua print("world");?>`:                                               "Hello world",
	`<html><?lua print("First");?><br><?lua print("Second")?><?lua print "Third"`: `<html>First<br>SecondThird`,
}

func TestInterpret(t *testing.T) {
	for script, expect := range luaMap {
		L := lua.NewState()
		ctx := NewContext(L, nil)
		in := []byte(script)

		if err := Interpret(L, in, &ctx.out); err != nil {
			t.Errorf("Error on interpret: %s", err)
		}

		if ctx.out.String() != expect {
			t.Errorf("Expected '%s', got '%s'", expect, ctx.out.String())
		}
		L.Close()
	}
}

func TestImport(t *testing.T) {
}
