package interpreter

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

var luaMap = map[string]string{
	`Hello <?lua write("world");?>`:                                               "Hello world",
	`<html><?lua write("First");?><br><?lua write("Second")?><?lua write "Third"`: `<html>First<br>SecondThird`,
	`<?lua for i = 1, 3, 1 do ?>Hello<?lua end ?>`:                                "HelloHelloHello",
	"[[test]]": "[[test]]",
}

func TestInterpret(t *testing.T) {
	for script, expect := range luaMap {
		L := lua.NewState()
		ctx := NewContext(L, nil)
		in := []byte(script)

		if err := Interpret(L, in, &ctx.Out); err != nil {
			t.Errorf("Error on interpret: %s", err)
		}

		if actual := ctx.Out.String(); actual != expect {
			t.Errorf("Expected '%s', got '%s'", expect, actual)
		}
		L.Close()
	}
}
