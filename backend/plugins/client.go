package plugins

import (
	"github.com/dop251/goja"
)

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

// func (p *PluginRuntime) DownloadChapter(id string) (ChapterDownload, error)
