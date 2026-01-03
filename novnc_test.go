package main

import (
	"io"
	"os"
	"testing"

	"github.com/shurcooL/httpfs/vfsutil"
)

func TestNoVNC(t *testing.T) {
	f, err := noVNC.Open("noVNC")
	if err != nil {
		t.Errorf("could not open noVNC root dir: %v", err)
	}

	_, err = f.Readdir(1)
	if err != nil {
		t.Errorf("could not read noVNC root dir: %v", err)
	}

	f, err = noVNC.Open("noVNC/vnc.html")
	if err != nil {
		t.Errorf("could not open vnc.html: %v", err)
	}

	buf, err := io.ReadAll(f)
	if err != nil {
		t.Errorf("could not read vnc.html: %v", err)
	}

	if len(buf) < 100 {
		t.Errorf("vnc.html is too small")
	}

	f, err = noVNC.Open("noVNC/package.json")
	if err != nil {
		t.Errorf("could not open VERSION: %v", err)
	}

	buf, err = io.ReadAll(f)
	if err != nil {
		t.Errorf("could not read VERSION: %v", err)
	}

	t.Logf("noVNC %s", string(buf))

	err = vfsutil.WalkFiles(noVNC, "/", func(path string, info os.FileInfo, rs io.ReadSeeker, err error) error {
		return err
	})
	if err != nil {
		t.Errorf("could not read fs: %v", err)
	}
}
