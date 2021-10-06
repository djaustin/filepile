package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

func main() {
	uploadDir := os.Getenv("FILEPILE_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./files"
	}
	maxUploadBytes, err := strconv.Atoi(os.Getenv("FILEPILE_MAX_UPLOAD_SIZE"))
	if err != nil {
		maxUploadBytes = 2 * 1024 * 1024
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir(uploadDir))
	http.Handle("/", fs)
	http.HandleFunc("/upload", uploadFileHandler(uploadDir, int64(maxUploadBytes)))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func uploadFileHandler(uploadDir string, maxBytes int64) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			fmt.Printf("Unable to parse multipart form on upload: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Cannot parse form"))
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			fmt.Printf("Unable to read file from multipart form: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid file"))
			return
		}
		defer file.Close()
		if fileHeader.Size > maxBytes {
			fmt.Printf("Unable to process file as file is too large: %v\n", err)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("File too large"))
			return
		}
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Printf("Unable to parse multipart form on upload: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error reading file"))
			return
		}
		detectedFileType := http.DetectContentType(fileBytes)
		fileEndings, err := mime.ExtensionsByType(detectedFileType)
		if err != nil {
			fmt.Printf("Unable to detect file ending: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error reading file"))
			return
		}
		id := uuid.New()
		fileName := id.String() + fileEndings[0]
		writePath := filepath.Join(uploadDir, fileName)
		fmt.Printf("Creating file: %s", writePath)
		newFile, err := os.Create(writePath)
		if err != nil {
			fmt.Printf("Unable to save file: %v", err)
			return
		}
		defer newFile.Close()
		newFile.Write(fileBytes)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fileName))
	})
}
