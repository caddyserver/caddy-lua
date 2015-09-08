package interpreter

import (
	"log"

	"github.com/yuin/gopher-lua"
)

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
