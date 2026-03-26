// Package packages provides system package manager detection and snapshot support.
package packages

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Manager identifies a system package manager.
type Manager string

const (
	Apt    Manager = "apt"
	Pacman Manager = "pacman"
	Snap   Manager = "snap"
)

// Snapshot holds the exported package list from one or more package managers.
type Snapshot struct {
	ExportedAt string              `json:"exported_at"`
	Managers   map[string][]string `json:"managers"` // manager name -> package list
}

// Detect returns the list of package managers available on the current system.
// It checks for the presence of apt, pacman, and snap binaries.
func Detect() []Manager {
	var found []Manager
	candidates := []Manager{Apt, Pacman, Snap}
	for _, mgr := range candidates {
		if _, err := exec.LookPath(string(mgr)); err == nil {
			found = append(found, mgr)
		}
	}
	return found
}

// Export queries each of the given managers and returns a Snapshot.
func Export(managers []Manager) (*Snapshot, error) {
	snap := &Snapshot{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Managers:   make(map[string][]string),
	}
	for _, mgr := range managers {
		pkgs, err := listPackages(mgr)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", mgr, err)
		}
		snap.Managers[string(mgr)] = pkgs
	}
	return snap, nil
}

// listPackages runs the appropriate query command and returns the package list.
func listPackages(mgr Manager) ([]string, error) {
	switch mgr {
	case Apt:
		return runLines("dpkg-query", "-f", "${binary:Package}\n", "-W")
	case Pacman:
		return runLines("pacman", "-Qqe")
	case Snap:
		return listSnapPackages()
	default:
		return nil, fmt.Errorf("unknown manager: %s", mgr)
	}
}

// runLines executes a command and returns stdout split into non-empty lines.
func runLines(name string, args ...string) ([]string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return nil, err
	}
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// listSnapPackages runs `snap list` and parses the Name column,
// skipping the header and the "snapd" entry.
func listSnapPackages() ([]string, error) {
	out, err := exec.Command("snap", "list", "--unicode=never").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // skip header
		}
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if name == "snapd" {
			continue
		}
		pkgs = append(pkgs, name)
	}
	return pkgs, scanner.Err()
}

// SaveSnapshot writes a Snapshot as JSON to path.
func SaveSnapshot(path string, s *Snapshot) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSnapshot reads a Snapshot from a JSON file at path.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// InstallCommand returns the shell command string to install the given packages
// with the given manager.
func InstallCommand(mgr Manager, pkgs []string) string {
	if len(pkgs) == 0 {
		return ""
	}
	switch mgr {
	case Apt:
		return "sudo apt-get install -y " + strings.Join(pkgs, " ")
	case Pacman:
		return "sudo pacman -S --needed " + strings.Join(pkgs, " ")
	case Snap:
		lines := make([]string, len(pkgs))
		for i, p := range pkgs {
			lines[i] = "sudo snap install " + p
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}
