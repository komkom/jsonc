//go:generate gopherjs build js/main.go
//go:generate mv main.js config/data/static
package main

import (
	"net/http"
	"path"
	"strings"

	"github.com/komkom/jsonc/web/config"
)

var (
	mimetypes = map[string]string{
		`css`: `text/css`,
		`svg`: `image/svg+xml`,
		`png`: `image/png`,
	}
)

func main() {

	// static files
	http.HandleFunc(`/static/`, func(resp http.ResponseWriter, req *http.Request) {

		_, f := path.Split(req.URL.Path)
		b, err := config.C.StaticFile(f)
		if err != nil {
			error404(resp)
			return
		}

		for k, v := range mimetypes {
			if strings.HasSuffix(req.URL.Path, k) {
				resp.Header().Add("Content-Type", v)
			}
		}

		resp.Write(b)
	})

	http.HandleFunc(`/index.html`, func(resp http.ResponseWriter, req *http.Request) {

		resp.Write(config.C.IndexPage)
	})

	http.ListenAndServe(`:1199`, nil)
}

func error404(resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusNotFound)
	resp.Write(config.C.ErrorPage)
}
