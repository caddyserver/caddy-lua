package lua

import (
	"net/http"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
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
