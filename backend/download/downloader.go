package download

import (
	"context"
	"fmt"
	"io"
	"main/common"
	"net/http"
	"os"
)

type ChapterDownloader struct {
	client *http.Client
}

func NewDownloader() *ChapterDownloader {
	return &ChapterDownloader{
		client: &http.Client{},
	}
}

func (d *ChapterDownloader) DownloadFile(ctx context.Context, url string, filePath string, domains []string) error {
	err := common.CheckDomains(url, domains)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsuccessful download, HTTP status: %s", resp.Status)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
