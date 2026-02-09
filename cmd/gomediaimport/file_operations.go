package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

// enumerateFiles scans the source directory and returns a list of FileInfo structs
func enumerateFiles(sourceDir string, cfg config) ([]FileInfo, error) {
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
	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				fmt.Printf("Warning: Permission denied accessing %s, skipping...\n", path)
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return fmt.Errorf("error accessing path %q: %w", path, err)
		}

		// Skip symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		// Skip directories and files containing "THMBNL" if skipThumbnails is true
		if cfg.SkipThumbnails && strings.Contains(path, "THMBNL") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("error getting info for %q: %w", path, err)
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
			// Check if it's a sidecar file
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext != "" {
				ext = ext[1:] // Remove leading dot
			}
			if isSidecarExtension(ext) {
				action := getSidecarAction(ext, cfg.Sidecars, cfg.SidecarDefault)
				if action == SidecarIgnore {
					return nil
				}
				fileInfo.MediaCategory = Sidecar
				files = append(files, fileInfo)
				return nil
			}
			// Skip non-media, non-sidecar files
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

// fileSizeTime is a composite key for the duplicate detection index
type fileSizeTime struct {
	Size      int64
	Timestamp time.Time
}

func setFinalDestinationFilename(files *[]FileInfo, currentIndex int, initialFilename string, cfg config, sizeTimeIndex map[fileSizeTime][]int) error {
	file := &(*files)[currentIndex]
	baseDir := file.DestDir
	ext := filepath.Ext(initialFilename)
	baseFilename := strings.TrimSuffix(initialFilename, ext)

	if cfg.RenameByDateTime {
		newExt := getFirstExtensionForFileType(file.FileType)
		if newExt != "" {
			ext = "." + newExt
		}
	}

	initialFilename = baseFilename + ext

	if isDuplicateInPreviousFiles(files, currentIndex, cfg.ChecksumDuplicates, sizeTimeIndex) {
		file.Status = StatusPreExisting
		file.DestName = initialFilename
		return nil
	}

	fullPath := filepath.Join(baseDir, initialFilename)
	fileExists, err := exists(fullPath)
	if err != nil {
		return fmt.Errorf("error checking file %s: %w", fullPath, err)
	}
	if !fileExists && !isNameTakenByPreviousFile(files, currentIndex, initialFilename) {
		file.DestName = initialFilename
		return nil
	}

	dup, err := isDuplicate(file, fullPath, cfg.ChecksumDuplicates)
	if err != nil {
		return err
	}
	if dup {
		file.Status = StatusPreExisting
		file.DestName = initialFilename
		return nil
	}

	for i := 1; i <= 999999; i++ {
		suffix := fmt.Sprintf("_%03d", i) // Ensure three-digit suffix
		newFilename := baseFilename + suffix + ext
		fullPath = filepath.Join(baseDir, newFilename)
		fileExists, err = exists(fullPath)
		if err != nil {
			return fmt.Errorf("error checking file %s: %w", fullPath, err)
		}
		if !fileExists && !isNameTakenByPreviousFile(files, currentIndex, newFilename) {
			file.DestName = newFilename
			return nil
		}
		dup, err = isDuplicate(file, fullPath, cfg.ChecksumDuplicates)
		if err != nil {
			return err
		}
		if dup {
			file.Status = StatusPreExisting
			file.DestName = newFilename
			return nil
		}
	}

	return fmt.Errorf("couldn't find a unique filename after 999,999 attempts")
}

func isDuplicateInPreviousFiles(files *[]FileInfo, currentIndex int, checksumDuplicates bool, sizeTimeIndex map[fileSizeTime][]int) bool {
	currentFile := &(*files)[currentIndex]
	key := fileSizeTime{Size: currentFile.Size, Timestamp: currentFile.CreationDateTime}

	indices, ok := sizeTimeIndex[key]
	if !ok {
		return false
	}

	if !checksumDuplicates {
		return true
	}

	// Calculate current file checksum if needed
	if currentFile.SourceChecksum == "" {
		checksum, err := calculateXXHash(filepath.Join(currentFile.SourceDir, currentFile.SourceName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to calculate checksum for %s: %v\n", filepath.Join(currentFile.SourceDir, currentFile.SourceName), err)
			return false
		}
		currentFile.SourceChecksum = checksum
	}

	for _, i := range indices {
		previousFile := &(*files)[i]
		if previousFile.SourceChecksum == "" {
			checksum, err := calculateXXHash(filepath.Join(previousFile.SourceDir, previousFile.SourceName))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to calculate checksum for %s: %v\n", filepath.Join(previousFile.SourceDir, previousFile.SourceName), err)
				continue
			}
			previousFile.SourceChecksum = checksum
		}

		if currentFile.SourceChecksum == previousFile.SourceChecksum {
			return true
		}
	}

	return false
}

func isNameTakenByPreviousFile(files *[]FileInfo, currentIndex int, proposedName string) bool {
	for i := 0; i < currentIndex; i++ {
		if (*files)[i].DestDir == (*files)[currentIndex].DestDir && (*files)[i].DestName == proposedName {
			return true
		}
	}
	return false
}

func exists(destPath string) (bool, error) {
	_, err := os.Stat(destPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func isDuplicate(file *FileInfo, destPath string, checksumDuplicates bool) (bool, error) {
	destInfo, err := os.Stat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("error checking destination file %s: %w", destPath, err)
	}

	if destInfo.Size() != file.Size {
		return false, nil
	}

	if checksumDuplicates {
		srcChecksum := file.SourceChecksum
		if srcChecksum == "" {
			srcChecksum, err = calculateXXHash(filepath.Join(file.SourceDir, file.SourceName))
			if err != nil {
				return false, fmt.Errorf("failed to calculate checksum for %s: %w", filepath.Join(file.SourceDir, file.SourceName), err)
			}
			file.SourceChecksum = srcChecksum
		}

		destChecksum, err := calculateXXHash(destPath)
		if err != nil {
			return false, fmt.Errorf("failed to calculate checksum for %s: %w", destPath, err)
		}

		return srcChecksum == destChecksum, nil
	}

	return true, nil
}

func calculateXXHash(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%016x", hash.Sum64()), nil
}

func setFileTimes(path string, modTime time.Time) error {
	// Set modification time only
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		return fmt.Errorf("failed to set modification time: %w", err)
	}
	return nil
}

// ejectDriveMacOS attempts to eject the specified drive on macOS.
// It prints messages to stdout regarding its progress.
func ejectDriveMacOS(sourceDir string) error {
	fmt.Printf("Attempting to eject drive: %s\n", sourceDir)

	cmd := exec.Command("diskutil", "eject", sourceDir)
	output, err := cmd.CombinedOutput() // Using CombinedOutput to capture stderr as well

	if err != nil {
		return fmt.Errorf("failed to eject drive %s: %v. Output: %s", sourceDir, err, string(output))
	}

	fmt.Printf("Successfully ejected drive: %s\nOutput: %s\n", sourceDir, string(output))
	return nil
}
