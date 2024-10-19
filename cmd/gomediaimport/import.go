package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
		fmt.Println("Checksum duplicates:", cfg.ChecksumDuplicates)
		fmt.Println("Checksum imports:", cfg.ChecksumImports)
		fmt.Println("Skip thumbnails:", cfg.SkipThumbnails)
		fmt.Println("Delete originals:", cfg.DeleteOriginals)
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
		if err := setFinalDestinationFilename(&files, i, initialFilename, cfg); err != nil {
			files[i].Status = "unnamable"
			continue
		}
	}

	// Copy files
	if err := copyFiles(files, cfg); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	// Delete original files if configured
	if err := deleteOriginalFiles(files, cfg); err != nil {
		return fmt.Errorf("failed to delete original files: %w", err)
	}

	// Enumerate file statuses if verbose
	if cfg.Verbose {
		var preExisting, failed, copied, total int
		for _, file := range files {
			total++
			switch file.Status {
			case "pre-existing":
				preExisting++
			case "failed":
				failed++
			case "copied":
				copied++
			}
		}
		fmt.Printf("\nFile status summary:\n")
		fmt.Printf("Total files: %d\n", total)
		fmt.Printf("Pre-existing: %d\n", preExisting)
		fmt.Printf("Failed: %d\n", failed)
		fmt.Printf("Copied: %d\n", copied)
	}

	return nil
}

func copyFiles(files []FileInfo, cfg config) error {
	var totalSize int64
	for _, file := range files {
		if file.Status != "unnamable" && file.Status != "pre-existing" {
			totalSize += file.Size
		}
	}

	if cfg.Verbose {
		fmt.Printf("Total size to copy: %s\n", humanReadableSize(totalSize))
	}

	var copiedSize int64
	startTime := time.Now()

	for i := range files {
		if files[i].Status == "unnamable" || files[i].Status == "pre-existing" {
			continue
		}

		// Create destination directory if it doesn't exist
		if !cfg.DryRun {
			if err := os.MkdirAll(files[i].DestDir, 0755); err != nil {
				files[i].Status = "directory creation failed"
				return fmt.Errorf("failed to create directory %s: %w", files[i].DestDir, err)
			}

			// Copy the file
			if err := copyFile(files[i].SourceDir+"/"+files[i].SourceName, files[i].DestDir+"/"+files[i].DestName); err != nil {
				files[i].Status = "failed"
			} else {
				files[i].Status = "copied"
			}
		}

		copiedSize += files[i].Size

		if cfg.Verbose {
			progress := float64(copiedSize) / float64(totalSize)
			elapsed := time.Since(startTime)
			estimatedTotal := time.Duration(float64(elapsed) / progress)
			remaining := estimatedTotal - elapsed

			fmt.Printf("%s -> %s (%s/%s, %.2f%%, %s/%s)\n",
				files[i].SourceDir+"/"+files[i].SourceName,
				files[i].DestDir+"/"+files[i].DestName,
				humanReadableSize(copiedSize),
				humanReadableSize(totalSize),
				progress*100,
				humanReadableDuration(remaining),
				humanReadableDuration(estimatedTotal),
			)
		}
	}

	return nil
}

func deleteOriginalFiles(files []FileInfo, cfg config) error {
	if !cfg.DeleteOriginals {
		return nil
	}

	var deletedCount int
	var deletedSize int64

	for _, file := range files {
		if file.Status == "copied" || file.Status == "pre-existing" {
			sourcePath := filepath.Join(file.SourceDir, file.SourceName)
			if !cfg.DryRun {
				err := os.Remove(sourcePath)
				if err != nil {
					if cfg.Verbose {
						fmt.Printf("Failed to delete %s: %v\n", sourcePath, err)
					}
					continue
				}
			}
			deletedCount++
			deletedSize += file.Size
			if cfg.Verbose {
				fmt.Printf("Deleted original file: %s\n", sourcePath)
			}
		}
	}

	if cfg.Verbose {
		fmt.Printf("\nOriginal files deleted: %d\n", deletedCount)
		fmt.Printf("Total size of deleted files: %s\n", humanReadableSize(deletedSize))
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func humanReadableSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func humanReadableDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, "")
}
