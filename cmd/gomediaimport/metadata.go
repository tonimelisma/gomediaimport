package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mp4 "github.com/abema/go-mp4"
	"github.com/bep/imagemeta"
)

// resolveImageFormat maps a FileInfo to the bep/imagemeta ImageFormat.
// Returns false if the format is not supported for EXIF extraction.
func resolveImageFormat(fileInfo FileInfo) (imagemeta.ImageFormat, bool) {
	switch fileInfo.MediaCategory {
	case ProcessedPicture:
		switch fileInfo.FileType {
		case JPEG:
			return imagemeta.JPEG, true
		case TIFF:
			return imagemeta.TIFF, true
		case PNG:
			return imagemeta.PNG, true
		case WEBP:
			return imagemeta.WebP, true
		case HEIF:
			return imagemeta.HEIF, true
		default:
			return 0, false
		}
	case RawPicture:
		ext := strings.ToLower(filepath.Ext(fileInfo.SourceName))
		if ext != "" {
			ext = ext[1:]
		}
		switch ext {
		case "arw":
			return imagemeta.ARW, true
		case "cr2":
			return imagemeta.CR2, true
		case "dng":
			return imagemeta.DNG, true
		case "nef":
			return imagemeta.NEF, true
		case "pef":
			return imagemeta.PEF, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

// extractImageDateTime opens an image file and extracts the creation date/time
// from EXIF or XMP metadata using the bep/imagemeta library.
func extractImageDateTime(filePath string, format imagemeta.ImageFormat) (time.Time, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("error opening file: %v", err)
	}
	defer func() { _ = file.Close() }()

	var tags imagemeta.Tags

	_, err = imagemeta.Decode(imagemeta.Options{
		R:           file,
		ImageFormat: format,
		Sources:     imagemeta.EXIF | imagemeta.XMP,
		ShouldHandleTag: func(tag imagemeta.TagInfo) bool {
			return true
		},
		HandleTag: func(tag imagemeta.TagInfo) error {
			tags.Add(tag)
			return nil
		},
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("error decoding metadata: %v", err)
	}

	t, err := tags.GetDateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("no valid date found in image metadata")
	}

	return t, nil
}

func extractCreationDateTimeFromMetadata(fileInfo FileInfo) (time.Time, error) {
	switch fileInfo.MediaCategory {
	case Sidecar:
		return time.Time{}, fmt.Errorf("sidecar files do not have embedded metadata")

	case ProcessedPicture, RawPicture:
		imgFormat, supported := resolveImageFormat(fileInfo)
		if !supported {
			return time.Time{}, fmt.Errorf("unsupported format for EXIF: %s", fileInfo.FileType)
		}
		return extractImageDateTime(filepath.Join(fileInfo.SourceDir, fileInfo.SourceName), imgFormat)

	case Video:
		filePath := filepath.Join(fileInfo.SourceDir, fileInfo.SourceName)
		return extractVideoCreationTime(filePath, fileInfo.FileType)

	case RawVideo:
		return time.Time{}, fmt.Errorf("raw video formats do not use ISO BMFF containers")

	default:
		return time.Time{}, fmt.Errorf("unsupported media category: %v", fileInfo.MediaCategory)
	}
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
	defer func() { _ = file.Close() }()

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

		t := time.Unix(int64(creationTime)-appleEpochOffset, 0)
		if t.Year() < 1970 {
			return time.Time{}, fmt.Errorf("mvhd creation time predates Unix epoch")
		}

		return t, nil
	}

	return time.Time{}, fmt.Errorf("mvhd box not found in %s", filePath)
}
