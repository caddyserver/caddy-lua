package lua

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
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
	L.SetGlobal("include", L.NewFunction(global.include))

	// Logging facility
	lfuncs := map[string]lua.LGFunction{
		"info":  logInfo,
		"warn":  logWarn,
		"error": logErr,
		"debug": logDebug,
	}
	logger := L.RegisterModule("log", lfuncs)
	L.SetGlobal("log", logger)

	// Global types and their fields
	responseMt := L.NewTypeMetatable("response")
	L.SetField(responseMt, "status", L.NewFunction(global.responseStatus))
	L.SetGlobal("response", responseMt)

	return global
}

func logInfo(L *lua.LState) int  { return logMsg("[info] ", L) }
func logDebug(L *lua.LState) int { return logMsg("[debug] ", L) }
func logWarn(L *lua.LState) int  { return logMsg("[warning] ", L) }
func logErr(L *lua.LState) int   { return logMsg("[error] ", L) }
func logMsg(level string, L *lua.LState) int {
	old := log.Prefix()
	log.SetPrefix(level)
	tpl := L.CheckString(1)
	top := L.GetTop()

	// Optimization for the case where no formatter needs to be
	// applied.
	if top <= 1 {
		log.Print(tpl)
		log.SetPrefix(old)
		return 0
	}

	args := make([]interface{}, top-1)
	for i := 2; i <= top; i++ {
		args[i-2] = L.Get(i)
	}

	// FIXME: If more args are supplied than placeholders, this will
	// generate an error.
	log.Printf(tpl, args...)
	log.SetPrefix(old)
	return 0
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
