package gowebdav

import (
	"bytes"
	"encoding/xml"
	"github.com/rickb777/gowebdav/auth"
	"io"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"sync"
	"time"
)

// responseStatusOK is the space-separated response code when OK
// (https://tools.ietf.org/html/rfc7230#section-3.1.2)
const responseStatusOK = " 200 "

const (
	MethodMove     = "MOVE"
	MethodCopy     = "COPY"
	MethodMkcol    = "MKCOL"
	MethodPropfind = "PROPFIND"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is compatible with Afero.Fs.
// https://pkg.go.dev/github.com/spf13/afero#Fs
type Client interface {
	// Ping tests the connection to the webdav server.
	Ping() error

	//----- Webdav methods -----

	// ReadDir reads the contents of a remote directory
	ReadDir(path string) ([]os.FileInfo, error)

	// Copy copies a file from oldpath to newpath.
	// If newpath already exists and is not a directory, Copy overwrites it.
	Copy(oldpath, newpath string) error

	// CopyWithoutOverwriting copies a file from oldpath to newpath.
	CopyWithoutOverwriting(oldpath, newpath string) error

	// ReadFile reads the contents of a remote file.
	ReadFile(path string) ([]byte, error)

	// ReadStream reads the stream for a given path. The caller must
	// close the returned io.ReadCloser.
	ReadStream(path string) (io.ReadCloser, error)

	// WriteFile writes data to a given path on the webdav server.
	WriteFile(path string, data []byte, contentType string) error

	// WriteStream writes from a stream to a resource on the webdav server.
	WriteStream(path string, stream io.Reader, contentType string) error

	//----- Afero.Fs methods below (incomplete) -----

	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	// Create(name string) (File, error)

	// Mkdir makes a directory (also known as a collection in Webdav)
	Mkdir(path string, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that do not exist yet.
	MkdirAll(path string, perm os.FileMode) error

	// Open opens a file for reading.
	// Open(name string) (File, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
	// is passed, it is created with mode perm (before umask). If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *PathError.
	// OpenFile(name string, flag int, perm os.FileMode) (File, error)

	// Remove removes a remote file
	Remove(path string) error

	// RemoveAll removes remote files
	RemoveAll(path string) error

	// Rename renames (moves) oldpath to newpath.
	// If newpath already exists and is not a directory, Rename replaces it.
	Rename(oldname, newname string) error

	// RenameWithoutOverwriting renames (moves) oldpath to newpath.
	// If newpath already exists, a *os.PathError error is returned
	// containing the message "file already exists".
	RenameWithoutOverwriting(oldpath, newpath string) error

	// Stat returns a FileInfo describing the named file, or an error, if any happens.
	Stat(path string) (os.FileInfo, error)

	// The name of this FileSystem.
	Name() string

	// Chmod changes the mode of the named file to mode.
	//Chmod(name string, mode os.FileMode) error

	// Chown changes the uid and gid of the named file.
	//Chown(name string, uid, gid int) error

	//Chtimes changes the access and modification times of the named file
	//Chtimes(name string, atime time.Time, mtime time.Time) error
}

// client defines our structure
type client struct {
	root    string
	headers http.Header
	hc      HttpClient

	authMutex sync.Mutex
	auth      auth.Authenticator
}

//-------------------------------------------------------------------------------------------------

// NewClient creates a new Client. By default, this uses the default HTTP client.
func NewClient(uri string, opts ...ClientOpt) Client {
	cl := &client{
		root:    withoutTrailingSlash(uri),
		headers: make(http.Header),
		hc:      http.DefaultClient,
		auth:    auth.Anonymous,
	}
	for _, opt := range opts {
		opt(cl)
	}
	return cl
}

//-------------------------------------------------------------------------------------------------

type ClientOpt func(Client)

// AddHeader lets us set arbitrary headers for a given client
func AddHeader(key, value string) ClientOpt {
	return func(c Client) {
		c.(*client).headers.Add(key, value)
	}
}

// SetAuthentication sets the authentication credentials and method.
// Leave the authenticator method blank to allow HTTP challenges to
// select an appropriate method. Otherwise it should be "basic".
func SetAuthentication(authenticator auth.Authenticator) ClientOpt {
	return func(c Client) {
		c.(*client).auth = authenticator
	}
}

// SetHttpClient changes the http.Client. This allows control over
// the http.Transport, timeouts etc.
func SetHttpClient(httpClient HttpClient) ClientOpt {
	return func(c Client) {
		c.(*client).hc = httpClient
	}
}

//-------------------------------------------------------------------------------------------------

func (c *client) Name() string {
	return "webdav:" + c.root
}

func (c *client) Ping() error {
	rs, err := c.options("/")
	if err != nil {
		return err
	}

	err = rs.Body.Close()
	if err != nil {
		return err
	}

	if rs.StatusCode != http.StatusOK {
		return newPathError("Connect", c.root, rs.StatusCode)
	}

	return nil
}

type props struct {
	Status      string   `xml:"DAV: status"`
	Name        string   `xml:"DAV: prop>displayname,omitempty"`
	Type        xml.Name `xml:"DAV: prop>resourcetype>collection,omitempty"`
	Size        string   `xml:"DAV: prop>getcontentlength,omitempty"`
	ContentType string   `xml:"DAV: prop>getcontenttype,omitempty"`
	ETag        string   `xml:"DAV: prop>getetag,omitempty"`
	Modified    string   `xml:"DAV: prop>getlastmodified,omitempty"`
}

type response struct {
	Href  string  `xml:"DAV: href"`
	Props []props `xml:"DAV: propstat"`
}

func getProps(r *response, status string) *props {
	for _, prop := range r.Props {
		if strings.Contains(prop.Status, status) {
			return &prop
		}
	}
	return nil
}

// ReadDir reads the contents of a remote directory
func (c *client) ReadDir(path string) ([]os.FileInfo, error) {
	path = withSurroundingSlashes(path)
	files := make([]os.FileInfo, 0)
	skipSelf := true
	parse := func(resp interface{}) error {
		r := resp.(*response)

		if skipSelf {
			skipSelf = false
			if p := getProps(r, responseStatusOK); p != nil && p.Type.Local == "collection" {
				r.Props = nil
				return nil
			}
			return newPathError("ReadDir", path, 405)
		}

		if p := getProps(r, responseStatusOK); p != nil {
			fi := fileinfo{
				contentType: p.ContentType,
				modified:    parseModified(&p.Modified),
				etag:        p.ETag,
			}
			if ps, err := url.PathUnescape(r.Href); err == nil {
				fi.name = pathpkg.Base(ps)
			} else {
				fi.name = p.Name
			}
			fi.path = path + fi.name

			if p.Type.Local == "collection" {
				fi.path += "/"
				fi.isdir = true
			} else {
				fi.size = parseInt64(&p.Size)
			}

			files = append(files, fi)
		}

		r.Props = nil
		return nil
	}

	err := c.propfind(path, false, requiredProperties, &response{}, parse)

	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			err = newPathErrorErr("ReadDir", path, err)
		}
	}
	return files, err
}

const requiredProperties = `<d:propfind xmlns:d='DAV:'>
			<d:prop>
				<d:displayname/>
				<d:resourcetype/>
				<d:getcontentlength/>
				<d:getcontenttype/>
				<d:getetag/>
				<d:getlastmodified/>
			</d:prop>
		</d:propfind>`

// Stat returns the file stats for a specified path
func (c *client) Stat(path string) (os.FileInfo, error) {
	var fi *fileinfo
	parse := func(resp interface{}) error {
		r := resp.(*response)
		if p := getProps(r, responseStatusOK); p != nil && fi == nil {
			fi = &fileinfo{
				name:        p.Name,
				contentType: p.ContentType,
				etag:        p.ETag,
			}

			if p.Type.Local == "collection" {
				fi.path = withTrailingSlash(path)
				fi.modified = time.Unix(0, 0)
				fi.isdir = true
			} else {
				fi.path = path
				fi.size = parseInt64(&p.Size)
				fi.modified = parseModified(&p.Modified)
			}
		}

		r.Props = nil
		return nil
	}

	err := c.propfind(path, true, requiredProperties, &response{}, parse)

	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			err = newPathErrorErr("Stat", path, err)
		}
	}
	return fi, err
}

// Remove removes a remote file
func (c *client) Remove(path string) error {
	return c.RemoveAll(path)
}

// RemoveAll removes remote files
func (c *client) RemoveAll(path string) error {
	path = withLeadingSlash(path)
	rs, err := c.request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return newPathErrorErr("Remove", path, err)
	}
	err = rs.Body.Close()
	if err != nil {
		return err
	}

	if rs.StatusCode == http.StatusOK || rs.StatusCode == http.StatusNoContent || rs.StatusCode == http.StatusNotFound {
		return nil
	}

	return newPathError("Remove", path, rs.StatusCode)
}

// Mkdir makes a directory (also known as a collection in Webdav)
func (c *client) Mkdir(path string, _ os.FileMode) error {
	path = withSurroundingSlashes(pathpkg.Clean(path))
	status := c.mkcol(path)
	if status == http.StatusCreated {
		return nil
	}

	return newPathError("Mkdir", path, status)
}

// MkdirAll like mkdir -p, but for Webdav
func (c *client) MkdirAll(path string, _ os.FileMode) error {
	path = withSurroundingSlashes(pathpkg.Clean(path))
	status := c.mkcol(path)
	if status == http.StatusCreated {
		return nil
	} else if status == http.StatusConflict {
		segments := strings.Split(path, "/")
		sub := "/"
		for _, e := range segments {
			if e == "" {
				continue
			}
			sub += e + "/"
			status = c.mkcol(sub)
			if status != http.StatusCreated {
				return newPathError("MkdirAll", sub, status)
			}
		}
		return nil
	}

	return newPathError("MkdirAll", path, status)
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
func (c *client) Rename(oldpath, newpath string) error {
	return c.copymove(MethodMove, oldpath, newpath, true)
}

// RenameWithoutOverwriting renames (moves) oldpath to newpath.
// If newpath already exists, an error is returned.
func (c *client) RenameWithoutOverwriting(oldpath, newpath string) error {
	return c.copymove(MethodMove, oldpath, newpath, false)
}

// Copy copies a file from oldpath to newpath.
// If newpath already exists and is not a directory, Copy overwrites it.
func (c *client) Copy(oldpath, newpath string) error {
	return c.copymove(MethodCopy, oldpath, newpath, true)
}

// CopyWithoutOverwriting copies a file from A to B
func (c *client) CopyWithoutOverwriting(oldpath, newpath string) error {
	return c.copymove(MethodCopy, oldpath, newpath, false)
}

// ReadFile reads the contents of a remote file.
func (c *client) ReadFile(path string) ([]byte, error) {
	var stream io.ReadCloser
	var err error

	if stream, err = c.ReadStream(path); err != nil {
		return nil, err
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadStream reads the stream for a given path. The caller must
// close the returned io.ReadCloser.
func (c *client) ReadStream(path string) (io.ReadCloser, error) {
	rs, err := c.request(http.MethodGet, withLeadingSlash(path), nil, nil)
	if err != nil {
		return nil, newPathErrorErr("ReadStream", path, err)
	}

	if rs.StatusCode == http.StatusOK {
		return rs.Body, nil
	}

	rs.Body.Close()
	return nil, newPathError("ReadStream", path, rs.StatusCode)
}

// Open opens a file for writing.
// func (c *client) Create(path string) (File, error) {
// 	err := c.createParentCollection(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	s := c.put(path, stream)

// 	switch s {
// 	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
// 		return nil, nil

// 	default:
// 		return nil, newPathError("Create", path, s)
// 	}
// }

// Open opens a file for reading.
// func (c *client) Open(path string) (File, error) {
// 	rs, err := c.req(http.MethodGet, path, nil, nil)
// 	if err != nil {
// 		return nil, newPathErrorErr("Open", path, err)
// 	}

// 	if rs.StatusCode == http.StatusOK {
// 		return rs.Body, nil
// 	}

// 	rs.Body.Close()
// 	return nil, newPathError("Open", path, rs.StatusCode)
// }

// OpenFile is the generalized open call; most users will use Open
// or Create instead.
// If there is an error, it will be of type *PathError.
// func (c *client) OpenFile(path string, flag int, perm os.FileMode) (File, error) {
// 	if flag|os.O_RDONLY == os.O_RDONLY {
// 		return c.Open(path)
// 	}
// 	// if flag|os.O_RDWR == os.O_RDWR {
// 	// 	return c.Create(path)
// 	// }
// 	panic(flag)
// }

// WriteFile writes data to a given path on the webdav server.
func (c *client) WriteFile(path string, data []byte, contentType string) error {
	return c.WriteStream(path, bytes.NewReader(data), contentType)
}

// WriteStream writes from a stream to a resource on the webdav server.
func (c *client) WriteStream(path string, stream io.Reader, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := make(http.Header)
	header.Add("Content-Type", contentType)

	s, err := c.put(path, stream, header)
	if err != nil {
		return newPathErrorErr("WriteStream", path, err)
	}

	if s >= 400 {
		return newPathError("WriteStream", path, s)
	}
	return nil
}
