package plugins

import (
	"fmt"
	"log"
	"sync"

	"github.com/dop251/goja"
)

const (
	FuncSearch      = "search"
	FuncBrowse      = "browse"
	FuncGetTitle    = "getTitle"
	FuncGetChapter  = "getChapter"
	FuncProcessPage = "processPage"
)

var RequiredFunctions = []string{
	FuncSearch,
	FuncBrowse,
	FuncGetTitle,
	FuncGetChapter,
}

type PluginRuntime struct {
	Plugin *Plugin

	vm     *goja.Runtime
	client *PluginAPIClient

	mu sync.Mutex

	search      goja.Callable
	browse      goja.Callable
	getTitle    goja.Callable
	getChapter  goja.Callable
	processPage goja.Callable
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

	code, err := LoadPluginCode(plugin)
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

	search, ok := goja.AssertFunction(vm.Get(FuncSearch))
	if !ok {
		log.Println("search not found")
		return nil, fmt.Errorf("plugin %q: search function not found", plugin.ID)
	}

	runtime.search = search

	browse, ok := goja.AssertFunction(vm.Get(FuncBrowse))
	if !ok {
		log.Println("browse not found")
		return nil, fmt.Errorf("plugin %q: browse function not found", plugin.ID)
	}
	runtime.browse = browse

	getTitle, ok := goja.AssertFunction(vm.Get(FuncGetTitle))
	if !ok {
		log.Println("getTitle not found")
		return nil, fmt.Errorf("plugin %q: getTitle function not found", plugin.ID)
	}
	runtime.getTitle = getTitle

	getChapter, ok := goja.AssertFunction(vm.Get(FuncGetChapter))
	if !ok {
		log.Println("getChapter not found")
		return nil, fmt.Errorf("plugin %q: getChapter function not found", plugin.ID)
	}
	runtime.getChapter = getChapter

	processPage, ok := goja.AssertFunction(vm.Get(FuncProcessPage))
	runtime.processPage = processPage

	return &runtime, nil
}

func (p *PluginRuntime) Search(query string) ([]TitleSummary, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result, err := p.search(
		goja.Undefined(),
		p.vm.ToValue(query),
	)
	if err != nil {
		return nil, err
	}

	var titles []TitleSummary
	err = p.vm.ExportTo(result, &titles)
	if err != nil {
		return nil, err
	}

	return titles, nil
}

func (p *PluginRuntime) Browse() ([]TitleSummary, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result, err := p.browse(goja.Undefined())
	if err != nil {
		return nil, err
	}

	var titles []TitleSummary
	err = p.vm.ExportTo(result, &titles)
	if err != nil {
		return nil, err
	}

	return titles, nil
}

func (p *PluginRuntime) GetTitle(id string) (*TitleDetails, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result, err := p.getTitle(
		goja.Undefined(),
		p.vm.ToValue(id),
	)
	if err != nil {
		return nil, err
	}

	var title TitleDetails
	err = p.vm.ExportTo(result, &title)
	if err != nil {
		return nil, err
	}

	return &title, nil
}

func (p *PluginRuntime) GetChapter(chapterID string) (*ChapterDescriptor, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result, err := p.getChapter(
		goja.Undefined(),
		p.vm.ToValue(chapterID),
	)
	if err != nil {
		return nil, err
	}

	var chapter ChapterDescriptor
	err = p.vm.ExportTo(result, &chapter)
	if err != nil {
		return nil, err
	}

	return &chapter, nil
}

func (p *PluginRuntime) ProcessPage(
	page PageDescriptor,
	data []byte,
) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.processPage == nil {
		return data, nil
	}

	result, err := p.processPage(
		goja.Undefined(),
		p.vm.ToValue(page),
		p.vm.ToValue(data),
	)
	if err != nil {
		return nil, err
	}

	var processed []byte
	if err := p.vm.ExportTo(result, &processed); err != nil {
		return nil, err
	}

	return processed, nil
}
