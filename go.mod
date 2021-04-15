module github.com/rickb777/gowebdav

go 1.16

require (
	github.com/onsi/gomega v1.11.0
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/rickb777/httpclient v0.0.4
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
)

replace github.com/rickb777/httpclient => ../httpclient
