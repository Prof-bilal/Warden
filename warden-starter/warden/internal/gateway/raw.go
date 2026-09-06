package gateway

import (
	"fmt"
	"os"
)

func readRaw(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gateway config %q: %w", path, err)
	}
	return data, nil
}
