package lua

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yuin/gopher-lua"
)

// NewContext creates a new context for executing Lua scripts
// given a Lua state L and HTTP ResponseWriter rw.
func NewContext(L *lua.LState, rw http.ResponseWriter) *Context {
	global := &Context{rw: rw}

	// Global functions
	L.SetGlobal("print", L.NewFunction(global.print))
	L.SetGlobal("println", L.NewFunction(global.println))
	L.SetGlobal("import", L.NewFunction(global.limport))

	// Global types and their fields
	responseMt := L.NewTypeMetatable("response")
	L.SetField(responseMt, "status", L.NewFunction(global.responseStatus))
	L.SetGlobal("response", responseMt)

	return global
}

type Context struct {
	out       bytes.Buffer   // buffer that becomes the response body
	callbacks []func() error // executed after successful Lua script
	rw        http.ResponseWriter
}

// print writes to the out buffer (not stdout).
// Example: print("foo", "bar")
func (c *Context) print(L *lua.LState) int {
	top := L.GetTop()
	for i := 1; i <= top; i++ {
		c.out.WriteString(L.Get(i).String())
		if i != top {
			c.out.WriteString(" ")
		}
	}
	return 0
}

// println writes to the out buffer with a trailing newline.
// Example: println("foo", "bar")
func (c *Context) println(L *lua.LState) int {
	c.print(L)
	c.out.WriteString("\n")
	return 0
}

// limport imports Lua markup files.
func (c *Context) limport(L *lua.LState) int {
	path := L.Get(1).String()

	if strings.ToLower(filepath.Ext(path)) == ".lua" {
		fmt.Println("TODO: Call doimport instead.")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		// TODO: How do we handle this correctly?
		fmt.Printf("Failed to import '%s': %s\n", path, err)
		return 0
	}

	Interpret(L, data, &c.out)

	return 0
}

// Response contains values and methods related to the HTTP response.
type Response struct {
	http.ResponseWriter
}

// status sets the status code.
// Example: response.status(403)
func (c *Context) responseStatus(L *lua.LState) int {
	if L.GetTop() > 0 {
		top := L.Get(-1)
		if L.Get(-1).Type() == lua.LTNumber {
			code, err := strconv.Atoi(top.String())
			if err != nil {
				// TODO
				fmt.Printf("cannot convert %s to a number\n", top)
			} else {
				c.callbacks = append(c.callbacks, func() error {
					c.rw.WriteHeader(code)
					return nil
				})
				//c.rw.WriteHeader(code)
			}
		}
	}
	return 0
}
