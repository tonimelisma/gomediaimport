package main

import (
	"fmt"
	"hash/crc32"
	"io"
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

func setDestinationFilename(file *FileInfo, cfg config) error {
	baseDir := file.DestDir
	baseFilename := file.CreationDateTime.Format("20060102_150405")
	ext := getFirstExtensionForFileType(file.FileType)

	for i := 0; i <= 999; i++ {
		suffix := fmt.Sprintf("%03d", i)
		file.DestName = baseFilename + suffix + "." + ext
		fullPath := filepath.Join(baseDir, file.DestName)

		if exists(fullPath) {
			if isDuplicate(file, fullPath, cfg.AutoRenameUnique) {
				file.Status = "pre-existing"
				return nil
			} else {
				continue
			}
		} else {
			// Found a non-duplicate filename
			return nil
		}
	}

	return fmt.Errorf("couldn't find a unique filename after 1000 attempts")
}

func exists(destPath string) bool {
	_, err := os.Stat(destPath)
	return !os.IsNotExist(err)
}

func isDuplicate(file *FileInfo, destPath string, autoRenameUnique bool) bool {
	destInfo, err := os.Stat(destPath)
	if os.IsNotExist(err) {
		return false
	}

	if destInfo.Size() != file.Size {
		return false
	}

	if autoRenameUnique {
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

		if srcChecksum == destChecksum {
			return true
		}
	} else {
		return true
	}

	return false
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
