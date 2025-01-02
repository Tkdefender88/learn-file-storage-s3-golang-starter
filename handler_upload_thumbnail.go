package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not parse file", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get file from request", err)
		return
	}

	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Could not find video", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error getting the video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "This is not your video", fmt.Errorf("Not your video"))
	}

	assetPath := getAssetPath(videoID, mediaType)

	assetFile, err := os.Create(cfg.getAssetDiskPath(assetPath))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save thumbnail to file", err)
		return
	}

	_, err = io.Copy(assetFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not write thumbnail to file", err)
		return
	}

	thumbUrl := cfg.getAssetUrl(assetPath)
	video.ThumbnailURL = &thumbUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
