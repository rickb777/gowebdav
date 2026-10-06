module github.com/rickb777/gowebdav

go 1.26.0

require (
	github.com/magefile/mage v1.17.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rickb777/expect v1.3.4
	github.com/rickb777/httpclient v0.56.0
	github.com/rickb777/netrc v1.0.3
	golang.org/x/net v0.59.0
)

require (
	github.com/go-xmlfmt/xmlfmt v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/rickb777/acceptable v1.0.0 // indirect
	github.com/rickb777/plural/v2 v2.1.1 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

//replace github.com/rickb777/httpclient => ../httpclient

tool github.com/magefile/mage
