package main

import (
	"database/sql"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse video id", err)
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

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "That video doesn't exist", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not fetch video data", err)
	}

	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You do not own this video", err)
		return
	}

	const maxMemory = 1 << 30
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not parse file", err)
		return
	}

	fmt.Println("uploading video file", videoID, "by user", userID)

	file, header, err := r.FormFile("video")
	defer file.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get file from request", err)
		return
	}

	videoReader := http.MaxBytesReader(w, file, maxMemory)

	mimeType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type", err)
		return
	}

	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusUnprocessableEntity, "video is not mp4", fmt.Errorf("video header is not mp4"))
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	defer tmpFile.Close()
	defer func() {
		info, _ := tmpFile.Stat()
		err := os.Remove(filepath.Join("/tmp", info.Name()))
		if err != nil {
			fmt.Printf("error removing file: %+v\n", err)
		}
	}()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save video", err)
		return
	}

	_, err = io.Copy(tmpFile, videoReader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not upload video", err)
		return
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem processing the video", err)
		return
	}

	assetPath, err := getAssetPath(mimeType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem generating asset path", err)
		return
	}

	_, err = cfg.s3client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(assetPath),
		Body:        tmpFile,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem uploading to S3", err)
		return
	}

	url := cfg.getObjectUrl(assetPath)
	videoMetaData.VideoURL = &url

	err = cfg.db.UpdateVideo(videoMetaData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem saving to db", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, videoMetaData)
}
