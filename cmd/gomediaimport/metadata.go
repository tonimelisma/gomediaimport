package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	mp4 "github.com/abema/go-mp4"
	"github.com/evanoberholster/imagemeta"
)

func extractCreationDateTimeFromMetadata(fileInfo FileInfo) (time.Time, error) {
	if fileInfo.MediaCategory == Sidecar {
		return time.Time{}, fmt.Errorf("sidecar files do not have embedded metadata")
	}

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
		filePath := filepath.Join(fileInfo.SourceDir, fileInfo.SourceName)
		return extractVideoCreationTime(filePath, fileInfo.FileType)
	} else if fileInfo.MediaCategory == RawVideo {
		return time.Time{}, fmt.Errorf("raw video formats do not use ISO BMFF containers")
	}

	return time.Time{}, fmt.Errorf("unsupported media category: %v", fileInfo.MediaCategory)
}

// appleEpochOffset is the number of seconds between the Apple/Mac epoch
// (1904-01-01 00:00:00 UTC) and the Unix epoch (1970-01-01 00:00:00 UTC).
const appleEpochOffset = 2082844800

// isoBaseMediaFileTypes contains video FileTypes that use the ISO Base Media
// File Format (ISO 14496-12) and store creation time in the mvhd box.
var isoBaseMediaFileTypes = map[FileType]bool{
	MP4:     true,
	MOV:     true,
	M4V:     true,
	THREEGP: true,
	THREEG2: true,
}

// extractVideoCreationTime reads the moov>mvhd box from an ISO BMFF container
// and returns the creation time. For non-ISO-BMFF video types, it returns an error
// so the caller falls back to filesystem mtime.
func extractVideoCreationTime(filePath string, fileType FileType) (time.Time, error) {
	if !isoBaseMediaFileTypes[fileType] {
		return time.Time{}, fmt.Errorf("file type %s is not an ISO BMFF container", fileType)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	boxes, err := mp4.ExtractBoxesWithPayload(file, nil, []mp4.BoxPath{
		{mp4.BoxTypeMoov(), mp4.BoxTypeMvhd()},
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("error reading MP4 structure: %v", err)
	}

	for _, box := range boxes {
		mvhd, ok := box.Payload.(*mp4.Mvhd)
		if !ok {
			continue
		}

		creationTime := mvhd.GetCreationTime()
		if creationTime == 0 {
			return time.Time{}, fmt.Errorf("mvhd creation time is zero")
		}

		t := time.Unix(int64(creationTime)-appleEpochOffset, 0).UTC()
		if t.Year() < 1970 {
			return time.Time{}, fmt.Errorf("mvhd creation time predates Unix epoch")
		}

		return t, nil
	}

	return time.Time{}, fmt.Errorf("mvhd box not found in %s", filePath)
}
