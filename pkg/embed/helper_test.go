package embed

import (
	"embed"
	"io/fs"
	"testing"
)

//go:embed *.go
var testFS embed.FS

func TestValidateEmbedFS(t *testing.T) {
	tests := []struct {
		name     string
		files    embed.FS
		rootPath string
		wantErr  bool
	}{
		{
			name:     "valid filesystem",
			files:    testFS,
			rootPath: ".",
			wantErr:  false,
		},
		{
			name:     "nonexistent root path",
			files:    testFS,
			rootPath: "nonexistent",
			wantErr:  true,
		},
		{
			name:     "empty root path",
			files:    testFS,
			rootPath: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbedFS(tt.files, tt.rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmbedFS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListEmbeddedFiles(t *testing.T) {
	files, err := ListEmbeddedFiles(testFS, ".")
	if err != nil {
		t.Fatalf("ListEmbeddedFiles() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ListEmbeddedFiles() returned no files")
	}

	// Check that all returned paths are files (not directories)
	for _, file := range files {
		info, err := fs.Stat(testFS, file)
		if err != nil {
			t.Errorf("ListEmbeddedFiles() returned invalid file path: %v", file)
			continue
		}
		if info.IsDir() {
			t.Errorf("ListEmbeddedFiles() returned directory: %v", file)
		}
	}
}

func TestListEmbeddedFiles_EmptyRoot(t *testing.T) {
	files, err := ListEmbeddedFiles(testFS, "")
	if err != nil {
		t.Fatalf("ListEmbeddedFiles() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ListEmbeddedFiles() returned no files with empty root")
	}
}

func TestListEmbeddedFiles_NonexistentPath(t *testing.T) {
	_, err := ListEmbeddedFiles(testFS, "nonexistent")
	if err == nil {
		t.Error("ListEmbeddedFiles() should return error for nonexistent path")
	}
}

func TestCountEmbeddedFiles(t *testing.T) {
	count, err := CountEmbeddedFiles(testFS, ".")
	if err != nil {
		t.Fatalf("CountEmbeddedFiles() error = %v", err)
	}

	if count == 0 {
		t.Error("CountEmbeddedFiles() returned 0")
	}

	// Verify count matches ListEmbeddedFiles
	files, err := ListEmbeddedFiles(testFS, ".")
	if err != nil {
		t.Fatalf("ListEmbeddedFiles() error = %v", err)
	}

	if count != len(files) {
		t.Errorf("CountEmbeddedFiles() = %v, want %v (from ListEmbeddedFiles)", count, len(files))
	}
}

func TestCountEmbeddedFiles_NonexistentPath(t *testing.T) {
	_, err := CountEmbeddedFiles(testFS, "nonexistent")
	if err == nil {
		t.Error("CountEmbeddedFiles() should return error for nonexistent path")
	}
}

func TestMustEmbed(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustEmbed() should panic")
		}
	}()

	MustEmbed("test")
}

