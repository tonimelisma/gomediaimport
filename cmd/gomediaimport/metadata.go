package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bep/imagemeta"
	"github.com/tonimelisma/videometa"
)

const (
	videoTimestampSourceQuickTime = "quicktime"
	videoTimestampSourceVendor    = "vendor"
	videoTimestampSourceFallback  = "fallback"

	videoTimestampFallbackUnsupportedContainer = "unsupported_container"
	videoTimestampFallbackDecodeError          = "decode_error"
	videoTimestampFallbackNoDateTime           = "no_datetime"
)

// VideoMetadata stores the subset of decoded video metadata that is useful for
// import behavior, diagnostics, and future metadata-driven features.
type VideoMetadata struct {
	ChosenTimestamp         time.Time
	TimestampSource         string
	TimestampTag            string
	TimestampNamespace      string
	TimestampFallbackReason string
	Width                   int
	Height                  int
	Duration                time.Duration
	Rotation                int
	Codec                   string
	GPSLatitude             *float64
	GPSLongitude            *float64
	Make                    string
	Model                   string
	Warnings                []string
}

type mediaMetadata struct {
	CreationDateTime time.Time
	VideoMetadata    *VideoMetadata
}

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

func extractMetadata(fileInfo FileInfo) (mediaMetadata, error) {
	switch fileInfo.MediaCategory {
	case Sidecar:
		return mediaMetadata{}, fmt.Errorf("sidecar files do not have embedded metadata")

	case ProcessedPicture, RawPicture:
		imgFormat, supported := resolveImageFormat(fileInfo)
		if !supported {
			return mediaMetadata{}, fmt.Errorf("unsupported format for EXIF: %s", fileInfo.FileType)
		}
		t, err := extractImageDateTime(filepath.Join(fileInfo.SourceDir, fileInfo.SourceName), imgFormat)
		if err != nil {
			return mediaMetadata{}, err
		}
		return mediaMetadata{CreationDateTime: t}, nil

	case Video:
		filePath := filepath.Join(fileInfo.SourceDir, fileInfo.SourceName)
		return extractVideoMetadata(filePath, fileInfo.FileType, fileInfo.CreationDateTime)

	case RawVideo:
		return mediaMetadata{}, fmt.Errorf("raw video formats do not use ISO BMFF containers")

	default:
		return mediaMetadata{}, fmt.Errorf("unsupported media category: %v", fileInfo.MediaCategory)
	}
}

// isoBaseMediaFileTypes contains video FileTypes that use the ISO Base Media
// File Format (ISO 14496-12) and are decoded with videometa.
var isoBaseMediaFileTypes = map[FileType]bool{
	MP4:     true,
	MOV:     true,
	M4V:     true,
	THREEGP: true,
	THREEG2: true,
}

func extractVideoMetadata(filePath string, fileType FileType, fallbackTime time.Time) (mediaMetadata, error) {
	videoMetadata := &VideoMetadata{
		ChosenTimestamp: fallbackTime,
	}

	if !isoBaseMediaFileTypes[fileType] {
		videoMetadata.TimestampSource = videoTimestampSourceFallback
		videoMetadata.TimestampFallbackReason = videoTimestampFallbackUnsupportedContainer
		return mediaMetadata{
			CreationDateTime: fallbackTime,
			VideoMetadata:    videoMetadata,
		}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		videoMetadata.TimestampSource = videoTimestampSourceFallback
		videoMetadata.TimestampFallbackReason = videoTimestampFallbackDecodeError
		videoMetadata.Warnings = append(videoMetadata.Warnings, fmt.Sprintf("error opening file: %v", err))
		return mediaMetadata{
			CreationDateTime: fallbackTime,
			VideoMetadata:    videoMetadata,
		}, nil
	}
	defer func() { _ = file.Close() }()

	tags, result, err := videometa.DecodeAll(videometa.Options{
		R:       file,
		Sources: videometa.QUICKTIME | videometa.CONFIG | videometa.MAKERNOTES | videometa.XML,
		Warnf: func(format string, args ...any) {
			videoMetadata.Warnings = append(videoMetadata.Warnings, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		videoMetadata.TimestampSource = videoTimestampSourceFallback
		videoMetadata.TimestampFallbackReason = videoTimestampFallbackDecodeError
		videoMetadata.Warnings = append(videoMetadata.Warnings, fmt.Sprintf("videometa decode failed: %v", err))
		return mediaMetadata{
			CreationDateTime: fallbackTime,
			VideoMetadata:    videoMetadata,
		}, nil
	}

	videoMetadata.Width = result.VideoConfig.Width
	videoMetadata.Height = result.VideoConfig.Height
	videoMetadata.Duration = result.VideoConfig.Duration
	videoMetadata.Rotation = result.VideoConfig.Rotation
	videoMetadata.Codec = result.VideoConfig.Codec

	if lat, lon, err := tags.GetLatLong(); err == nil {
		videoMetadata.GPSLatitude = &lat
		videoMetadata.GPSLongitude = &lon
	}

	videoMetadata.Make = findFirstStringValue(
		[]map[string]videometa.TagInfo{tags.QuickTime(), tags.MakerNotes(), tags.XML()},
		"Make",
	)
	videoMetadata.Model = findFirstStringValue(
		[]map[string]videometa.TagInfo{tags.QuickTime(), tags.MakerNotes(), tags.XML()},
		"Model",
	)

	timestamp, err := tags.GetDateTime()
	provenanceTimestamp, source, tag, namespace, found := resolveVideoTimestampProvenance(tags)
	if err != nil && found {
		timestamp = provenanceTimestamp
		err = nil
	}
	if err != nil {
		if len(tags.All()) == 0 && result.VideoConfig == (videometa.VideoConfig{}) {
			videoMetadata.TimestampSource = videoTimestampSourceFallback
			videoMetadata.TimestampFallbackReason = videoTimestampFallbackDecodeError
			if len(videoMetadata.Warnings) == 0 {
				videoMetadata.Warnings = append(videoMetadata.Warnings, "videometa returned no tags or config for supported container")
			}
		} else {
			videoMetadata.TimestampSource = videoTimestampSourceFallback
			videoMetadata.TimestampFallbackReason = videoTimestampFallbackNoDateTime
		}
		return mediaMetadata{
			CreationDateTime: fallbackTime,
			VideoMetadata:    videoMetadata,
		}, nil
	}

	if found && !provenanceTimestamp.Equal(timestamp) {
		videoMetadata.Warnings = append(videoMetadata.Warnings, "timestamp provenance did not exactly match videometa selection; using videometa result")
	}
	if found {
		videoMetadata.TimestampSource = source
		videoMetadata.TimestampTag = tag
		videoMetadata.TimestampNamespace = namespace
		videoMetadata.TimestampFallbackReason = ""
	} else {
		videoMetadata.Warnings = append(videoMetadata.Warnings, "videometa returned a timestamp but provenance could not be resolved locally")
	}

	videoMetadata.ChosenTimestamp = timestamp

	return mediaMetadata{
		CreationDateTime: timestamp,
		VideoMetadata:    videoMetadata,
	}, nil
}

func findFirstStringValue(sourceTags []map[string]videometa.TagInfo, keys ...string) string {
	for _, source := range sourceTags {
		if tag, found := firstVideoTagInfo(source, keys...); found {
			value := strings.TrimSpace(fmt.Sprint(tag.Value))
			if value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return ""
}

func firstVideoTagInfo(sourceTags map[string]videometa.TagInfo, keys ...string) (videometa.TagInfo, bool) {
	for _, key := range keys {
		if tag, ok := sourceTags[key]; ok {
			return tag, true
		}
	}
	return videometa.TagInfo{}, false
}

func resolveVideoTimestampProvenance(tags videometa.Tags) (time.Time, string, string, string, bool) {
	candidates := []struct {
		sourceName string
		sourceTags map[string]videometa.TagInfo
		keys       []string
	}{
		{
			sourceName: videoTimestampSourceQuickTime,
			sourceTags: tags.QuickTime(),
			keys:       []string{"CreationDate", "CreateDate", "ModifyDate"},
		},
		{
			sourceName: videoTimestampSourceVendor,
			sourceTags: tags.MakerNotes(),
			keys:       []string{"CreationDate", "CreateDate", "ModifyDate", "DateTimeOriginal"},
		},
		{
			sourceName: videoTimestampSourceVendor,
			sourceTags: tags.XML(),
			keys:       []string{"CreationDate", "CreateDate", "ModifyDate", "DateTimeOriginal"},
		},
	}

	for _, candidate := range candidates {
		tag, found := firstVideoTagInfo(candidate.sourceTags, candidate.keys...)
		if !found {
			continue
		}
		t, err := parseVideoMetadataTimeValue(tag.Value)
		if err == nil {
			return t, candidate.sourceName, tag.Tag, tag.Namespace, true
		}
	}

	return time.Time{}, "", "", "", false
}

func parseVideoMetadataTimeValue(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return time.Time{}, fmt.Errorf("zero time")
		}
		return t, nil
	case string:
		return parseVideoMetadataTimeString(t)
	default:
		return time.Time{}, fmt.Errorf("unsupported type %T for date/time", v)
	}
}

func parseVideoMetadataTimeString(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0000:00:00 00:00:00" {
		return time.Time{}, fmt.Errorf("empty or zero date")
	}

	formats := []string{
		"2006:01:02 15:04:05",
		"2006:01:02 15:04:05-07:00",
		"2006:01:02 15:04:05Z07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006-01-02",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}
