package gowebdav

import (
	"fmt"
	"os"
	"time"
)

var _ os.FileInfo = fileinfo{}

// fileinfo is our structure for a given fileinfo
type fileinfo struct {
	path        string
	name        string
	contentType string
	size        int64
	modified    time.Time
	etag        string
	isdir       bool
}

// Path returns the full path of a file
func (f fileinfo) Path() string {
	return f.path
}

// Name returns the name of a file
func (f fileinfo) Name() string {
	return f.name
}

// ContentType returns the content type of a file
func (f fileinfo) ContentType() string {
	return f.contentType
}

// Size returns the size of a file
func (f fileinfo) Size() int64 {
	return f.size
}

// Mode will return the mode of a given file
func (f fileinfo) Mode() os.FileMode {
	// TODO check webdav perms
	if f.isdir {
		return 0775 | os.ModeDir
	}

	return 0664
}

// ModTime returns the modified time of a file
func (f fileinfo) ModTime() time.Time {
	return f.modified
}

// ETag returns the ETag of a file
func (f fileinfo) ETag() string {
	return f.etag
}

// IsDir let us see if a given file is a directory or not
func (f fileinfo) IsDir() bool {
	return f.isdir
}

// Sys ????
func (f fileinfo) Sys() interface{} {
	return nil
}

// String lets us see file information
func (f fileinfo) String() string {
	if f.isdir {
		return fmt.Sprintf("Dir : '%s' - '%s'", f.path, f.name)
	}

	return fmt.Sprintf("File: '%s' SIZE: %d MODIFIED: %s ETAG: %s CTYPE: %s", f.path, f.size, f.modified.String(), f.etag, f.contentType)
}
