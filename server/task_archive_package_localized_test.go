package server

import (
	"os"
	"path/filepath"
	"testing"
)

// task_archive_package.go           
// .   F3a   assertKoreanError( ·  0) .
//
//   (errArchiveStagedAndOriginalCoexist   ):   
//   runOneTaskArchiveJob(task_archives.go:102) FailTaskArchiveJob 
// task_archives.error  , tasks/page.tsx  TaskArchivesPanel  {archive.error}
//       (actool·planner  0).   
//  (archiveTask·validateArchivePath)      .
//
// stageTaskArchivePackageDelete  DB      , DB   
func TestTaskArchiveStagedAndOriginalCoexistErrorLocalized(t *testing.T) {
	dataDir := t.TempDir()
	archivePath := taskArchivePath(dataDir, 21, "42")
	if err := os.MkdirAll(filepath.Dir(archivePath), archiveDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, []byte("original"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	staged := archivePath + ".deleting-21"
	if err := os.WriteFile(staged, []byte("archive"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	_, moved, err := stageTaskArchivePackageDelete(archivePath, 21)
	if err == nil {
		t.Fatalf("        (moved=%v)", moved)
	}
	if moved {
		t.Fatal("   moved  true    ")
	}
	assertKoreanError(t, "staged_and_original_coexist", err.Error())
}
