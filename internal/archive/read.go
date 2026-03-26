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

	"github.com/klauspost/compress/zstd"
)

// InspectEntry describes a single entry read from an archive header.
type InspectEntry struct {
	// Path is the full archive-relative path.
	Path string
	// Size is the uncompressed size in bytes.
	Size int64
	// Mode is the file permission and type bits.
	Mode fs.FileMode
	// IsDir reports whether the entry is a directory.
	IsDir bool
}

// Inspect reads the archive headers (without extracting file content) and
// returns all entries.
func Inspect(archivePath string) ([]InspectEntry, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	zr, err := zstd.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("create zstd reader: %w", err)
	}
	defer zr.Close()

	tr := tar.NewReader(zr)

	var entries []InspectEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}
		entries = append(entries, InspectEntry{
			Path:  hdr.Name,
			Size:  hdr.Size,
			Mode:  hdr.FileInfo().Mode(),
			IsDir: hdr.Typeflag == tar.TypeDir,
		})
	}
	return entries, nil
}

// Restore extracts the archive to destination.
// Paths stored as "home/aman/..." are written to "<destination>/home/aman/...".
// When destination is "/", files land at their original absolute paths.
func Restore(ctx context.Context, archivePath, destination string, progress func(int, string)) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	zr, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("create zstd reader: %w", err)
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	count := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		// Sanitise path: prevent path traversal.
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue
		}

		target := filepath.Join(destination, cleanName)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}

		case tar.TypeSymlink:
			// Remove existing file/link before creating.
			_ = os.Remove(target)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir for symlink %s: %w", target, err)
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", target, hdr.Linkname, err)
			}
			count++
			if progress != nil {
				progress(count, target)
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir for file %s: %w", target, err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("write file %s: %w", target, err)
			}
			outFile.Close()
			count++
			if progress != nil {
				progress(count, target)
			}
		}
	}
	return nil
}
