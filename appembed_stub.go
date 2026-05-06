//go:build !webui

package appembed

import "io/fs"

func Dist() (fs.FS, bool) {
	return nil, false
}
