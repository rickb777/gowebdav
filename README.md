# GoWebDAV

[![GoDoc](https://godoc.org/github.com/rickb777/gowebdav?status.svg)](https://godoc.org/github.com/rickb777/gowebdav)
[![Go Report Card](https://goreportcard.com/badge/github.com/rickb777/gowebdav)](https://goreportcard.com/report/github.com/rickb777/gowebdav)
[![Issues](https://img.shields.io/github/issues/rickb777/gowebdav.svg)](https://github.com/rickb777/gowebdav/issues)

A golang WebDAV client library with a command line tool included.

## Main features

This API allows the following actions on remote WebDAV servers:
* [create folders](#create-folders-on-a-webdav-server)
* [list files](#list-files)
* [download file](#download-file-to-byte-array)
* [upload file](#upload-file-from-byte-array)
* [get information about specified file/folder](#get-information-about-specified-filefolder)
* [move file to another location](#move-file-to-another-location)
* [copy file to another location](#copy-file-to-another-location)
* [delete file](#delete-file)

## Get started

`go get github.com/rickb777/gowebdav`

## Usage

Start by creating a `Client` instance using the `NewClient()` function:

```go
root := "https://webdav.mydomain.me"
user := "user"
password := "password"

c := gowebdav.NewClient(root)
```

Then use this `Client` to perform actions described below.

**NOTICE:** we will not check errors in examples, to focus you on the `gowebdav` library's code, but you should do it in your code.

### Create folders on a WebDAV server
```go
err := c.Mkdir("folder", 0644)
```
If you want to create several nested folders, you can use `c.MkdirAll()`:
```go
err := c.MkdirAll("folder/subfolder/subfolder2", 0644)
```

### List files
```go
files, _ := c.ReadDir("folder/subfolder")
for _, file := range files {
    //notice that [file] has os.FileInfo type
    fmt.Println(file.Name())
}
```

### Download file to byte array
```go
webdavFilePath := "folder/subfolder/file.txt"
localFilePath := "/tmp/webdav/file.txt"

bytes, _ := c.ReadFile(webdavFilePath)
ioutil.WriteFile(localFilePath, bytes, 0644)
```

### Download file via reader
Alternatively, use the `c.ReadStream()` method:
```go
webdavFilePath := "folder/subfolder/file.txt"
localFilePath := "/tmp/webdav/file.txt"

reader, _ := c.ReadStream(webdavFilePath)

file, _ := os.Create(localFilePath)
defer file.Close()

io.Copy(file, reader)
```

### Upload file from byte array
```go
webdavFilePath := "folder/subfolder/file.txt"
localFilePath := "/tmp/webdav/file.txt"

bytes, _ := ioutil.ReadFile(localFilePath)

c.WriteFile(webdavFilePath, bytes, 0644)
```

### Upload file via writer
```go
webdavFilePath := "folder/subfolder/file.txt"
localFilePath := "/tmp/webdav/file.txt"

file, _ := os.Open(localFilePath)
defer file.Close()

c.WriteStream(webdavFilePath, file, 0644)
```

### Get information about specified file/folder
```go
webdavFilePath := "folder/subfolder/file.txt"

info := c.Stat(webdavFilePath)
//notice that [info] has os.FileInfo type
fmt.Println(info)
```

### Move file to another location
```go
oldPath := "folder/subfolder/file.txt"
newPath := "folder/subfolder/moved.txt"
isOverwrite := true

c.Rename(oldPath, newPath)
```

or use `RenameWithoutOverwriting(oldpath, newpath string)`.

### Copy file to another location
```go
oldPath := "folder/subfolder/file.txt"
newPath := "folder/subfolder/file-copy.txt"
isOverwrite := true

c.Copy(oldPath, newPath)
```

or use `CopyWithoutOverwriting(oldpath, newpath string) error`.

### Delete file
```go
webdavFilePath := "folder/subfolder/file.txt"

c.Remove(webdavFilePath)
```

## Links

You can read more details about WebDAV from the following resources:

* [RFC 4918 - HTTP Extensions for Web Distributed Authoring and Versioning (WebDAV)](https://tools.ietf.org/html/rfc4918)
* [RFC 5689 - Extended MKCOL for Web Distributed Authoring and Versioning (WebDAV)](https://tools.ietf.org/html/rfc5689)
* [RFC 7231 - HTTP/1.1 Status Code Definitions](https://tools.ietf.org/html/rfc7231#section-6 "HTTP/1.1 Status Code Definitions")
* [WebDav: Next Generation Collaborative Web Authoring By Lisa Dusseaul](https://books.google.de/books?isbn=0130652083 "WebDav: Next Generation Collaborative Web Authoring By Lisa Dusseault")

## Contributing

All contributing are welcome. If you have any suggestions or find some bug, please create an Issue to let us make this project better. We appreciate your help!

## License & Acknowledgement

This library is distributed under the BSD 3-Clause license found in the [LICENSE](https://github.com/rickb777/gowebdav/blob/master/LICENSE) file.

My thanks to 

 * Studio-b12: did much of the original work - see [gowebdav](https://github.com/studio-b12/gowebdav).
 * Andrew Koltyakov: wrote [Gosip](https://github.com/koltyakov/gosip), on which the SAML authentication used here is based.  
