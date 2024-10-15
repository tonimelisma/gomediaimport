package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/evanoberholster/imagemeta"
)

func extractCreationDateTimeFromMetadata(fileInfo FileInfo) (time.Time, error) {
	if fileInfo.MediaCategory == ProcessedPicture || fileInfo.MediaCategory == RawPicture {
		filePath := filepath.Join(fileInfo.SourceDir, fileInfo.SourceName)
		file, err := os.Open(filePath)
		if err != nil {
			return time.Time{}, fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()

		exif, err := imagemeta.Decode(file)
		if err != nil {
			return time.Time{}, fmt.Errorf("error decoding EXIF: %v", err)
		}

		if !exif.DateTimeOriginal().IsZero() {
			return exif.DateTimeOriginal(), nil
		}

		if !exif.CreateDate().IsZero() {
			return exif.CreateDate(), nil
		}

		return time.Time{}, fmt.Errorf("no valid date found in image metadata")
	} else if fileInfo.MediaCategory == Video {
		// TODO: Implement video metadata extraction
		return time.Time{}, fmt.Errorf("video metadata extraction not implemented yet")
	}

	return time.Time{}, fmt.Errorf("unsupported media category: %v", fileInfo.MediaCategory)
}
