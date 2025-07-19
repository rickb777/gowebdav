package main

import (
	"errors"
	"flag"
	"fmt"
	d "github.com/rickb777/gowebdav"
	"github.com/rickb777/gowebdav/auth"
	netrcpkg "github.com/rickb777/gowebdav/netrc"
	"github.com/rickb777/httpclient/logging"
	"github.com/rickb777/httpclient/logging/logger"
	"github.com/rickb777/httpclient/loggingclient"
	"io"
	"net/http"
	"os"
	userpkg "os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	root := flag.String("root", os.Getenv("ROOT"), "WebDAV Endpoint [ENV.ROOT]")
	user := flag.String("user", os.Getenv("USER"), "User [ENV.USER]")
	site := flag.String("site", os.Getenv("SITE_URL"), "Site URL [ENV.SITE_URL]")
	password := flag.String("pw", os.Getenv("PASSWORD"), "Password [ENV.PASSWORD]")
	netrc := flag.String("netrc", filepath.Join(getHome(), ".netrc"), "read credentials from netrc file")
	authenticator := flag.String("auth", "", "specify which authentication to use: basic, digest")
	verbose := flag.Bool("v", false, "verbose logging")
	veryVerbose := flag.Bool("z", false, "very verbose logging")
	method := flag.String("X", "", `Method:
	ls <PATH>
	stat <PATH>

	mkdir <PATH>
	mkdirall <PATH>

	get <PATH> [<FILE>]
	put <PATH> [<FILE>]

	mv <OLD> <NEW>
	cp <OLD> <NEW>

	rm <PATH>
	`)
	flag.Parse()

	if *root == "" {
		fail("Set WebDAV ROOT")
	}

	if flag.NArg() < 1 {
		fail("Too few arguments")
	}

	if *password == "" {
		if u, p := netrcpkg.ReadConfig(*root, *netrc); u != "" && p != "" {
			user = &u
			password = &p
		}
	}

	lgr := logger.LogWriter(os.Stdout, nil)
	level := logging.Off
	if *veryVerbose {
		level = logging.WithHeadersAndBodies
	} else if *verbose {
		level = logging.WithHeaders
	}
	httpClient := loggingclient.New(http.DefaultClient, lgr, level)

	c := d.NewClient(*root,
		d.SetAuthentication(selectAuthenticator(*user, *password, *site, *authenticator)),
		d.SetHttpClient(httpClient))

	cmd := getCmd(*method)

	if e := cmd(c, flag.Args()...); e != nil {
		fail(e)
	}
}

func selectAuthenticator(user, pw, site, authenticator string) auth.Authenticator {
	switch authenticator {
	case "basic":
		return auth.Basic(user, pw)
	case "digest":
		return auth.Digest(user, pw)
	case "saml":
		return auth.SAML(user, pw, site, nil)
	default:
		return auth.Deferred(user, pw)
	}
}

func fail(err interface{}) {
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(-1)
}

func getHome() string {
	u, e := userpkg.Current()
	if e != nil {
		return os.Getenv("HOME")
	}

	if u != nil {
		return u.HomeDir
	}

	switch runtime.GOOS {
	case "windows":
		return ""
	default:
		return "~/"
	}
}

type command func(c d.Client, p0 ...string) error

func getCmd(method string) command {
	switch strings.ToLower(method) {
	case "ls", "list", "propfind":
		return cmdLs

	case "stat":
		return cmdStat

	case "get", "pull", "read":
		return cmdGet

	case "delete", "rm", "del":
		return cmdRm

	case "mkcol", "mkdir":
		return cmdMkdir

	case "mkcolall", "mkdirall", "mkdirp":
		return cmdMkdirAll

	case "rename", "mv", "move":
		return cmdMv

	case "copy", "cp":
		return cmdCp

	case "put", "push", "write":
		return cmdPut

	default:
		return func(c d.Client, p ...string) (err error) {
			return errors.New("Unsupported method: " + method)
		}
	}
}

func failIfTooManyArgs(p []string, max int) {
	if len(p) > max {
		fail("Too many arguments")
	}
}

func cmdLs(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 1)

	files, err := c.ReadDir(p[0])
	if err == nil {
		fmt.Println(fmt.Sprintf("ReadDir: '%s' entries: %d ", p[0], len(files)))
		for _, f := range files {
			fmt.Println(f)
		}
	}
	return
}

func cmdStat(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 1)

	file, err := c.Stat(p[0])
	if err == nil {
		fmt.Println(file)
	}
	return
}

func cmdGet(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 2)

	bytes, err := c.ReadFile(p[0])
	if err == nil {
		p1 := filepath.Join(".", p[0])
		if len(p) > 1 {
			p1 = p[1]
		}
		err = writeFile(p1, bytes, 0644)
		if err == nil {
			fmt.Println(fmt.Sprintf("Written %d bytes to: %s", len(bytes), p1))
		}
	}
	return
}

func cmdRm(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 1)

	if err = c.Remove(p[0]); err == nil {
		fmt.Println("Remove: " + p[0])
	}
	return
}

func cmdMkdir(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 1)

	if err = c.Mkdir(p[0], 0755); err == nil {
		fmt.Println("Mkdir: " + p[0])
	}
	return
}

func cmdMkdirAll(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 1)

	if err = c.MkdirAll(p[0], 0755); err == nil {
		fmt.Println("MkdirAll: " + p[0])
	}
	return
}

func cmdMv(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 2)

	if err = c.Rename(p[0], p[1]); err == nil {
		fmt.Println("Rename: " + p[0] + " -> " + p[1])
	}
	return
}

func cmdCp(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 2)

	if err = c.Copy(p[0], p[1]); err == nil {
		fmt.Println("Copy: " + p[0] + " -> " + p[1])
	}
	return
}

func cmdPut(c d.Client, p ...string) (err error) {
	failIfTooManyArgs(p, 2)

	p1 := filepath.Join(".", p[0])
	if len(p) > 1 {
		p1 = p[1]
	}

	stream, err := getStream(p1)
	if err != nil {
		return
	}
	defer stream.Close()

	if err = c.WriteStream(p[0], stream, ""); err == nil {
		fmt.Println("Put: " + p1 + " -> " + p[0])
	}
	return
}

func writeFile(path string, bytes []byte, mode os.FileMode) error {
	parent := filepath.Dir(path)
	if _, e := os.Stat(parent); os.IsNotExist(e) {
		if e := os.MkdirAll(parent, os.ModePerm); e != nil {
			return e
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	return err
}

func getStream(pathOrString string) (io.ReadCloser, error) {
	fi, err := os.Stat(pathOrString)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, &os.PathError{
			Op:   "Open",
			Path: pathOrString,
			Err:  errors.New("Path: '" + pathOrString + "' is a directory"),
		}
	}

	f, err := os.Open(pathOrString)
	if err == nil {
		return f, nil
	}

	return nil, &os.PathError{
		Op:   "Open",
		Path: pathOrString,
		Err:  err,
	}
}
