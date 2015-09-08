package interpreter

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/yuin/gopher-lua"
)

// write writes to the out buffer (not stdout).
//
// write does not append a trailing newline.
// Example: write("foo", "bar")
func (c *Context) write(L *lua.LState) int {
	top := L.GetTop()
	for i := 1; i <= top; i++ {
		c.Out.WriteString(L.Get(i).String())
		if i != top {
			c.Out.WriteString(" ")
		}
	}
	return 0
}

// print writes to the out buffer with a trailing newline.
// Example: print("foo", "bar")
func (c *Context) print(L *lua.LState) int {
	c.write(L)
	c.Out.WriteString("\n")
	return 0
}

// include imports Lua markup files.
func (c *Context) include(L *lua.LState) int {
	path := L.Get(1).String()

	if strings.ToLower(filepath.Ext(path)) == ".lua" {
		fmt.Println("TODO: Call doimport instead.")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		// TODO: How do we handle this correctly?
		fmt.Printf("Failed to import '%s': %s\n", path, err)
		L.Error(L.CheckAny(1), 0)
		return 0
	}

	Interpret(L, data, &c.Out)

	return 0
}
