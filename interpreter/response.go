package interpreter

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/yuin/gopher-lua"
)

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
				c.Callbacks = append(c.Callbacks, func() error {
					c.rw.WriteHeader(code)
					return nil
				})
			}
		}
	}
	return 0
}
