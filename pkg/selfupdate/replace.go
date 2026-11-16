/*
Copyright 2026 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotWritable indicates the directory holding the CLI binary cannot be written to.
var ErrNotWritable = errors.New("install directory is not writable")

// ErrOldBinaryInUse indicates a previous dapr binary (".old") is still running and cannot be removed.
var ErrOldBinaryInUse = errors.New("a previous dapr binary is still in use; stop any running dapr processes and try again")

// StageBinary copies extractedPath next to currentPath as currentPath+".new",
// giving it the mode of the current binary. The current binary is never touched.
func StageBinary(currentPath, extractedPath string) (string, error) {
	if extractedPath == "" {
		return "", errors.New("no binary found in the downloaded archive")
	}

	info, err := os.Stat(currentPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat current binary: %w", err)
	}

	src, err := os.Open(extractedPath)
	if err != nil {
		return "", fmt.Errorf("failed to open extracted binary: %w", err)
	}
	defer src.Close()

	stagedPath := currentPath + ".new"
	// Remove any stale staged file so the new one gets the right mode.
	if err = os.Remove(stagedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", wrapStageError(currentPath, err)
	}

	dst, err := os.OpenFile(stagedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return "", wrapStageError(currentPath, err)
	}

	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(stagedPath)
		return "", wrapStageError(currentPath, err)
	}
	if err = dst.Close(); err != nil {
		os.Remove(stagedPath)
		return "", wrapStageError(currentPath, err)
	}
	// OpenFile's mode is subject to umask; set it explicitly.
	if err = os.Chmod(stagedPath, info.Mode().Perm()); err != nil {
		os.Remove(stagedPath)
		return "", wrapStageError(currentPath, err)
	}

	return stagedPath, nil
}

func wrapStageError(currentPath string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%w: %s: %w", ErrNotWritable, filepath.Dir(currentPath), err)
	}
	return fmt.Errorf("failed to stage new binary: %w", err)
}

// rename is a variable so tests can inject failures.
var rename = os.Rename

// ReplaceBinary swaps stagedPath into place of currentPath. On failure the
// original binary at currentPath is left in place and stagedPath is removed,
// except when the original could not be restored: then stagedPath is kept so
// the new binary is not lost.
func ReplaceBinary(currentPath, stagedPath, goos string) error {
	keepStaged, err := replace(currentPath, stagedPath, goos)
	if err != nil && !keepStaged {
		os.Remove(stagedPath)
	}
	return err
}

func replace(currentPath, stagedPath, goos string) (keepStaged bool, err error) {
	if goos != "windows" {
		if err := rename(stagedPath, currentPath); err != nil {
			return false, wrapReplaceError(currentPath, err)
		}
		return false, nil
	}

	// A running .exe can be renamed but not overwritten, so move it aside first.
	oldPath := currentPath + ".old"
	// A leftover .old from a previous update may still be a running image.
	os.Remove(oldPath)
	if _, statErr := os.Lstat(oldPath); statErr == nil {
		return false, fmt.Errorf("%w: %s", ErrOldBinaryInUse, oldPath)
	}
	if err := rename(currentPath, oldPath); err != nil {
		return false, wrapReplaceError(currentPath, err)
	}
	if err := rename(stagedPath, currentPath); err != nil {
		if rbErr := rename(oldPath, currentPath); rbErr != nil {
			return true, fmt.Errorf("failed to replace %s and failed to restore it; the original binary is at %s and the new binary is at %s: %w (restore error: %v)",
				currentPath, oldPath, stagedPath, err, rbErr)
		}
		return false, wrapReplaceError(currentPath, err)
	}
	return false, nil
}

func wrapReplaceError(currentPath string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%w: %s: %w", ErrNotWritable, filepath.Dir(currentPath), err)
	}
	return fmt.Errorf("failed to replace binary: %w", err)
}

// CleanupOld removes the ".old" binary left behind by a previous Windows update.
// It is best-effort and ignores errors.
func CleanupOld(currentPath string) {
	_ = os.Remove(currentPath + ".old")
}
