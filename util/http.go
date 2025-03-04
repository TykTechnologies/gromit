package util

import (
	"fmt"
	"net/http"
)

// GetContentLength gets the size of a URL without downloading it
func GetContentLength(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, fmt.Errorf("failed to send HEAD request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status for HEAD: %s", resp.Status)
	}

	size := resp.ContentLength
	if size < 0 {
		return 0, fmt.Errorf("content length not found")
	}

	return size, nil
}
