package interpreter

import (
	"bytes"
	"log"
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
