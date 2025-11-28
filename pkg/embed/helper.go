package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
)

// MustEmbed creates an embed.FS from a directory path.
// It panics if the directory cannot be embedded or does not exist.
// This is a convenience function for users who want to embed manifests
// without manually writing the //go:embed directive.
//
// Example usage:
//
//	var manifestFiles = embed.MustEmbed("manifests")
//
// Note: This function is intended for use in build scripts or code generation.
// For most use cases, prefer using Go's native //go:embed directive:
//
//	//go:embed manifests
//	var manifestFiles embed.FS
func MustEmbed(dir string) embed.FS {
	// This is a placeholder that documents the intended usage.
	// In practice, users should use Go's native //go:embed directive.
	// This function exists to provide a clear API and documentation.
	panic(fmt.Sprintf("MustEmbed cannot be called at runtime. Use Go's //go:embed directive instead:\n\n\t//go:embed %s\n\tvar manifestFiles embed.FS", dir))
}

// ValidateEmbedFS validates that an embed.FS contains the expected directory structure.
// It checks that the rootPath exists and contains at least one file.
// Returns an error if validation fails.
func ValidateEmbedFS(files embed.FS, rootPath string) error {
	if rootPath == "" {
		rootPath = "."
	}

	// Check if root path exists
	_, err := fs.Stat(files, rootPath)
	if err != nil {
		return fmt.Errorf("root path %q does not exist in embedded filesystem: %w", rootPath, err)
	}

	// Check if root path contains at least one file
	hasFiles := false
	err = fs.WalkDir(files, rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			hasFiles = true
			return filepath.SkipAll // Stop walking once we find a file
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk embedded filesystem: %w", err)
	}

	if !hasFiles {
		return fmt.Errorf("root path %q exists but contains no files", rootPath)
	}

	return nil
}

// ListEmbeddedFiles returns a list of all files in the embedded filesystem
// under the given root path. Only returns files, not directories.
func ListEmbeddedFiles(files embed.FS, rootPath string) ([]string, error) {
	if rootPath == "" {
		rootPath = "."
	}

	var fileList []string
	err := fs.WalkDir(files, rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk embedded filesystem: %w", err)
	}

	return fileList, nil
}

// CountEmbeddedFiles returns the number of files in the embedded filesystem
// under the given root path.
func CountEmbeddedFiles(files embed.FS, rootPath string) (int, error) {
	fileList, err := ListEmbeddedFiles(files, rootPath)
	if err != nil {
		return 0, err
	}
	return len(fileList), nil
}

