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

var idPattern = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

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

	return Plugin{
		Manifest: manifest,
		Path:     path,
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
