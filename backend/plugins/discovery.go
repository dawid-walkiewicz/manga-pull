package plugins

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func FindPlugins(dir string) ([]string, error) {
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
	plugins := make([]Plugin, 0, len(zips))
	for _, p := range zips {
		r, err := zip.OpenReader(p)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, f := range r.File {
			if f.Name == "manifest.json" {
				rc, err := f.Open()
				if err != nil {
					log.Println(err)
					continue
				}

				decoder := json.NewDecoder(rc)

				var plugin Plugin
				if err := decoder.Decode(&plugin); err != nil {
					log.Println(err)
					continue
				}

				err = validateManifest(plugin)
				if err != nil {
					log.Printf("invalid plugin %q: %v", p, err)
					continue
				}
				plugin.Path = p
				plugins = append(plugins, plugin)

				rc.Close()
			}
		}
		r.Close()
	}
	return plugins
}

func validateManifest(manifest Plugin) error {
	if len(manifest.ID) < 1 {
		return errors.New("id is required")
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
