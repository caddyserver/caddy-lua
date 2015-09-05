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

		file, err := h.FileSys.Open(filepath.Join(h.Root, fpath))
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
			fmt.Println(err)
			return http.StatusInternalServerError, err
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

// Interpret reads a source, executes any Lua, and writes the results.
//
// This assumes that the reader has Lua embedded in `<?lua ... ?>` sections.
func Interpret(L *lua.LState, src []byte, out io.Writer) error {
	var luaIn bytes.Buffer

	inCode := false
	line := 1
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			line++
		}
		if inCode {
			if isEnd(i, src) {
				i++ // Skip two characters: ? and >
				if err := L.DoString(luaIn.String()); err != nil {
					// TODO: Need to make it easy to tell that this is a
					// parse error.
					// Issue opened upstream:
					// https://github.com/yuin/gopher-lua/issues/46
					// BUG: Line number is not always correct; we need it from the Lua engine.
					return fmt.Errorf("Lua Error (Line %d): %s", line, err)
				}
				luaIn.Reset()
				inCode = false
			} else {
				luaIn.WriteByte(src[i])
			}
		} else {
			if isStart(i, src) {
				i += 4
				inCode = true
			} else if _, err := out.Write([]byte{src[i]}); err != nil {
				return err
			}
		}
	}

	// Handle the case where a file ends inside of a <?lua block.
	// Mimic PHP's behavior.
	if inCode && luaIn.Len() > 0 {
		if err := L.DoString(luaIn.String()); err != nil {
			// TODO: Need to make it easy to tell that this is a
			// parse error.
			return fmt.Errorf("Lua Error (Line %d): %s", line, err)
		}
	}

	return nil
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
