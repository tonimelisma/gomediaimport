package main

import (
	"path/filepath"
	"strings"
)

type FileType string

const (
	// Processed Picture Types
	JPEG     FileType = "jpeg"
	JPEG2000 FileType = "jpeg2000"
	JPEGXL   FileType = "jpegxl"
	PNG      FileType = "png"
	GIF      FileType = "gif"
	BMP      FileType = "bmp"
	TIFF     FileType = "tiff"
	PSD      FileType = "psd"
	EPS      FileType = "eps"
	SVG      FileType = "svg"
	ICO      FileType = "ico"
	WEBP     FileType = "webp"
	HEIF     FileType = "heif"

	// Raw Picture Types
	RAW FileType = "raw"

	// Video Types
	MP4     FileType = "mp4"
	AVI     FileType = "avi"
	MOV     FileType = "mov"
	WMV     FileType = "wmv"
	FLV     FileType = "flv"
	MKV     FileType = "mkv"
	WEBM    FileType = "webm"
	OGV     FileType = "ogv"
	M4V     FileType = "m4v"
	THREEGP FileType = "3gp"
	THREEG2 FileType = "3g2"
	ASF     FileType = "asf"
	VOB     FileType = "vob"
	MTS     FileType = "mts"

	// Raw Video Types
	RAWVIDEO FileType = "rawvideo"
)

type MediaCategory string

const (
	ProcessedPicture MediaCategory = "processed_picture"
	RawPicture       MediaCategory = "raw_picture"
	Video            MediaCategory = "video"
	RawVideo         MediaCategory = "raw_video"
)

var fileExtensionToFileType = map[string]FileType{
	// Processed Picture Types
	"jpg": JPEG, "jpeg": JPEG, "jpe": JPEG, "jif": JPEG, "jfif": JPEG, "jfi": JPEG,
	"jp2": JPEG2000, "j2k": JPEG2000, "jpf": JPEG2000, "jpm": JPEG2000, "jpg2": JPEG2000, "j2c": JPEG2000, "jpc": JPEG2000, "jpx": JPEG2000, "mj2": JPEG2000,
	"jxl":  JPEGXL,
	"png":  PNG,
	"gif":  GIF,
	"bmp":  BMP,
	"tiff": TIFF, "tif": TIFF,
	"psd":  PSD,
	"eps":  EPS,
	"svg":  SVG,
	"ico":  ICO,
	"webp": WEBP,
	"heif": HEIF, "heifs": HEIF, "heic": HEIF, "heics": HEIF, "avci": HEIF, "avcs": HEIF, "hif": HEIF,

	// Raw Picture Types
	"arw": RAW, "cr2": RAW, "cr3": RAW, "crw": RAW, "dng": RAW, "erf": RAW, "kdc": RAW, "mrw": RAW,
	"nef": RAW, "orf": RAW, "pef": RAW, "raf": RAW, "raw": RAW, "rw2": RAW, "sr2": RAW, "srf": RAW, "x3f": RAW,

	// Video Types
	"mp4":  MP4,
	"avi":  AVI,
	"mov":  MOV,
	"wmv":  WMV,
	"flv":  FLV,
	"mkv":  MKV,
	"webm": WEBM,
	"ogv":  OGV,
	"m4v":  M4V,
	"3gp":  THREEGP,
	"3g2":  THREEG2,
	"asf":  ASF,
	"vob":  VOB,
	"mts":  MTS, "m2ts": MTS,

	// Raw Video Types
	"braw": RAWVIDEO, "r3d": RAWVIDEO, "ari": RAWVIDEO,
}

var fileTypeToMediaCategory = map[FileType]MediaCategory{
	// Processed Picture Types
	JPEG:     ProcessedPicture,
	JPEG2000: ProcessedPicture,
	JPEGXL:   ProcessedPicture,
	PNG:      ProcessedPicture,
	GIF:      ProcessedPicture,
	BMP:      ProcessedPicture,
	TIFF:     ProcessedPicture,
	PSD:      ProcessedPicture,
	EPS:      ProcessedPicture,
	SVG:      ProcessedPicture,
	ICO:      ProcessedPicture,
	WEBP:     ProcessedPicture,
	HEIF:     ProcessedPicture,

	// Raw Picture Types
	RAW: RawPicture,

	// Video Types
	MP4:     Video,
	AVI:     Video,
	MOV:     Video,
	WMV:     Video,
	FLV:     Video,
	MKV:     Video,
	WEBM:    Video,
	OGV:     Video,
	M4V:     Video,
	THREEGP: Video,
	THREEG2: Video,
	ASF:     Video,
	VOB:     Video,
	MTS:     Video,

	// Raw Video Types
	RAWVIDEO: RawVideo,
}

func getMediaTypeInfo(fi FileInfo) (MediaCategory, FileType) {
	ext := strings.ToLower(filepath.Ext(fi.SourceName))
	if ext == "" {
		return "", ""
	}

	fileType, ok := fileExtensionToFileType[ext[1:]] // Remove the leading dot
	if !ok {
		return "", ""
	}

	category, ok := fileTypeToMediaCategory[fileType]
	if !ok {
		return "", ""
	}

	return category, fileType
}
