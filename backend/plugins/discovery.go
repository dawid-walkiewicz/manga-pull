package plugins

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

var ErrPluginIconNotFound = errors.New("plugin icon not found")

func FindPluginZips(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	files := make([]string, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.ToLower(filepath.Ext(entry.Name())) != ".zip" {
			continue
		}

		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}

func LoadPlugins(zips []string) []Plugin {
	result := make([]Plugin, 0, len(zips))

	for _, path := range zips {
		plugin, err := loadPlugin(path)
		if err != nil {
			log.Printf("load plugin %q: %v", path, err)
			continue
		}

		result = append(result, plugin)
	}

	return result
}

func loadPlugin(path string) (Plugin, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Plugin{}, err
	}
	defer r.Close()

	manifestFile, err := r.Open("manifest.json")
	if err != nil {
		return Plugin{}, fmt.Errorf("open manifest: %w", err)
	}
	defer manifestFile.Close()

	var manifest Manifest

	if err := json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		return Plugin{}, fmt.Errorf("decode manifest: %w", err)
	}

	if err := validateManifest(manifest); err != nil {
		return Plugin{}, fmt.Errorf("invalid manifest: %w", err)
	}

	var errMsg *string
	if err = validateCode(path); err != nil {
		msg := err.Error()
		errMsg = &msg
	}

	return Plugin{
		Manifest: manifest,
		Path:     path,
		Error:    errMsg,
	}, nil
}

func validateManifest(manifest Manifest) error {
	if len(manifest.ID) < 1 {
		return errors.New("id is required")
	}

	if !idPattern.MatchString(manifest.ID) {
		return fmt.Errorf("invalid id: %q", manifest.ID)
	}

	if len(manifest.Name) < 1 {
		return errors.New("name is required")
	}

	if !versionPattern.MatchString(manifest.Version) {
		return fmt.Errorf("invalid version: %q", manifest.Version)
	}

	if manifest.APIVersion < 1 {
		return fmt.Errorf("invalid apiVersion: %d", manifest.APIVersion)
	}

	if !slices.Contains([]string{"direct", "proxy"}, manifest.Transport) {
		return fmt.Errorf("unknown transport: %q", manifest.Transport)
	}

	if len(manifest.Domains) < 1 {
		return errors.New("at least one domain is required")
	}

	return nil
}

func ReadIcon(plugin *Plugin) ([]byte, string, error) {
	r, err := zip.OpenReader(plugin.Path)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	extension := []string{
		".png",
		".jpg",
		".jpeg",
		".webp",
	}

	for _, ext := range extension {
		file, err := r.Open("icon" + ext)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, "", err
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, "", err
		}

		contentType := http.DetectContentType(data)
		if !strings.HasPrefix(contentType, "image/") {
			return nil, "", fmt.Errorf("invalid plugin icon %q", "icon"+ext)
		}

		return data, contentType, nil
	}

	return nil, "", ErrPluginIconNotFound
}

func LoadPluginCode(plugin *Plugin) ([]byte, error) {
	r, err := zip.OpenReader(plugin.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	mainFile, err := r.Open("main.js")
	if err != nil {
		return nil, err
	}
	defer mainFile.Close()

	return io.ReadAll(mainFile)
}

func validateCode(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	mainFile, err := r.Open("main.js")
	if err != nil {
		return err
	}
	defer mainFile.Close()

	program, err := parser.ParseFile(nil, "main.js", mainFile, 0)
	if err != nil {
		return err
	}

	missing := make(map[string]struct{}, len(RequiredFunctions))
	for _, fn := range RequiredFunctions {
		missing[fn] = struct{}{}
	}

	for _, stmt := range program.Body {
		if funcDecl, isFunc := stmt.(*ast.FunctionDeclaration); isFunc {
			funcName := funcDecl.Function.Name.Name.String()

			delete(missing, funcName)
			if len(missing) == 0 {
				break
			}
		}
	}

	if len(missing) > 0 {
		var missingNames []string
		for fnName := range missing {
			missingNames = append(missingNames, fnName)
		}

		sort.Strings(missingNames)

		return fmt.Errorf("missing required functions: %s", strings.Join(missingNames, ", "))
	}

	return nil
}
