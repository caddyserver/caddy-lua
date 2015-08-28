caddy-lua
=========

An experimental way of serving dynamic sites like PHP, but without PHP. Embed Lua scripts into your HTML files and this Caddy add-on can execute them in-process, no extra setup required.

For example (steps 1 and 2 only needed if you haven't run them before):

1. `$ go get github.com/caddyserver/caddy-lua`

2. `$ go get github.com/caddyserver/caddydev`

3. `$ cd $GOPATH/src/github.com/caddyserver/caddy-lua`

4. `$ caddydev lua`

5. Open `localhost:2015` in your browser and you should see text printed from a Lua script embedded in index.html.

If all goes well, we intend to build a large standard library of functions like PHP has, but better (more organized, consistent and modern). This would allow you to write dynamic sites in Lua that are even easier and safer to deploy than PHP sites.

For now, tweet to @mholt6 with questions or open an issue.
