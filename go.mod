module github.com/rickb777/gowebdav

go 1.24.1

toolchain go1.24.2

require (
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rickb777/expect v1.0.6
	github.com/rickb777/httpclient v0.35.1
	golang.org/x/net v0.47.0
)

require (
	github.com/go-xmlfmt/xmlfmt v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/rickb777/plural v1.4.7 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

//replace github.com/rickb777/httpclient => ../httpclient
