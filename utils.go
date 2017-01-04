package billy

import (
	"io"
	"os"
)

// CopyFile copies a file across filesystems. If there is any error when
// opening, copying or closing any of the files, it tries to remove the
// destination file.
func CopyFile(src, dst Filesystem, srcPath, dstPath string) error {
	srcFile, err := src.Open(srcPath)
	if err != nil {
		return err
	}

	dstFile, err := dst.Create(dstPath)
	if err != nil {
		_ = srcFile.Close()
		_ = dst.Remove(dstPath)
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		_ = dstFile.Close()
		_ = dst.Remove(dstPath)
		_ = srcFile.Close()
		return err
	}

	if err := srcFile.Close(); err != nil {
		_ = dstFile.Close()
		_ = dst.Remove(dstPath)
		return err
	}

	err = dstFile.Close()
	if err != nil {
		_ = dst.Remove(dstPath)
		_ = srcFile.Close()
		return err
	}

	return nil
}

// Exists returns true if the path exists in the filesystem. False, otherwise.
// If there is an I/O error that prevents checking the existence of the file
// an error is returned.
func Exists(fs Filesystem, path string) (bool, error) {
	_, err := fs.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	return err == nil, err
}
