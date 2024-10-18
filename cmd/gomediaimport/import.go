package main

import (
	"fmt"
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
	if cfg.Verbose {
		fmt.Println("Source directory:", cfg.SourceDir)
		fmt.Println("Destination directory:", cfg.DestDir)
		fmt.Println("Organize by date:", cfg.OrganizeByDate)
		fmt.Println("Rename by date and time:", cfg.RenameByDateTime)
		fmt.Println("Auto rename unique files:", cfg.AutoRenameUnique)
		fmt.Println("Skip thumbnails:", cfg.SkipThumbnails)
	}

	// Enumerate files in the source directory
	files, err := enumerateFiles(cfg.SourceDir, cfg.SkipThumbnails)
	if err != nil {
		return fmt.Errorf("failed to enumerate files: %w", err)
	}

	// Print the number of files enumerated
	if cfg.Verbose {
		fmt.Printf("Number of files enumerated: %d\n", len(files))
	}

	// Process each file
	for i := range files {
		// Set destination directory
		if cfg.OrganizeByDate {
			files[i].DestDir = filepath.Join(cfg.DestDir, files[i].CreationDateTime.Format("2006/01"))
		} else {
			files[i].DestDir = cfg.DestDir
		}

		// Determine initial filename
		var initialFilename string
		if cfg.RenameByDateTime {
			initialFilename = files[i].CreationDateTime.Format("20060102_150405") + filepath.Ext(files[i].SourceName)
		} else {
			initialFilename = files[i].SourceName
		}

		// Set final destination filename
		if err := setFinalDestinationFilename(&files[i], initialFilename, cfg); err != nil {
			files[i].Status = "unnamable"
			continue
		}
	}

	//fmt.Println(files[len(files)-1])
	// TODO: Implement the actual media import logic here using the 'files' slice

	return nil
}
