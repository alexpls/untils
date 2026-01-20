package public

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed css js
var publicFs embed.FS

var fingerprintRegex regexp.Regexp = *regexp.MustCompile(`(-\w{32})(\.\w+)$`)
var fingerprinted = make(map[string]string)
var devMode = false

func init() {
	_ = buildFingerprintMap()
}

func SetDevMode() {
	devMode = true
}

func Handler() http.Handler {
	var handler http.Handler
	if devMode {
		handler = http.FileServer(http.Dir("public"))
	} else {
		handler = http.FileServer(http.FS(publicFs))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/assets/")

		if devMode {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else if fingerprintRegex.MatchString(path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			path = fingerprintRegex.ReplaceAllString(path, "$2")
		}

		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		r2.URL.Path = path

		handler.ServeHTTP(w, r2)
	})
}

func AssetURL(path string) string {
	if devMode {
		return filepath.Join("/assets", path)
	}

	fp, found := fingerprinted[path]
	if !found {
		return filepath.Join("/assets", path)
	}
	dir, file := filepath.Split(path)
	ext := filepath.Ext(file)
	return "/assets/" + dir + strings.TrimSuffix(file, ext) + "-" + fp + ext
}

func fingerprintFile(path string) (string, error) {
	h := md5.New()
	var f io.ReadCloser
	var err error

	if devMode {
		f, err = os.Open(filepath.Join("public", path))
	} else {
		f, err = publicFs.Open(path)
	}
	if err != nil {
		return "", err
	}
	defer f.Close() // nolint:errcheck
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func buildFingerprintMap() error {
	fingerprinted = make(map[string]string)

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		hash, err := fingerprintFile(path)
		if err != nil {
			return err
		}
		fingerprinted[path] = hash

		return nil
	}

	return fs.WalkDir(publicFs, ".", walkFunc)
}
