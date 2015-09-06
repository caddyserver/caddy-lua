package lua

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/caddy/config/setup"
	"github.com/mholt/caddy/middleware"
	"github.com/mholt/caddy/middleware/browse"
	"github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

func Setup(c *setup.Controller) (middleware.Middleware, error) {
	root := c.Root

	rules, err := parseLuaCaddyfile(c)
	if err != nil {
		return nil, err
	}

	return func(next middleware.Handler) middleware.Handler {
		return &Handler{
			Next:    next,
			Rules:   rules,
			Root:    root,
			FileSys: http.Dir(root),
		}
	}, nil
}

type Handler struct {
	Next    middleware.Handler
	Rules   []Rule
	Root    string // site root
	FileSys http.FileSystem
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	for _, rule := range h.Rules {
		if !middleware.Path(r.URL.Path).Matches(rule.BasePath) {
			continue
		}

		// Check for index file
		fpath := r.URL.Path
		if idx, ok := middleware.IndexFile(h.FileSys, fpath, browse.IndexPages); ok {
			fpath = idx
		}

		// TODO: Check extension. If .lua, assume whole file is Lua script.
		fileName := filepath.Join(h.Root, fpath)
		file, err := h.FileSys.Open(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				return http.StatusNotFound, nil
			} else if os.IsPermission(err) {
				return http.StatusForbidden, nil
			}
			return http.StatusInternalServerError, err
		}
		defer file.Close()

		input, err := ioutil.ReadAll(file)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		L := lua.NewState()
		defer L.Close()
		ctx := NewContext(L, w)

		if err := Interpret(L, input, &ctx.out); err != nil {
			var errReport error

			ierr := err.(interpretationError)
			if lerr, ok := ierr.err.(*lua.ApiError); ok {
				switch cause := lerr.Cause.(type) {
				case *parse.Error:
					errReport = fmt.Errorf("%s:%d (col %d): Syntax error near '%s'", fileName,
						cause.Pos.Line+ierr.lineOffset, cause.Pos.Column, cause.Token)
				case *lua.CompileError:
					errReport = fmt.Errorf("%s:%d: %s", fileName,
						cause.Line+ierr.lineOffset, cause.Message)
				default:
					errReport = fmt.Errorf("%s: %s", fileName, cause.Error())
				}
			}

			return http.StatusInternalServerError, errReport
		}

		for _, f := range ctx.callbacks {
			err := f()
			if err != nil {
				// TODO
				fmt.Println(err)
			}
		}

		// Write the combined text to the http.ResponseWriter
		w.Write(ctx.out.Bytes())

		return http.StatusOK, nil
	}

	return h.Next.ServeHTTP(w, r)
}

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
	closel := `]], "&lualb;", "]");print(__buf__); __buf__ = nil`
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

func parseLuaCaddyfile(c *setup.Controller) ([]Rule, error) {
	var rules []Rule

	for c.Next() {
		r := Rule{BasePath: "/"}
		if c.NextArg() {
			r.BasePath = c.Val()
		}
		if c.NextArg() {
			return rules, c.ArgErr()
		}
		rules = append(rules, r)
	}

	return rules, nil
}

type Rule struct {
	BasePath string // base request path to match
}
