package lua

import (
	"bytes"
	"errors"
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

	rules, err := parse(c)
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

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		L := lua.NewState()
		defer L.Close()

		// buf is filled with the output (print) of the Lua script
		var buf bytes.Buffer

		L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
			top := L.GetTop()
			for i := 1; i <= top; i++ {
				buf.WriteString(L.Get(i).String())
				if i != top {
					buf.WriteString(" ")
				}
			}
			buf.WriteString("\n")
			return 0
		}))

		// Extract all Lua snippets and execute them
		// TODO: We're starting with just the first one... put this into a loop of some sort
		startToken, endToken := []byte("<?lua"), []byte("?>")
		if startPos := bytes.Index(contents, startToken); startPos != -1 {
			start := startPos + len(startToken)
			end := bytes.Index(contents[start:], endToken) + start // BUG: For now, I'm assuming "?>" doesn't appear in the Lua code anywhere
			if end <= start {
				return http.StatusInternalServerError, errors.New("Un-closed Lua block") // TODO: Better message
			}
			luaSnippet := string(contents[start:end])

			if err := L.DoString(luaSnippet); err != nil {
				return http.StatusInternalServerError, err
			}

			// Replace the Lua block with its output
			contents = append(contents[:startPos], append(buf.Bytes(), contents[end+len(endToken):]...)...)
		}

		// Write the combined text to the http.ResponseWriter
		w.Write(contents)

		return http.StatusOK, nil
	}

	return h.Next.ServeHTTP(w, r)
}

func parse(c *setup.Controller) ([]Rule, error) {
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
