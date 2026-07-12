package view

import (
	"net/url"
	"strconv"
	"time"
)

// assetVersion stamps static asset URLs so the aggressive /static
// Cache-Control (1 year, ADR-016) can never strand a returning browser on
// stale CSS/JS. Releases override it with the build version at boot; the
// process-start default means dev-server restarts bust caches too.
var assetVersion = strconv.FormatInt(time.Now().Unix(), 10)

// SetAssetVersion overrides the asset stamp with the build version (main's
// -X main.version ldflag). Blank and "dev" are not real versions — they keep
// the process-start default rather than erasing the stamp.
func SetAssetVersion(v string) {
	if v == "" || v == "dev" {
		return
	}
	assetVersion = url.QueryEscape(v)
}

// AssetURL returns the versioned URL for a static asset path.
func AssetURL(path string) string {
	return path + "?v=" + assetVersion
}
