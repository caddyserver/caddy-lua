package interpreter

import (
	"io/ioutil"
	"testing"

	"github.com/yuin/gopher-lua"
)

func TestInclude(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	ctx := NewContext(L, nil)

	in, err := ioutil.ReadFile("./testdata/outer.lim")
	if err != nil {
		t.Errorf("failed to load outer.lim")
		return
	}

	if err := Interpret(L, in, &ctx.Out); err != nil {
		t.Errorf("Failed to interpret: %s", err)
	}
	t.Logf("output: %s", ctx.Out.String())
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
	if err := Interpret(L, in, &ctx.Out); err == nil {
		t.Error("Expected include_error.lim to produce an error")
	}
	t.Logf("output: %s", ctx.Out.String())
}
