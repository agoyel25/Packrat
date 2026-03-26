// Package archive implements tar+zstd backup and restore for home directories.
package archive

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	"packrat/internal/categories"
)

// BackupConfig configures a backup operation.
type BackupConfig struct {
	// Home is the user's home directory (e.g. /home/aman).
	Home string
	// Username is the OS username, used in the manifest.
	Username string
	// Categories is the list of categories to include (with Paths already populated).
	Categories []categories.Category
	// ArchivePath is the destination .tar.zst file path.
	ArchivePath string
	// DryRun, when true, counts files without writing an archive.
	DryRun bool
	// Progress is called for every file processed; may be nil.
	Progress func(filesProcessed int, currentPath string)
}

// BackupResult holds metadata about a completed backup.
type BackupResult struct {
	// ArchivePath is the path of the written archive (empty for dry runs).
	ArchivePath string
	// ManifestPath is the path of the sidecar manifest file (empty for dry runs).
	ManifestPath string
	// FilesArchived is the number of files (and symlinks) written.
	FilesArchived int
	// BytesWritten is the compressed size of the archive.
	BytesWritten int64
}

// Backup performs the backup described by cfg.
func Backup(ctx context.Context, cfg BackupConfig) (BackupResult, error) {
	prune := categories.DefaultPruneRules()

	// Collect the root paths to walk.
	roots := collectRoots(cfg)

	if cfg.DryRun {
		count, err := countFiles(ctx, cfg.Home, roots, prune, cfg.Progress)
		if err != nil {
			return BackupResult{}, err
		}
		return BackupResult{FilesArchived: count}, nil
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(cfg.ArchivePath), 0o755); err != nil {
		return BackupResult{}, fmt.Errorf("create archive directory: %w", err)
	}

	archiveFile, err := os.Create(cfg.ArchivePath)
	if err != nil {
		return BackupResult{}, fmt.Errorf("create archive file: %w", err)
	}
	defer archiveFile.Close()

	zw, err := zstd.NewWriter(archiveFile, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return BackupResult{}, fmt.Errorf("create zstd writer: %w", err)
	}

	tw := tar.NewWriter(zw)

	filesArchived := 0

	for _, root := range roots {
		absRoot := filepath.Join(cfg.Home, root)
		info, err := os.Lstat(absRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return BackupResult{}, fmt.Errorf("stat %s: %w", absRoot, err)
		}

		if !info.IsDir() {
			// Single file – write directly.
			if err := ctx.Err(); err != nil {
				return BackupResult{}, err
			}
			if err := writeEntry(tw, absRoot, cfg.Home); err != nil {
				return BackupResult{}, err
			}
			filesArchived++
			if cfg.Progress != nil {
				cfg.Progress(filesArchived, absRoot)
			}
			continue
		}

		// Directory – walk recursively.
		err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil // skip unreadable entries
			}
			if err := ctx.Err(); err != nil {
				return err
			}

			// Prune by directory name.
			if d.IsDir() && prune.DirNames[d.Name()] {
				return filepath.SkipDir
			}

			// Prune by home-relative path.
			rel, err := filepath.Rel(cfg.Home, path)
			if err != nil {
				return err
			}
			if prune.RelPaths[rel] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// Check if any prune rel path is a prefix of this path.
			for pruneRel := range prune.RelPaths {
				if strings.HasPrefix(rel, pruneRel+string(os.PathSeparator)) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if err := writeEntry(tw, path, cfg.Home); err != nil {
				return err
			}
			if !d.IsDir() {
				filesArchived++
				if cfg.Progress != nil {
					cfg.Progress(filesArchived, path)
				}
			}
			return nil
		})
		if err != nil {
			return BackupResult{}, fmt.Errorf("walk %s: %w", absRoot, err)
		}
	}

	if err := tw.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close tar: %w", err)
	}
	if err := zw.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close zstd: %w", err)
	}
	if err := archiveFile.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close archive: %w", err)
	}

	stat, err := os.Stat(cfg.ArchivePath)
	if err != nil {
		return BackupResult{}, err
	}

	manifestPath := cfg.ArchivePath + ".manifest.txt"
	if err := writeManifest(manifestPath, cfg, roots, filesArchived); err != nil {
		return BackupResult{}, fmt.Errorf("write manifest: %w", err)
	}

	return BackupResult{
		ArchivePath:   cfg.ArchivePath,
		ManifestPath:  manifestPath,
		FilesArchived: filesArchived,
		BytesWritten:  stat.Size(),
	}, nil
}

// collectRoots assembles the list of home-relative root paths from the config.
func collectRoots(cfg BackupConfig) []string {
	seen := make(map[string]bool)
	var roots []string
	for _, cat := range cfg.Categories {
		for _, p := range cat.Paths {
			if !seen[p] {
				seen[p] = true
				roots = append(roots, p)
			}
		}
	}
	return roots
}

// writeEntry writes a single filesystem entry into the tar archive.
// Symlinks are archived as symlinks (not followed).
func writeEntry(tw *tar.Writer, absPath, home string) error {
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil // skip missing
	}

	// Build tar header.
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	// Store path relative to "/" so it restores to the correct location.
	// e.g. /home/aman/projects/foo -> home/aman/projects/foo
	hdr.Name = strings.TrimPrefix(absPath, "/")
	if info.IsDir() {
		hdr.Name += "/"
	}

	// Resolve symlink target.
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absPath)
		if err != nil {
			return err
		}
		hdr.Linkname = target
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	// Write file content for regular files.
	if info.Mode().IsRegular() {
		f, err := os.Open(absPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
	}
	return nil
}

// countFiles walks the roots and counts files without writing an archive.
func countFiles(ctx context.Context, home string, roots []string, prune categories.PruneRules, progress func(int, string)) (int, error) {
	count := 0
	for _, root := range roots {
		absRoot := filepath.Join(home, root)
		info, err := os.Lstat(absRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		if !info.IsDir() {
			count++
			if progress != nil {
				progress(count, absRoot)
			}
			continue
		}
		err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() && prune.DirNames[d.Name()] {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(home, path)
			if err != nil {
				return err
			}
			if prune.RelPaths[rel] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			for pruneRel := range prune.RelPaths {
				if strings.HasPrefix(rel, pruneRel+string(os.PathSeparator)) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if !d.IsDir() {
				count++
				if progress != nil {
					progress(count, path)
				}
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

// writeManifest writes a human-readable sidecar file alongside the archive.
func writeManifest(path string, cfg BackupConfig, roots []string, fileCount int) error {
	prune := categories.DefaultPruneRules()

	var sb strings.Builder
	sb.WriteString("WSL Backup Manifest\n")
	sb.WriteString("===================\n\n")
	sb.WriteString(fmt.Sprintf("Created:    %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("User:       %s\n", cfg.Username))
	sb.WriteString(fmt.Sprintf("Home:       %s\n", cfg.Home))
	sb.WriteString(fmt.Sprintf("Archive:    %s\n", cfg.ArchivePath))
	sb.WriteString(fmt.Sprintf("Files:      %d\n\n", fileCount))

	sb.WriteString("Included roots:\n")
	for _, r := range roots {
		sb.WriteString(fmt.Sprintf("  %s\n", filepath.Join(cfg.Home, r)))
	}

	sb.WriteString("\nPruned directory names:\n")
	for name := range prune.DirNames {
		sb.WriteString(fmt.Sprintf("  %s\n", name))
	}

	sb.WriteString("\nPruned relative paths:\n")
	for p := range prune.RelPaths {
		sb.WriteString(fmt.Sprintf("  %s\n", p))
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}
