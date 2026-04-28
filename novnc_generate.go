//go:build novnc_generate

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shurcooL/vfsgen"
	"github.com/spkg/zipfs"
)

const noVNCZip = "https://github.com/novnc/noVNC/archive/refs/tags/v1.7.0.zip"
const vncScript = ""

// normalizeSemver normalizes a semver string by removing any pre-release or build metadata.
// v1.1.0-hotfix1 -> v1.1.0
// v1.5.0 -> v1.5.0
// v2.0.3+build.7 -> v2.0.3
func normalizeSemver(v string) (string, error) {
	// Remove pre-release and build metadata
	if i := strings.IndexAny(v, "-+"); i != -1 {
		v = v[:i]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid semver: %s", v)
	}

	return strings.Join(parts, "."), nil
}

func main() {
	resp, err := http.Get(noVNCZip)
	if err != nil {
		panic(err)
	}

	f, err := os.CreateTemp("", "novnc*.zip")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		panic(err)
	}

	f.Close()
	resp.Body.Close()

	err = modifyZip(f.Name())
	if err != nil {
		panic(err)
	}

	zfs, err := zipfs.New(f.Name())
	if err != nil {
		panic(err)
	}

	err = vfsgen.Generate(zfs, vfsgen.Options{
		Filename:        "novnc_generated.go",
		PackageName:     "main",
		VariableName:    "noVNC",
		VariableComment: "noVNC is the latest version of noVNC from GitHub as a http.FileSystem",
	})
	if err != nil {
		panic(err)
	}
}

// modifyZip adds the custom easy-novnc code into the noVNC zip file.
func modifyZip(zf string) error {
	newRootName := "noVNC"
	includePrefixes := []string{
		"core/",
		"app/",
		"po/",
		"vnc.html",
		"vendor/",
		"utils/",
		"defaults.json",
		"mandatory.json",
		"package.json",
	}
	shouldInclude := func(relPath string) bool {
		for _, p := range includePrefixes {
			if strings.HasPrefix(relPath, p) {
				return true
			}
		}
		return false
	}

	buf, err := os.ReadFile(zf)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	f, err := os.Create(zf)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	_, err = zw.CreateHeader(&zip.FileHeader{
		Name:   newRootName + "/",
		Method: zip.Store,
	})
	if err != nil {
		return err
	}

	var found bool
	for _, e := range zr.File {
		parts := strings.SplitN(e.Name, "/", 2)

		if len(parts) < 2 {
			continue
		}
		relPath := parts[1]

		if !shouldInclude(relPath) {
			continue
		}

		newName := filepath.Join(newRootName, relPath)
		if strings.HasSuffix(e.Name, "/") {
			newName += "/"
		}

		var w io.Writer

		rc, err := e.Open()
		if err != nil {
			return err
		}

		fbuf, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		if filepath.Base(e.Name) == "vnc.html" {
			found = true
			fbuf = bytes.ReplaceAll(fbuf, []byte("</body>"), []byte(fmt.Sprintf("<script>%s</script></body>", vncScript)))
			fi, err := os.Stat("novnc_generate.go")
			if err != nil {
				return err
			}
			w, err = zw.CreateHeader(&zip.FileHeader{
				Name:          newName,
				Flags:         e.Flags,
				Method:        e.Method,
				Modified:      fi.ModTime(),
				Extra:         e.Extra,
				ExternalAttrs: e.ExternalAttrs,
			})
		} else {
			w, err = zw.CreateHeader(&zip.FileHeader{
				Name:          newName,
				Flags:         e.Flags,
				Method:        e.Method,
				Modified:      e.Modified,
				Extra:         e.Extra,
				ExternalAttrs: e.ExternalAttrs,
			})
		}

		if err != nil {
			return err
		}

		_, err = io.Copy(w, bytes.NewReader(fbuf))
		if err != nil {
			return err
		}
		rc.Close()
	}

	if !found {
		return errors.New("could not find vnc.html")
	}

	return nil
}
