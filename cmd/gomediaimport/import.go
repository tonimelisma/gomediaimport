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

	// Process each file
	for i := range files {
		// Set destination directory
		if cfg.OrganizeByDate {
			files[i].DestDir = filepath.Join(cfg.DestDir, files[i].CreationDateTime.Format("2006/01"))
		} else {
			files[i].DestDir = cfg.DestDir
		}

		// Set destination filename and check for duplicates
		if cfg.RenameByDateTime {
			if err := setDestinationFilename(&files[i], cfg); err != nil {
				files[i].Status = "unnamable"
				continue
			}
		} else {
			files[i].DestName = files[i].SourceName
			fullDestPath := filepath.Join(files[i].DestDir, files[i].DestName)
			if isDuplicate(&files[i], fullDestPath, cfg.AutoRenameUnique) {
				files[i].Status = "pre-existing"
				continue
			}
		}
	}

	// TODO: Implement the actual media import logic here using the 'files' slice

	return nil
}
