//go:build webui

package appembed

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var webDist embed.FS

func Dist() (fs.FS, bool) {
	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		return nil, false
	}
	return dist, true
}
