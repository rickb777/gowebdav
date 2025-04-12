module github.com/rickb777/gowebdav

go 1.24.1

toolchain go1.24.2

require (
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rickb777/expect v0.19.2
	github.com/rickb777/httpclient v0.0.7
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/rickb777/plural v1.4.2 // indirect
)

//replace github.com/rickb777/httpclient => ../httpclient
