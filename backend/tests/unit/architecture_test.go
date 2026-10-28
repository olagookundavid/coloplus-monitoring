package unit

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbidden lists, per layer, the import prefixes that layer must never use.
// It mirrors the depguard rules in .golangci.yml so a violation fails
// `go test` locally as well as in lint.
var forbidden = map[string][]string{
	"internal/domain": {
		"coloplus-backend/", "pkg-common", "github.com/", "golang.org/x/",
	},
	"internal/ports": {
		"coloplus-backend/internal/adapters", "coloplus-backend/internal/services",
		"coloplus-backend/internal/transport", "github.com/", "golang.org/x/",
	},
	"internal/services": {
		"coloplus-backend/internal/adapters", "coloplus-backend/internal/transport",
		"github.com/", "net/http", "database/sql",
	},
	"internal/adapters": {
		"coloplus-backend/internal/services", "coloplus-backend/internal/transport",
	},
	"internal/transport": {
		"coloplus-backend/internal/adapters", "github.com/jackc/",
	},
}

// allowedExceptions are imports that look forbidden by prefix but are fine.
var allowedExceptions = map[string][]string{
	"internal/domain": {"coloplus-backend/internal/domain"},
	"internal/ports":  {"coloplus-backend/internal/domain"},
}

func TestLayerImportDirection(t *testing.T) {
	root := filepath.Join("..", "..")
	for layer, banned := range forbidden {
		dir := filepath.Join(root, filepath.FromSlash(layer))
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				p, _ := strconv.Unquote(imp.Path.Value)
				if isBanned(layer, p, banned) {
					t.Errorf("%s imports %q, which %s must not depend on", path, p, layer)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func isBanned(layer, imp string, banned []string) bool {
	for _, ok := range allowedExceptions[layer] {
		if imp == ok || strings.HasPrefix(imp, ok+"/") {
			return false
		}
	}
	for _, b := range banned {
		if strings.HasPrefix(imp, b) {
			return true
		}
	}
	return false
}
