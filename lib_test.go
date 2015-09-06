package lua

import (
	"bytes"
	"log"
	"testing"

	"github.com/yuin/gopher-lua"
)

// runLuaTest is a utility for running a piece of code and getting a result.
//
// It bootstraps a pristine Lua environment each time, and loads the standard
// context.
//
// If the lim parameter is set to true, this will first send the code
// thorugh the Interpret function.
func runLuaTest(code string, out *bytes.Buffer, lim bool) error {
	L := lua.NewState()
	defer L.Close()
	ctx := NewContext(L, nil)
	ctx.out = *out
	if lim {
		if err := Interpret(L, []byte(code), &ctx.out); err != nil {
			return err
		}
		return nil
	} else if err := L.DoString(code); err != nil {
		return err
	}

	for _, f := range ctx.callbacks {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func TestLogger(t *testing.T) {
	tests := map[string]string{
		`log.info("test")`:                          "[info] test\n",
		`log.warn("test")`:                          "[warning] test\n",
		`log.debug("test")`:                         "[debug] test\n",
		`log.error("test %s %d", "a", 1)`:           "[error] test a 1\n",
		`log.warn("%%")`:                            "[warning] %%\n",
		`log.debug("%s %s", "one", "two", "three")`: "[debug] one two%!(EXTRA lua.LString=three)\n",
	}

	// Send log to buffer.
	var out bytes.Buffer
	log.SetOutput(&out)
	// Turn off date/time.
	log.SetFlags(0)

	for eval, expect := range tests {
		if err := runLuaTest(eval, &out, false); err != nil {
			t.Errorf("Failed eval of '%s': %s", eval, err)
		} else if out.String() != expect {
			t.Errorf("Expected '%s', got '%s'", expect, out.String())
		}
		out.Reset()
	}
}
