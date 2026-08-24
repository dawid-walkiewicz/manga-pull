package plugins

import (
	"archive/zip"
	"fmt"
	"io"
	"log"

	"github.com/dop251/goja"
)

func ReadPlugin(plugin *Plugin) ([]byte, error) {
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

	code, err := io.ReadAll(mainFile)
	if err != nil {
		return nil, err
	}

	return code, nil
}

func NewRuntime(plugin *Plugin) (*PluginRuntime, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	client := NewPluginAPIClient(plugin)

	if err := vm.Set("api", map[string]any{
		"request": client.Request,
		"log":     client.Log,
	}); err != nil {
		return nil, fmt.Errorf("register plugin api: %w", err)
	}

	code, err := ReadPlugin(plugin)
	if err != nil {
		return nil, fmt.Errorf("read plugin: %w", err)
	}

	if _, err := vm.RunScript("main.js", string(code)); err != nil {
		return nil, fmt.Errorf("run plugin: %w", err)
	}

	runtime := PluginRuntime{
		Plugin: plugin,
		vm:     vm,
		client: client,
	}

	search, ok := goja.AssertFunction(vm.Get("search"))
	if !ok {
		log.Println("search not found")
		return nil, fmt.Errorf("plugin %q: search function not found", plugin.ID)
	}

	runtime.search = search

	browse, ok := goja.AssertFunction(vm.Get("browse"))
	if !ok {
		log.Println("browse not found")
		return nil, fmt.Errorf("plugin %q: browse function not found", plugin.ID)
	}
	runtime.browse = browse

	getTitle, ok := goja.AssertFunction(vm.Get("getTitle"))
	if !ok {
		log.Println("getTitle not found")
		return nil, fmt.Errorf("plugin %q: getTitle function not found", plugin.ID)
	}
	runtime.getTitle = getTitle

	downloadChapter, ok := goja.AssertFunction(vm.Get("downloadChapter"))
	if !ok {
		log.Println("downloadChapter not found")
		return nil, fmt.Errorf("plugin %q: downloadChapter function not found", plugin.ID)
	}
	runtime.downloadChapter = downloadChapter

	return &runtime, nil
}
