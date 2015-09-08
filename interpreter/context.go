package interpreter

import (
	"bytes"
	"net/http"

	"github.com/yuin/gopher-lua"
)

type Context struct {
	Callbacks []func() error // executed after successful Lua script
	Out       bytes.Buffer   // buffer that becomes the response body
	rw        http.ResponseWriter
}

// NewContext creates a new context for executing Lua scripts
// given a Lua state L and HTTP ResponseWriter rw.
func NewContext(L *lua.LState, rw http.ResponseWriter) *Context {
	global := &Context{rw: rw}

	// Global functions
	L.SetGlobal("write", L.NewFunction(global.write))
	L.SetGlobal("print", L.NewFunction(global.print))
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
