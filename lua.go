package lua

import (
	"bytes"
	"fmt"
	"io"

	"github.com/yuin/gopher-lua"
)

type interpretationError struct {
	err        error
	lineOffset int
}

func (e interpretationError) Error() string {
	return e.err.Error()
}

// Interpret reads a source, executes any Lua, and writes the results.
//
// This assumes that the reader has Lua embedded in `<?lua ... ?>` sections.
func Interpret(L *lua.LState, src []byte, out io.Writer) error {
	// NOTE: There are several ways we could walk a lim source file and
	// generate code. We opted for the route of converting non-Lua text
	// into Lua code. Basically, code outside of <?lua ... ?> sections is
	// converted into calls to print(). This provides the advantage of
	// being able to embed loops and conditionals into lim files, but does
	// not incur the overhead of working directly with a Lua parse tree.
	//
	// Lua's heredoc syntax uses [[ and ]] as delimiters, so we escape
	// ] characters when creating the strings, and then replace them
	// latter.
	var luaIn, pbuf bytes.Buffer

	// We have to encode ] into &lualb; to prevent heredocs from breaking
	// when we inline text.
	lent := "&lualb;"

	inCode := false
	//line, luaStartLine := 0, 0
	line := 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			line++
		}
		if inCode {
			if isEnd(i, src) {
				i++ // Skip two characters: ? and >
				inCode = false
			} else {
				luaIn.WriteByte(src[i])
			}
		} else {
			if isStart(i, src) {
				i += 4
				inCode = true
				//luaStartLine = line
				if pbuf.Len() > 0 {
					inlineText(&pbuf, &luaIn)
				}
				pbuf.Reset()
			} else if src[i] == ']' {
				pbuf.WriteString(lent)
			} else {
				pbuf.WriteByte(src[i])
			}
		}
	}

	if !inCode && pbuf.Len() > 0 {
		inlineText(&pbuf, &luaIn)
	}

	// FIXME: MPB: Test line count. The generated Lua should have the same
	// line numbers as the source file.
	if err := executeLua(L, &luaIn); err != nil {
		return interpretationError{err: err, lineOffset: 0}
	}

	return nil
}

// inlineText takes some non-Lua text and inlines it into Lua code.
func inlineText(b, luaIn *bytes.Buffer) {
	openl := `__buf__ = string.gsub([[`
	closel := `]], "&lualb;", "]");write(__buf__); __buf__ = nil`
	luaIn.WriteString(openl)
	luaIn.Write(b.Bytes())
	luaIn.WriteString(closel)
}

func executeLua(L *lua.LState, input io.Reader) error {
	fn, err := L.Load(input, "<Caddy-Lua>")
	if err != nil {
		return err
	}

	L.Push(fn)
	return L.PCall(0, lua.MultRet, L.NewFunction(func(L *lua.LState) int {
		// TODO: Clean up error handling here
		obj := L.Get(1)
		dbg, _ := L.GetStack(1)
		L.GetInfo("Slunf", dbg, lua.LNil)
		fmt.Println("Runtime error:", obj.String())
		fmt.Println("Line:", dbg.CurrentLine)
		return 1 // returns the original object
	}))
}

var startSeq = []byte{'<', '?', 'l', 'u', 'a'}

func isStart(start int, slice []byte) bool {
	if start+5 >= len(slice) {
		return false
	}
	for i := 0; i < 5; i++ {
		if startSeq[i] != slice[start+i] {
			return false
		}
	}
	return true
}

func isEnd(start int, slice []byte) bool {
	if start+1 >= len(slice) {
		return false
	}
	if slice[start] == '?' && slice[start+1] == '>' {
		return true
	}
	return false
}
