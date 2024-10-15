package main

import (
	"fmt"
	"time"
)

func extractCreationDateTimeFromMetadata(fileInfo FileInfo) (time.Time, error) {
	if fileInfo.MediaCategory == ProcessedPicture || fileInfo.MediaCategory == RawPicture {
		// TODO: detect EXIF
		return time.Time{}, fmt.Errorf("EXIF detection not implemented yet")
	} else if fileInfo.MediaCategory == Video {
		// TODO: Implement video metadata extraction
		return time.Time{}, fmt.Errorf("video metadata extraction not implemented yet")
	}

	return time.Time{}, fmt.Errorf("unsupported media category: %v", fileInfo.MediaCategory)
}
