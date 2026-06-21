package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func writeTestArtifact(t *testing.T, root, relativePath string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func cleanupTargetKinds(targets []sourceCleanupTarget) []sourceArtifactKind {
	kinds := make([]sourceArtifactKind, len(targets))
	for i, target := range targets {
		kinds[i] = target.Kind
	}
	return kinds
}

func TestSourceArtifactPathClassification(t *testing.T) {
	trashTests := []struct {
		path string
		want bool
	}{
		{".Trashes", true},
		{".trashes", true},
		{".Trash", true},
		{".Trash-501", true},
		{".trash-1000", true},
		{"$RECYCLE.BIN", true},
		{"$recycle.bin", true},
		{".Trash-user", false},
		{"folder/.Trash", false},
		{"Trash", false},
	}
	for _, tt := range trashTests {
		t.Run("trash_"+strings.ReplaceAll(tt.path, "/", "_"), func(t *testing.T) {
			if got := isTopLevelTrashDir(filepath.FromSlash(tt.path)); got != tt.want {
				t.Fatalf("isTopLevelTrashDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}

	thumbnailTests := []struct {
		path string
		want bool
	}{
		{"M4ROOT/THMBNL", true},
		{"private/m4root/thmbnl", true},
		{"PRIVATE/M4ROOT/CLIP/THMBNL", false},
		{"holiday/THMBNL", false},
		{"THMBNL", false},
	}
	for _, tt := range thumbnailTests {
		if got := isSonyThumbnailDir(filepath.FromSlash(tt.path)); got != tt.want {
			t.Errorf("isSonyThumbnailDir(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}

	xmlTests := []struct {
		path string
		want bool
	}{
		{"M4ROOT/CLIP/C0001M01.XML", true},
		{"private/m4root/clip/custom.xml", true},
		{"PRIVATE/M4ROOT/SUB/proxy.xml", false},
		{"holiday/CLIP/notes.xml", false},
		{"M4ROOT/CLIP/nested/notes.xml", false},
	}
	for _, tt := range xmlTests {
		if got := isSonyClipXML(filepath.FromSlash(tt.path)); got != tt.want {
			t.Errorf("isSonyClipXML(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestEnumerateFilesSeparatesSourceArtifacts(t *testing.T) {
	sourceDir := t.TempDir()

	writeTestArtifact(t, sourceDir, "live.jpg")
	writeTestArtifact(t, sourceDir, ".Trashes/501/PRIVATE/M4ROOT/THMBNL/C0001T01.JPG")
	writeTestArtifact(t, sourceDir, ".Trash/files/old.jpg")
	writeTestArtifact(t, sourceDir, ".Trash-501/files/older.jpg")
	writeTestArtifact(t, sourceDir, "$recycle.bin/recycled.jpg")
	writeTestArtifact(t, sourceDir, "PRIVATE/M4ROOT/THMBNL/C0002T01.JPG")
	writeTestArtifact(t, sourceDir, "PRIVATE/M4ROOT/CLIP/C0002M01.XML")
	writeTestArtifact(t, sourceDir, "M4ROOT/THMBNL/C0003T01.JPG")
	writeTestArtifact(t, sourceDir, "M4ROOT/CLIP/C0003M01.xml")
	writeTestArtifact(t, sourceDir, "._live.jpg")
	writeTestArtifact(t, sourceDir, "holiday/THMBNL/real.jpg")
	writeTestArtifact(t, sourceDir, ".Trash-user/not-system-trash.jpg")

	result, err := enumerateFiles(sourceDir, config{SidecarDefault: SidecarDelete})
	if err != nil {
		t.Fatal(err)
	}

	var imported []string
	for _, file := range result.Files {
		relPath, err := filepath.Rel(sourceDir, filepath.Join(file.SourceDir, file.SourceName))
		if err != nil {
			t.Fatal(err)
		}
		imported = append(imported, filepath.ToSlash(relPath))
	}
	wantImported := []string{".Trash-user/not-system-trash.jpg", "holiday/THMBNL/real.jpg", "live.jpg"}
	if !reflect.DeepEqual(imported, wantImported) {
		t.Fatalf("imported files = %v, want %v", imported, wantImported)
	}

	wantKinds := []sourceArtifactKind{
		sourceArtifactTrash,
		sourceArtifactTrash,
		sourceArtifactTrash,
		sourceArtifactTrash,
		sourceArtifactAppleDouble,
		sourceArtifactSonyXML,
		sourceArtifactSonyThumbnail,
		sourceArtifactSonyXML,
		sourceArtifactSonyThumbnail,
	}
	if got := cleanupTargetKinds(result.CleanupTargets); !reflect.DeepEqual(got, wantKinds) {
		t.Fatalf("cleanup target kinds = %v, want %v", got, wantKinds)
	}

	for _, target := range result.CleanupTargets {
		if strings.Contains(target.Path, filepath.Join(".Trashes", "501")) && target.Kind != sourceArtifactTrash {
			t.Fatalf("artifact beneath .Trashes classified as %q instead of trash", target.Kind)
		}
	}
}

func TestImportMediaSourceArtifactLifecycle(t *testing.T) {
	for _, deleteOriginals := range []bool{false, true} {
		name := "preserve"
		if deleteOriginals {
			name = "delete"
		}
		t.Run(name, func(t *testing.T) {
			sourceDir := t.TempDir()
			destDir := t.TempDir()
			livePath := writeTestArtifact(t, sourceDir, "live.jpg")
			trashPath := writeTestArtifact(t, sourceDir, ".Trashes/501/old.jpg")
			thumbnailPath := writeTestArtifact(t, sourceDir, "PRIVATE/M4ROOT/THMBNL/C0001T01.JPG")
			xmlPath := writeTestArtifact(t, sourceDir, "PRIVATE/M4ROOT/CLIP/C0001M01.XML")
			appleDoublePath := writeTestArtifact(t, sourceDir, "._live.jpg")

			cfg := config{
				SourceDir:          sourceDir,
				DestDir:            destDir,
				DeleteOriginals:    deleteOriginals,
				CheckDiskSpace:     false,
				SidecarDefault:     SidecarDelete,
				ChecksumDuplicates: true,
				Workers:            1,
			}
			if err := importMedia(cfg); err != nil {
				t.Fatalf("importMedia failed: %v", err)
			}

			if _, err := os.Stat(filepath.Join(destDir, "live.jpg")); err != nil {
				t.Fatalf("live media was not imported: %v", err)
			}
			for _, path := range []string{livePath, trashPath, thumbnailPath, xmlPath, appleDoublePath} {
				_, err := os.Stat(path)
				if deleteOriginals && !os.IsNotExist(err) {
					t.Errorf("expected %s to be deleted, stat error: %v", path, err)
				}
				if !deleteOriginals && err != nil {
					t.Errorf("expected %s to be preserved, stat error: %v", path, err)
				}
			}
		})
	}
}

func TestImportMediaCopyFailurePreservesSourceArtifacts(t *testing.T) {
	sourceDir := t.TempDir()
	livePath := writeTestArtifact(t, sourceDir, "live.jpg")
	trashPath := writeTestArtifact(t, sourceDir, ".Trashes/501/old.jpg")
	destFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(destFile, []byte("file"), 0644); err != nil {
		t.Fatal(err)
	}

	err := importMedia(config{
		SourceDir:       sourceDir,
		DestDir:         destFile,
		DeleteOriginals: true,
		CheckDiskSpace:  false,
		SidecarDefault:  SidecarDelete,
		Workers:         1,
	})
	if err == nil {
		t.Fatal("expected copy failure")
	}
	for _, path := range []string{livePath, trashPath} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("source path %s should remain after copy failure: %v", path, statErr)
		}
	}
}

func TestImportMediaOriginalDeletionFailurePreservesSourceArtifacts(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	readOnlyDir := filepath.Join(sourceDir, "readonly")
	livePath := writeTestArtifact(t, readOnlyDir, "live.jpg")
	trashPath := writeTestArtifact(t, sourceDir, ".Trashes/501/old.jpg")
	if err := os.Chmod(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(readOnlyDir, 0755) }()

	err := importMedia(config{
		SourceDir:       sourceDir,
		DestDir:         destDir,
		DeleteOriginals: true,
		CheckDiskSpace:  false,
		SidecarDefault:  SidecarDelete,
		Workers:         1,
	})
	if err == nil {
		t.Fatal("expected original deletion failure")
	}
	for _, path := range []string{livePath, trashPath} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("source path %s should remain after original deletion failure: %v", path, statErr)
		}
	}
}

func TestCleanupSourceArtifactsDryRunAndErrors(t *testing.T) {
	t.Run("DryRun", func(t *testing.T) {
		sourceDir := t.TempDir()
		path := writeTestArtifact(t, sourceDir, "._live.jpg")
		removeCalls := 0
		err := cleanupSourceArtifacts(sourceDir, []sourceCleanupTarget{{Path: path, Kind: sourceArtifactAppleDouble}}, config{
			DeleteOriginals: true,
			DryRun:          true,
		}, func(string) error {
			removeCalls++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if removeCalls != 0 {
			t.Fatalf("remove called %d times during dry run", removeCalls)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dry run removed source artifact: %v", err)
		}
	})

	t.Run("JoinsFailures", func(t *testing.T) {
		sourceDir := t.TempDir()
		first := writeTestArtifact(t, sourceDir, "._first.jpg")
		second := writeTestArtifact(t, sourceDir, "._second.jpg")
		var removeCalls []string
		err := cleanupSourceArtifacts(sourceDir, []sourceCleanupTarget{
			{Path: first, Kind: sourceArtifactAppleDouble},
			{Path: second, Kind: sourceArtifactAppleDouble},
		}, config{DeleteOriginals: true}, func(path string) error {
			removeCalls = append(removeCalls, path)
			return errors.New("injected removal failure")
		})
		if err == nil {
			t.Fatal("expected cleanup error")
		}
		if len(removeCalls) != 2 {
			t.Fatalf("remove called %d times, want 2", len(removeCalls))
		}
		for _, path := range []string{first, second} {
			if !strings.Contains(err.Error(), path) {
				t.Errorf("joined error does not contain %s: %v", path, err)
			}
			if _, statErr := os.Stat(path); statErr != nil {
				t.Errorf("failed cleanup unexpectedly removed %s: %v", path, statErr)
			}
		}
	})
}

func TestCleanupSourceArtifactsRejectsUnsafeTargets(t *testing.T) {
	t.Run("OutsideSource", func(t *testing.T) {
		sourceDir := t.TempDir()
		outside := writeTestArtifact(t, t.TempDir(), "outside.jpg")
		removeCalls := 0
		err := cleanupSourceArtifacts(sourceDir, []sourceCleanupTarget{{Path: outside, Kind: sourceArtifactAppleDouble}}, config{
			DeleteOriginals: true,
		}, func(string) error {
			removeCalls++
			return nil
		})
		if err == nil {
			t.Fatal("expected outside-source target to fail")
		}
		if removeCalls != 0 {
			t.Fatalf("unsafe target reached remover %d times", removeCalls)
		}
	})

	t.Run("Symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink setup may require elevated privileges on Windows")
		}
		sourceDir := t.TempDir()
		target := writeTestArtifact(t, sourceDir, "real.jpg")
		link := filepath.Join(sourceDir, "._link.jpg")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		removeCalls := 0
		err := cleanupSourceArtifacts(sourceDir, []sourceCleanupTarget{{Path: link, Kind: sourceArtifactAppleDouble}}, config{
			DeleteOriginals: true,
		}, func(string) error {
			removeCalls++
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("expected symlink target to fail safely, got: %v", err)
		}
		if removeCalls != 0 {
			t.Fatalf("symlink target reached remover %d times", removeCalls)
		}
	})
}
