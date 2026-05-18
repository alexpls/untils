package docscontent

import (
	"embed"
	"io/fs"

	"github.com/alexpls/untils/internal/must"
)

//go:embed public
var content embed.FS

func PublicFS() fs.FS {
	return must.NoErrVal(fs.Sub(content, "public"))
}
