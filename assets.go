package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(assetID uuid.UUID, mediaType string) string {
	ext := getExtensionFromType(mediaType)
	return fmt.Sprintf("%s%s", assetID.String(), ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetUrl(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func getExtensionFromType(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) < 2 {
		return ".bin"
	}
	return "." + parts[1]
}
