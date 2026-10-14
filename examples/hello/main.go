// Command hello is Rackd with one small extension compiled in: a page, a nav
// entry, an API route, a permission and content on each device's page. It
// shows the extension API end to end, and the web UI's end-to-end tests run
// against it.
package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/martinsuchenak/rackd/pkg/app"
	"github.com/martinsuchenak/rackd/pkg/extension"
)

//go:embed ui
var uiFiles embed.FS

type hello struct{}

var readHello = &extension.Permission{Resource: "hello", Action: "read"}

func (hello) Manifest() extension.Manifest {
	assets, _ := fs.Sub(uiFiles, "ui")
	return extension.Manifest{
		Name:    "hello",
		Version: "1.0.0",
		Permissions: []extension.Permission{
			{Resource: "hello", Action: "read", DefaultRoles: []string{"admin"}},
		},
		UI: extension.UI{
			Pages:   []extension.Page{{Path: "/hello", Title: "Hello", Permission: readHello}},
			Nav:     []extension.NavItem{{Label: "Hello", Path: "/hello", Icon: "zap", Order: 97, Permission: readHello}},
			Assets:  assets,
			Scripts: []string{"hello.js"},
			Styles:  []string{"hello.css"},
		},
	}
}

func (hello) RegisterRoutes(r extension.Router, host extension.Host) {
	r.Handle("GET /hello", func(w http.ResponseWriter, req *http.Request) {
		name := "there"
		if c := extension.CallerFrom(req.Context()); c != nil && c.Username != "" {
			name = c.Username
		}
		extension.WriteJSON(w, http.StatusOK, map[string]string{"greeting": "Hello, " + name})
	}, extension.Require(readHello.Resource, readHello.Action))
}

func main() {
	app.Main(app.Options{Features: []extension.Feature{hello{}}})
}
