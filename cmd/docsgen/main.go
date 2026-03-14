package main

import (
	"path/filepath"
	"runtime"

	"github.com/alexpls/untils/internal/must"
)

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to locate docsgen source file")
	}

	root := must.NoErrVal(filepath.Abs(filepath.Join(filepath.Dir(filename), "..", "..")))
	must.NoErr(WriteGeneratedFile(root))
}
