package lua

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/caddy-lua/interpreter"
	"github.com/mholt/caddy/middleware"
	"github.com/mholt/caddy/middleware/browse"
	"github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

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
		ctx := interpreter.NewContext(L, w)

		if err := interpreter.Interpret(L, input, &ctx.Out); err != nil {
			var errReport error

			ierr := err.(interpreter.InterpretationError)
			if lerr, ok := ierr.Err.(*lua.ApiError); ok {
				switch cause := lerr.Cause.(type) {
				case *parse.Error:
					errReport = fmt.Errorf("%s:%d (col %d): Syntax error near '%s'", fileName,
						cause.Pos.Line+ierr.LineOffset, cause.Pos.Column, cause.Token)
				case *lua.CompileError:
					errReport = fmt.Errorf("%s:%d: %s", fileName,
						cause.Line+ierr.LineOffset, cause.Message)
				default:
					errReport = fmt.Errorf("%s: %s", fileName, cause.Error())
				}
			}

			return http.StatusInternalServerError, errReport
		}

		for _, f := range ctx.Callbacks {
			err := f()
			if err != nil {
				// TODO
				fmt.Println(err)
			}
		}

		// Write the combined text to the http.ResponseWriter
		w.Write(ctx.Out.Bytes())

		return http.StatusOK, nil
	}

	return h.Next.ServeHTTP(w, r)
}
