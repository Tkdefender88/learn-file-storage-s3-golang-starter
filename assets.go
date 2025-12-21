package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"mime"
	"os"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetURL(fileName string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
}

func getThumbnailName(mediaExt string) string {
	b := [32]byte{}
	rand.Read(b[:])
	return fmt.Sprintf("%s%s", base64.RawURLEncoding.EncodeToString(b[:]), mediaExt)
}

func mediaTypeToExt(mediaType string) (string, error) {
	t, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return "", err
	}
	switch t {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	default:
		return "", errors.New("unsupported media type")
	}
}
