//go:generate go-bindata -pkg config data/...
package config

type Config struct {
	IndexPage []byte
	ErrorPage []byte
}

func (c Config) StaticFile(name string) ([]byte, error) {

	return Asset(`data/static/` + name)
}

var (
	C Config
)

func init() {

	b, err := Asset(`data/index.html`)
	if err != nil {
		panic(err)
	}

	C.IndexPage = b

	b, err = Asset(`data/error.html`)
	if err != nil {
		panic(err)
	}

	C.ErrorPage = b
}
