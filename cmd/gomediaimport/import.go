package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileInfo represents information about each file being imported
type FileInfo struct {
	SourceName       string
	SourceDir        string
	DestName         string
	DestDir          string
	SourceChecksum   string
	DestChecksum     string
	CreationDateTime time.Time
	Size             int64
	MediaCategory    MediaCategory
	FileType         FileType
	Status           string
}

// importMedia handles the main functionality of the program
func importMedia(cfg config) error {
	fmt.Println("Source directory:", cfg.SourceDir)
	fmt.Println("Destination directory:", cfg.DestDir)
	fmt.Println("Organize by date:", cfg.OrganizeByDate)
	fmt.Println("Rename by date and time:", cfg.RenameByDateTime)
	fmt.Println("Auto rename unique files:", cfg.AutoRenameUnique)

	// Enumerate files in the source directory
	files, err := enumerateFiles(cfg.SourceDir)
	if err != nil {
		return fmt.Errorf("failed to enumerate files: %w", err)
	}

	// Print the number of files enumerated
	fmt.Printf("Number of files enumerated: %d\n", len(files))

	// TODO: Implement the actual media import logic here using the 'files' slice

	return nil
}

// enumerateFiles scans the source directory and returns a list of FileInfo structs
func enumerateFiles(sourceDir string) ([]FileInfo, error) {
	var files []FileInfo

	// Check if the source directory exists
	_, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source directory does not exist: %w", err)
		}
		return nil, fmt.Errorf("error accessing source directory: %w", err)
	}

	// Walk through the directory
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %w", path, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Create FileInfo struct for each file
		fileInfo := FileInfo{
			SourceName:       info.Name(),
			SourceDir:        filepath.Dir(path),
			Size:             info.Size(),
			CreationDateTime: info.ModTime(), // Using ModTime as default CreationDateTime
		}

		// Get media type information
		category, fileType := getMediaTypeInfo(fileInfo)
		if category == "" {
			// Skip non-media files
			return nil
		}

		fileInfo.MediaCategory = category
		fileInfo.FileType = fileType

		// Extract creation date and time from metadata
		extractedDateTime, err := extractCreationDateTimeFromMetadata(fileInfo)
		if err == nil {
			fileInfo.CreationDateTime = extractedDateTime
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %w", sourceDir, err)
	}

	return files, nil
}
