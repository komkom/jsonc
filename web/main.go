//go:generate gopherjs build pages/index.go
//go:generate mv index.js config/data/static
package main

import (
	"bytes"
	"io"
	"net/http"
	"path"

	"github.com/komkom/jsonc/reader"
	"github.com/komkom/jsonc/web/config"
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

		resp.Write(b)
	})

	http.HandleFunc(`/index.html`, func(resp http.ResponseWriter, req *http.Request) {

		resp.Write(config.C.IndexPage)
	})

	http.HandleFunc(`/fmt`, func(resp http.ResponseWriter, req *http.Request) {

		buf := bytes.Buffer{}
		buf.ReadFrom(req.Body)

		//fmt.Printf("_____ %s", buf.Bytes())

		jcr, err := reader.New(&buf, true)
		if err != nil {
			//fmt.Printf("____uu   %v", err.Error())
			error404(resp)
			return
		}

		io.Copy(resp, jcr)

		//resp.Write([]byte(`{"test":"test"}`))

	})

	http.ListenAndServe(`:1199`, nil)

}

func error404(resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusNotFound)
	resp.Write(config.C.ErrorPage)
}
