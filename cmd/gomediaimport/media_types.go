package main

import (
	"path/filepath"
	"strings"
)

type FileType string
type MediaCategory string

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

	// Media Categories
	ProcessedPicture MediaCategory = "processed_picture"
	RawPicture       MediaCategory = "raw_picture"
	Video            MediaCategory = "video"
	RawVideo         MediaCategory = "raw_video"
	Sidecar          MediaCategory = "sidecar"
)

// SidecarAction defines how a sidecar file type should be handled
type SidecarAction string

const (
	SidecarIgnore SidecarAction = "ignore"
	SidecarCopy   SidecarAction = "copy"
	SidecarDelete SidecarAction = "delete"
)

// sidecarDefaults maps sidecar extensions to their default actions
var sidecarDefaults = map[string]SidecarAction{
	"thm": SidecarDelete,
	"ctg": SidecarDelete,
	"xmp": SidecarCopy,
	"aae": SidecarDelete,
	"lrf": SidecarDelete,
	"srt": SidecarCopy,
	"mpl": SidecarDelete,
	"cpi": SidecarDelete,
}

// isSidecarExtension returns true if the extension (without dot, lowercase) is a known sidecar type
func isSidecarExtension(ext string) bool {
	_, ok := sidecarDefaults[ext]
	return ok
}

// getSidecarAction returns the action for a sidecar extension, checking overrides first,
// then built-in defaults, then the global default
func getSidecarAction(ext string, sidecarOverrides map[string]SidecarAction, sidecarDefault SidecarAction) SidecarAction {
	if action, ok := sidecarOverrides[ext]; ok {
		return action
	}
	if action, ok := sidecarDefaults[ext]; ok {
		return action
	}
	return sidecarDefault
}

// isValidSidecarAction returns true if the action is a valid SidecarAction value
func isValidSidecarAction(action SidecarAction) bool {
	return action == SidecarIgnore || action == SidecarCopy || action == SidecarDelete
}

type FileTypeInfo struct {
	FileType      FileType
	MediaCategory MediaCategory
	Extensions    []string
}

var fileTypes = []FileTypeInfo{
	{JPEG, ProcessedPicture, []string{"jpg", "jpeg", "jpe", "jif", "jfif", "jfi"}},
	{JPEG2000, ProcessedPicture, []string{"jp2", "j2k", "jpf", "jpm", "jpg2", "j2c", "jpc", "jpx", "mj2"}},
	{JPEGXL, ProcessedPicture, []string{"jxl"}},
	{PNG, ProcessedPicture, []string{"png"}},
	{GIF, ProcessedPicture, []string{"gif"}},
	{BMP, ProcessedPicture, []string{"bmp"}},
	{TIFF, ProcessedPicture, []string{"tiff", "tif"}},
	{PSD, ProcessedPicture, []string{"psd"}},
	{EPS, ProcessedPicture, []string{"eps"}},
	{SVG, ProcessedPicture, []string{"svg"}},
	{ICO, ProcessedPicture, []string{"ico"}},
	{WEBP, ProcessedPicture, []string{"webp"}},
	{HEIF, ProcessedPicture, []string{"heif", "heifs", "heic", "heics", "avci", "avcs", "hif"}},
	{RAW, RawPicture, []string{"arw", "cr2", "cr3", "crw", "dng", "erf", "kdc", "mrw", "nef", "orf", "pef", "raf", "raw", "rw2", "sr2", "srf", "x3f"}},
	{MP4, Video, []string{"mp4"}},
	{AVI, Video, []string{"avi"}},
	{MOV, Video, []string{"mov"}},
	{WMV, Video, []string{"wmv"}},
	{FLV, Video, []string{"flv"}},
	{MKV, Video, []string{"mkv"}},
	{WEBM, Video, []string{"webm"}},
	{OGV, Video, []string{"ogv"}},
	{M4V, Video, []string{"m4v"}},
	{THREEGP, Video, []string{"3gp"}},
	{THREEG2, Video, []string{"3g2"}},
	{ASF, Video, []string{"asf"}},
	{VOB, Video, []string{"vob"}},
	{MTS, Video, []string{"mts", "m2ts"}},
	{RAWVIDEO, RawVideo, []string{"braw", "r3d", "ari"}},
}

func getMediaTypeInfo(fi FileInfo) (MediaCategory, FileType) {
	ext := strings.ToLower(filepath.Ext(fi.SourceName))
	if ext == "" {
		return "", ""
	}
	ext = ext[1:] // Remove the leading dot

	for _, ft := range fileTypes {
		for _, e := range ft.Extensions {
			if e == ext {
				return ft.MediaCategory, ft.FileType
			}
		}
	}

	return "", ""
}

func getFirstExtensionForFileType(fileType FileType) string {
	for _, ft := range fileTypes {
		if ft.FileType == fileType && len(ft.Extensions) > 0 {
			return ft.Extensions[0]
		}
	}
	return ""
}
