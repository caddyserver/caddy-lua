package lua

import (
	"io/ioutil"
	"testing"

	"github.com/yuin/gopher-lua"
)

var luaMap = map[string]string{
	`Hello <?lua print("world");?>`:                                               "Hello world",
	`<html><?lua print("First");?><br><?lua print("Second")?><?lua print "Third"`: `<html>First<br>SecondThird`,
	`<?lua for i = 1, 3, 1 do ?>Hello<?lua end ?>`:                                "HelloHelloHello",
	"[[test]]": "[[test]]",
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

func TestInclude(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	ctx := NewContext(L, nil)

	in, err := ioutil.ReadFile("./testdata/outer.lim")
	if err != nil {
		t.Errorf("failed to load outer.lim")
		return
	}

	if err := Interpret(L, in, &ctx.out); err != nil {
		t.Errorf("Failed to interpret: %s", err)
	}
	t.Logf("output: %s", ctx.out.String())
}

func TestIncludeError(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	ctx := NewContext(L, nil)

	in, err := ioutil.ReadFile("./testdata/include_error.lim")
	if err != nil {
		t.Errorf("failed to load outer.lim")
		return
	}

	t.Log("Expecting an error whose message we can't surpress.")
	if err := Interpret(L, in, &ctx.out); err == nil {
		t.Error("Expected include_error.lim to produce an error")
	}
	t.Logf("output: %s", ctx.out.String())
}
