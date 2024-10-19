package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// enumerateFiles scans the source directory and returns a list of FileInfo structs
func enumerateFiles(sourceDir string, skipThumbnails bool) ([]FileInfo, error) {
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

		// Skip .Spotlight-V100 and .fseventsd folders
		if info.IsDir() && (info.Name() == ".Spotlight-V100" || info.Name() == ".fseventsd") {
			return filepath.SkipDir
		}

		// Skip directories and files containing "THMBNL" if skipThumbnails is true
		if skipThumbnails && strings.Contains(path, "THMBNL") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

func setFinalDestinationFilename(files *[]FileInfo, currentIndex int, initialFilename string, cfg config) error {
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

	if isDuplicateInPreviousFiles(files, currentIndex, cfg.ChecksumDuplicates) {
		file.Status = "pre-existing"
		file.DestName = initialFilename
		return nil
	}

	fullPath := filepath.Join(baseDir, initialFilename)
	if !exists(fullPath) && !isNameTakenByPreviousFile(files, currentIndex, initialFilename) {
		file.DestName = initialFilename
		return nil
	}

	if isDuplicate(file, fullPath, cfg.ChecksumDuplicates) {
		file.Status = "pre-existing"
		file.DestName = initialFilename
		return nil
	}

	for i := 1; i <= 999999; i++ {
		suffix := fmt.Sprintf("_%d", i)
		newFilename := baseFilename + suffix + ext
		fullPath = filepath.Join(baseDir, newFilename)
		if !exists(fullPath) && !isNameTakenByPreviousFile(files, currentIndex, newFilename) {
			file.DestName = newFilename
			return nil
		}
		if isDuplicate(file, fullPath, cfg.ChecksumDuplicates) {
			file.Status = "pre-existing"
			file.DestName = newFilename
			return nil
		}
	}

	return fmt.Errorf("couldn't find a unique filename after 999,999 attempts")
}

func isDuplicateInPreviousFiles(files *[]FileInfo, currentIndex int, checksumDuplicates bool) bool {
	currentFile := &(*files)[currentIndex]

	for i := 0; i < currentIndex; i++ {
		previousFile := &(*files)[i]

		if currentFile.CreationDateTime == previousFile.CreationDateTime && currentFile.Size == previousFile.Size {
			if !checksumDuplicates {
				return true
			}

			// Calculate and store checksums if needed
			if currentFile.SourceChecksum == "" {
				checksum, err := calculateCRC32(filepath.Join(currentFile.SourceDir, currentFile.SourceName))
				if err != nil {
					// Handle error (e.g., log it)
					return false
				}
				currentFile.SourceChecksum = checksum
			}

			if previousFile.SourceChecksum == "" {
				checksum, err := calculateCRC32(filepath.Join(previousFile.SourceDir, previousFile.SourceName))
				if err != nil {
					// Handle error (e.g., log it)
					return false
				}
				previousFile.SourceChecksum = checksum
			}

			if currentFile.SourceChecksum == previousFile.SourceChecksum {
				return true
			}
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

func exists(destPath string) bool {
	_, err := os.Stat(destPath)
	return !os.IsNotExist(err)
}

func isDuplicate(file *FileInfo, destPath string, checksumDuplicates bool) bool {
	destInfo, err := os.Stat(destPath)
	if os.IsNotExist(err) {
		return false
	}

	if destInfo.Size() != file.Size {
		return false
	}

	if checksumDuplicates {
		srcChecksum, err := calculateCRC32(filepath.Join(file.SourceDir, file.SourceName))
		if err != nil {
			// Handle error (e.g., log it)
			return false
		}
		file.SourceChecksum = srcChecksum

		destChecksum, err := calculateCRC32(destPath)
		if err != nil {
			// Handle error (e.g., log it)
			return false
		}

		return srcChecksum == destChecksum
	}

	return true
}

func calculateCRC32(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%08x", hash.Sum32()), nil
}
