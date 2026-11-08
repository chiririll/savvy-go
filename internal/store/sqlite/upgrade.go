package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate/upgrade"
)

// oldBackup is the suffix of the copy of an old space file that is kept when
// the file is converted. It does not end in the space extension, so the copy
// is never taken for a space.
const oldBackup = ".v1.1"

// upgradeSpace replaces the space file at path by its conversion when the file
// does not record the current base (internal/migrate/upgrade, which knows the
// old releases and refuses a file it does not know); a file at the base, or
// none, is left alone.
//
// Nothing is lost on a crash: the old file is copied aside first, the
// conversion is written next to the original, and only the final rename
// replaces it. A start after a crash simply converts again.
func upgradeSpace(ctx context.Context, path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if need, err := upgrade.Needed(ctx, path); err != nil || !need {
		return err
	}
	if err := foldWAL(ctx, path); err != nil {
		return err
	}
	if err := copyFile(path, path+oldBackup); err != nil {
		return fmt.Errorf("keep the old file: %w", err)
	}
	tmp := path + ".converting"
	removeFiles(tmp)
	if err := upgrade.Convert(ctx, path, tmp); err != nil {
		return err
	}
	// The log of the old file must not be applied to the new one.
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
	return os.Rename(tmp, path)
}

// foldWAL writes what the write-ahead log of the file holds into the file.
func foldWAL(ctx context.Context, path string) error {
	d, err := db.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return db.Checkpoint(ctx, d)
}

func copyFile(from, to string) (err error) {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()
	_, err = io.Copy(out, in)
	return err
}
