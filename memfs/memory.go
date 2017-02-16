// Package memfs provides a billy filesystem base on memory.
package memfs // import "srcd.works/go-billy.v1/memfs"

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"srcd.works/go-billy.v1"
)

const separator = '/'

// Memory a very convenient filesystem based on memory files
type Memory struct {
	base      string
	s         *storage
	tempCount int
}

//New returns a new Memory filesystem
func New() *Memory {
	return &Memory{
		base: "/",
		s: newStorage(),
	}
}

// Create returns a new file in memory from a given filename.
func (fs *Memory) Create(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0)
}

// Open returns a readonly file from a given name.
func (fs *Memory) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

func (fs *Memory) open(path string, flag int) (*storage, *entry, error) {
	fullpath := fs.Join(fs.base, path)
	parts := filepath.SplitList(fullpath)
	if len(parts) == 0 {
		return fs.s, nil, nil
	}

	currentDir := fs.s
	for {
		if len(parts) == 1 {
			path := parts[0]
			e, ok := currentDir.entries[path]
			if !ok {
				if !isCreate(flag) {
					return nil, nil, os.ErrNotExist
				}

				f := newFile(fs.base, fullpath, flag)
				e = &entry{file: f}
				currentDir.entries[path] = e
				return currentDir, e, nil
			}

			return currentDir, e, nil
		}

		dirPath := parts[0]
		e, ok := currentDir.entries[dirPath]
		if !ok {
			if !isCreate(flag) {
				return nil, nil, os.ErrNotExist
			}

			e = &entry{dir: newStorage()}
			currentDir.entries[dirPath] = e
		}

		currentDir = e.dir
		parts = parts[1:]
	}
}

// OpenFile returns the file from a given name with given flag and permits.
func (fs *Memory) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	fullpath := fs.Join(fs.base, filename)
	_, e, err := fs.open(filename, flag)
	if err != nil {
		return nil, err
	}

	if e.IsDir() {
		return nil, fmt.Errorf("cannot open a directory: %s", filename)
	}

	n := newFile(fs.base, fullpath, flag)
	n.content = e.file.content

	if isAppend(flag) {
		n.position = int64(n.content.Len())
	}

	if isTruncate(flag) {
		n.content.Truncate()
	}

	return n, nil
}

// Stat returns a billy.FileInfo with the information of the requested file.
func (fs *Memory) Stat(filename string) (billy.FileInfo, error) {
	_, e, err := fs.open(filename, 0)
	if err != nil {
		return nil, err
	}

	if e.IsDir() {
		return newDirInfo(filename, e.dir.Size()), nil
	}

	return newFileInfo(filename, e.file.content.Len()), nil
}

// ReadDir returns a list of billy.FileInfo in the given directory.
func (fs *Memory) ReadDir(base string) ([]billy.FileInfo, error) {
	_, e, err := fs.open(base, 0)
	if err != nil {
		return nil, err
	}

	if !e.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", base)
	}

	var entries []billy.FileInfo
	for path, d := range e.dir.entries {
		if d.IsDir() {
			entries = append(entries, newDirInfo(path, d.dir.Size()))
		} else {
			entries = append(entries, newFileInfo(path, d.file.content.Len()))
		}
	}

	return entries, nil
}

var maxTempFiles = 1024 * 4

// TempFile creates a new temporary file.
func (fs *Memory) TempFile(dir, prefix string) (billy.File, error) {
	var fullpath string
	for {
		if fs.tempCount >= maxTempFiles {
			return nil, errors.New("max. number of tempfiles reached")
		}

		fullpath = fs.getTempFilename(dir, prefix)
		if _, err := fs.Stat(fullpath); !os.IsNotExist(err) {
			continue
		}
	}

	return fs.Create(fullpath)
}

func (fs *Memory) getTempFilename(dir, prefix string) string {
	fs.tempCount++
	filename := fmt.Sprintf("%s_%d_%d", prefix, fs.tempCount, time.Now().UnixNano())
	return fs.Join(fs.base, dir, filename)
}

// Rename moves a the `from` file to the `to` file.
func (fs *Memory) Rename(from, to string) error {
	fromDir, fromEntry, err := fs.open(from, 0)
	if err != nil {
		return err
	}

	toDir, toEntry, err := fs.open(from, 0)
	if err != nil && err != os.ErrNotExist {
		return err
	}

	fromBasename := filepath.Base(from)
	toBasename := filepath.Base(to)
	if fromEntry.IsDir() {
		if !toEntry.IsDir() {
			return fmt.Errorf("rename %s %s: not a directory", from, to)
		}

		if toEntry.dir.Size() > 0 {
			return fmt.Errorf("rename %s %s: directory not empty", from, to)
		}

		toDir.entries[to] = fromEntry
		delete(fromDir.entries, fromBasename)
		if !fromEntry.IsDir() {
			fromEntry.file.BaseFilename = filepath.Clean(to)
		}

		return nil
	}

	if toEntry.IsDir() {
		return fmt.Errorf("rename %s %s: is a directory", from, to)
	}

	toDir.entries[toBasename] = fromEntry
	delete(fromDir.entries, fromBasename)
	return nil
}

// Remove deletes a given file from storage.
func (fs *Memory) Remove(filename string) error {
	d, e, err := fs.open(filename, 0)
	if err != nil {
		return err
	}

	basename := filepath.Base(filename)
	if e.IsDir() && e.dir.Size() > 0 {
		return fmt.Errorf("remove %s: directory not empty", filename)
	}

	delete(d.entries, basename)
	return nil
}

// Join concatenatess part of a path together.
func (fs *Memory) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Dir creates a new memory filesystem whose root is the given path inside the current
// filesystem.
func (fs *Memory) Dir(path string) billy.Filesystem {
	return &Memory{
		base: fs.Join(fs.base, path),
		s:    fs.s,
	}
}

// Base returns the base path for the filesystem.
func (fs *Memory) Base() string {
	return fs.base
}

type file struct {
	billy.BaseFile

	content  *content
	position int64
	flag     int
}

func newFile(base, fullpath string, flag int) *file {
	filename, _ := filepath.Rel(base, fullpath)

	return &file{
		BaseFile: billy.BaseFile{BaseFilename: filename},
		content:  &content{},
		flag:     flag,
	}
}

func (f *file) Read(b []byte) (int, error) {
	n, err := f.ReadAt(b, f.position)
	if err != nil {
		return 0, err
	}

	return n, err
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	if f.IsClosed() {
		return 0, billy.ErrClosed
	}

	if !isReadAndWrite(f.flag) && !isReadOnly(f.flag) {
		return 0, errors.New("read not supported")
	}

	n, err := f.content.ReadAt(b, off)
	f.position += int64(n)

	return n, err
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f.IsClosed() {
		return 0, billy.ErrClosed
	}

	switch whence {
	case io.SeekCurrent:
		f.position += offset
	case io.SeekStart:
		f.position = offset
	case io.SeekEnd:
		f.position = int64(f.content.Len()) - offset
	}

	return f.position, nil
}

func (f *file) Write(p []byte) (int, error) {
	if f.IsClosed() {
		return 0, billy.ErrClosed
	}

	if !isReadAndWrite(f.flag) && !isWriteOnly(f.flag) {
		return 0, errors.New("write not supported")
	}

	n, err := f.content.WriteAt(p, f.position)
	f.position += int64(n)

	return n, err
}

func (f *file) Close() error {
	if f.IsClosed() {
		return errors.New("file already closed")
	}

	f.Closed = true
	return nil
}

func (f *file) Open() error {
	f.Closed = false
	return nil
}

type fileInfo struct {
	name  string
	size  int
	isDir bool
}

func newFileInfo(base string, size int) *fileInfo {
	return &fileInfo{
		name: base,
		size: size,
	}
}

func newDirInfo(base string, size int) *fileInfo {
	return &fileInfo{
		name:  base,
		size:  size,
		isDir: true,
	}
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return int64(fi.size)
}

func (fi *fileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (*fileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *fileInfo) IsDir() bool {
	return fi.isDir
}

func (*fileInfo) Sys() interface{} {
	return nil
}

type storage struct {
	entries map[string]*entry
}

func newStorage() *storage {
	return &storage{
		entries: make(map[string]*entry),
	}
}

func (s *storage) Size() int {
	return len(s.entries)
}

type entry struct {
	dir  *storage
	file *file
}

func (e *entry) IsDir() bool {
	return e.dir != nil
}

type content struct {
	bytes []byte
}

func (c *content) WriteAt(p []byte, off int64) (int, error) {
	prev := len(c.bytes)
	c.bytes = append(c.bytes[:off], p...)
	if len(c.bytes) < prev {
		c.bytes = c.bytes[:prev]
	}

	return len(p), nil
}

func (c *content) ReadAt(b []byte, off int64) (int, error) {
	size := int64(len(c.bytes))
	if off >= size {
		return 0, io.EOF
	}

	l := int64(len(b))
	if off+l > size {
		l = size - off
	}

	n := copy(b, c.bytes[off:off+l])
	return n, nil
}

func (c *content) Truncate() {
	c.bytes = make([]byte, 0)
}

func (c *content) Len() int {
	return len(c.bytes)
}

func isCreate(flag int) bool {
	return flag&os.O_CREATE != 0
}

func isAppend(flag int) bool {
	return flag&os.O_APPEND != 0
}

func isTruncate(flag int) bool {
	return flag&os.O_TRUNC != 0
}

func isReadAndWrite(flag int) bool {
	return flag&os.O_RDWR != 0
}

func isReadOnly(flag int) bool {
	return flag == os.O_RDONLY
}

func isWriteOnly(flag int) bool {
	return flag&os.O_WRONLY != 0
}
