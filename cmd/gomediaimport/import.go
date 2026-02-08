package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FileStatus represents the status of a file during import
type FileStatus string

const (
	StatusCopied                  FileStatus = "copied"
	StatusPreExisting             FileStatus = "pre-existing"
	StatusFailed                  FileStatus = "failed"
	StatusUnnamable               FileStatus = "unnamable"
	StatusDirectoryCreationFailed FileStatus = "directory creation failed"
	StatusSidecarDeleted          FileStatus = "sidecar deleted"
)

// FileInfo represents information about each file being imported
type FileInfo struct {
	SourceName       string
	SourceDir        string
	DestName         string
	DestDir          string
	SourceChecksum   string
	CreationDateTime time.Time
	Size             int64
	MediaCategory    MediaCategory
	FileType         FileType
	Status           FileStatus
	ParentIndex      int // Index of parent media file for sidecars, -1 if N/A
}

// importMedia handles the main functionality of the program
func importMedia(cfg config) error {
	if cfg.Verbose {
		fmt.Println("Source directory:", cfg.SourceDir)
		fmt.Println("Destination directory:", cfg.DestDir)
		fmt.Println("Organize by date:", cfg.OrganizeByDate)
		fmt.Println("Rename by date and time:", cfg.RenameByDateTime)
		fmt.Println("Checksum duplicates:", cfg.ChecksumDuplicates)
		fmt.Println("Skip thumbnails:", cfg.SkipThumbnails)
		fmt.Println("Delete originals:", cfg.DeleteOriginals)
		fmt.Println("Sidecar default:", cfg.SidecarDefault)
	}

	// Enumerate files in the source directory
	files, err := enumerateFiles(cfg.SourceDir, cfg)
	if err != nil {
		return fmt.Errorf("failed to enumerate files: %w", err)
	}

	// Initialize ParentIndex to -1 for all files
	for i := range files {
		files[i].ParentIndex = -1
	}

	// Print the number of files enumerated
	if cfg.Verbose {
		fmt.Printf("Number of files enumerated: %d\n", len(files))
	}

	// Pass 1: Process non-sidecar files
	sizeTimeIndex := make(map[fileSizeTime][]int)
	for i := range files {
		if files[i].MediaCategory == Sidecar {
			continue
		}

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
		if err := setFinalDestinationFilename(&files, i, initialFilename, cfg, sizeTimeIndex); err != nil {
			files[i].Status = StatusUnnamable
			continue
		}

		// Add to index for subsequent duplicate detection
		key := fileSizeTime{Size: files[i].Size, Timestamp: files[i].CreationDateTime}
		sizeTimeIndex[key] = append(sizeTimeIndex[key], i)
	}

	// Build parent index: map (sourceDir, lowerBaseName) → first media file index
	type parentKey struct {
		dir      string
		baseName string
	}
	parentIndex := make(map[parentKey]int)
	for i := range files {
		if files[i].MediaCategory == Sidecar {
			continue
		}
		ext := filepath.Ext(files[i].SourceName)
		base := strings.TrimSuffix(files[i].SourceName, ext)
		key := parentKey{dir: files[i].SourceDir, baseName: strings.ToLower(base)}
		if _, exists := parentIndex[key]; !exists {
			parentIndex[key] = i
		}
	}

	// Pass 2: Process sidecar files
	for i := range files {
		if files[i].MediaCategory != Sidecar {
			continue
		}

		sidecarExt := strings.ToLower(filepath.Ext(files[i].SourceName))
		if sidecarExt != "" {
			sidecarExt = sidecarExt[1:]
		}
		action := getSidecarAction(sidecarExt, cfg.Sidecars, cfg.SidecarDefault)

		if action == SidecarDelete {
			files[i].Status = StatusSidecarDeleted
			continue
		}

		// action == SidecarCopy — find parent
		ext := filepath.Ext(files[i].SourceName)
		base := strings.TrimSuffix(files[i].SourceName, ext)
		key := parentKey{dir: files[i].SourceDir, baseName: strings.ToLower(base)}

		if pi, ok := parentIndex[key]; ok {
			// Parent found — derive destination from parent
			files[i].ParentIndex = pi
			parentFile := files[pi]
			files[i].DestDir = parentFile.DestDir
			parentDestExt := filepath.Ext(parentFile.DestName)
			parentDestBase := strings.TrimSuffix(parentFile.DestName, parentDestExt)
			files[i].DestName = parentDestBase + ext
		} else {
			// Orphaned sidecar — plan destination independently using own mtime
			if cfg.OrganizeByDate {
				files[i].DestDir = filepath.Join(cfg.DestDir, files[i].CreationDateTime.Format("2006/01"))
			} else {
				files[i].DestDir = cfg.DestDir
			}
			if cfg.RenameByDateTime {
				files[i].DestName = files[i].CreationDateTime.Format("20060102_150405") + ext
			} else {
				files[i].DestName = files[i].SourceName
			}
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
		var preExisting, failed, copied, sidecarDeleted, total int
		for _, file := range files {
			total++
			switch file.Status {
			case StatusPreExisting:
				preExisting++
			case StatusFailed:
				failed++
			case StatusCopied:
				copied++
			case StatusSidecarDeleted:
				sidecarDeleted++
			}
		}
		fmt.Printf("\nFile status summary:\n")
		fmt.Printf("Total files: %d\n", total)
		fmt.Printf("Pre-existing: %d\n", preExisting)
		fmt.Printf("Failed: %d\n", failed)
		fmt.Printf("Copied: %d\n", copied)
		if sidecarDeleted > 0 {
			fmt.Printf("Sidecars marked for deletion: %d\n", sidecarDeleted)
		}
	}

	// If we've reached this point, all main import operations were successful.
	if cfg.AutoEjectMacOS && runtime.GOOS == "darwin" {
		fmt.Printf("INFO: Import operations completed successfully. Attempting auto-eject for %s.\n", cfg.SourceDir)
		ejectErr := ejectDriveMacOS(cfg.SourceDir)
		if ejectErr != nil {
			// Log the error but do not change the function's success return,
			// as the import itself was successful.
			fmt.Printf("WARNING: Failed to eject source drive %s after successful import: %v\n", cfg.SourceDir, ejectErr)
		} else {
			fmt.Printf("INFO: Successfully ejected source drive %s after successful import.\n", cfg.SourceDir)
		}
	}
	// The function will then return nil, indicating overall success of importMedia.
	return nil
}

func copyFiles(files []FileInfo, cfg config) error {
	var totalSize int64
	for _, file := range files {
		if file.Status != StatusUnnamable && file.Status != StatusPreExisting && file.Status != StatusSidecarDeleted {
			totalSize += file.Size
		}
	}

	if cfg.Verbose {
		fmt.Printf("Total size to copy: %s\n", humanReadableSize(totalSize))
	}

	var copiedSize int64
	startTime := time.Now()

	for i := range files {
		if files[i].Status == StatusUnnamable || files[i].Status == StatusPreExisting || files[i].Status == StatusSidecarDeleted {
			continue
		}

		srcPath := filepath.Join(files[i].SourceDir, files[i].SourceName)
		destPath := filepath.Join(files[i].DestDir, files[i].DestName)

		// Create destination directory if it doesn't exist
		if !cfg.DryRun {
			if err := os.MkdirAll(files[i].DestDir, 0755); err != nil {
				files[i].Status = StatusDirectoryCreationFailed
				return fmt.Errorf("failed to create directory %s: %w", files[i].DestDir, err)
			}

			// Copy the file
			if err := copyFile(srcPath, destPath); err != nil {
				os.Remove(destPath)
				files[i].Status = StatusFailed
			} else {
				// Set file times
				if err := setFileTimes(destPath, files[i].CreationDateTime); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to set file times for %s: %v\n", destPath, err)
				}
				files[i].Status = StatusCopied
			}
		}

		copiedSize += files[i].Size

		if cfg.Verbose {
			if totalSize > 0 {
				progress := float64(copiedSize) / float64(totalSize)
				elapsed := time.Since(startTime)
				var remaining time.Duration
				if progress > 0 {
					estimatedTotal := time.Duration(float64(elapsed) / progress)
					remaining = estimatedTotal - elapsed
				}

				fmt.Printf("%s -> %s (%s/%s, %d%%, %s remaining)\n",
					srcPath,
					destPath,
					humanReadableSize(copiedSize),
					humanReadableSize(totalSize),
					int(progress*100),
					humanReadableDuration(remaining),
				)
			} else {
				fmt.Printf("%s -> %s\n", srcPath, destPath)
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	written, err := io.Copy(destFile, sourceFile)
	if err != nil {
		destFile.Close()
		return err
	}

	if written != sourceInfo.Size() {
		destFile.Close()
		return fmt.Errorf("incomplete copy: wrote %d of %d bytes", written, sourceInfo.Size())
	}

	if err := destFile.Sync(); err != nil {
		destFile.Close()
		return err
	}

	return destFile.Close()
}

func deleteOriginalFiles(files []FileInfo, cfg config) error {
	if !cfg.DeleteOriginals {
		return nil
	}

	var deletedCount int
	var deletedSize int64

	for _, file := range files {
		if file.Status == StatusCopied || file.Status == StatusPreExisting || file.Status == StatusSidecarDeleted {
			sourcePath := filepath.Join(file.SourceDir, file.SourceName)
			if !cfg.DryRun {
				err := os.Remove(sourcePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", sourcePath, err)
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
