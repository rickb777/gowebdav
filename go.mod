module github.com/rickb777/gowebdav

go 1.24.1

toolchain go1.24.2

require (
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rickb777/expect v0.24.0
	github.com/rickb777/httpclient v0.34.0
	golang.org/x/net v0.39.0
)

require (
	github.com/go-xmlfmt/xmlfmt v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/rickb777/plural v1.4.4 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

//replace github.com/rickb777/httpclient => ../httpclient
