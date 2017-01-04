package billy_test

import (
	"errors"
	"os"
	"testing"

	"srcd.works/go-billy.v1"
	"srcd.works/go-billy.v1/memory"

	. "gopkg.in/check.v1"
	"io/ioutil"
)

func Test(t *testing.T) { TestingT(t) }

type UtilsSuite struct{}

var _ = Suite(&UtilsSuite{})

func (s *UtilsSuite) TestCopyFile(c *C) {
	src := memory.New()
	dst := memory.New()
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	_, err = f.Write([]byte("foo"))
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, IsNil)
	f, err = dst.Open(path)
	c.Assert(err, IsNil)
	b, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	c.Assert(string(b), Equals, "foo")
	c.Assert(f.Close(), IsNil)
}

func (s *UtilsSuite) TestCopyFileNonExistentSource(c *C) {
	src := memory.New()
	dst := memory.New()
	path := "path"

	err := billy.CopyFile(src, dst, path, path)
	c.Assert(os.IsNotExist(err), Equals, true)
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestCopyFileNonWriteableDest(c *C) {
	src := memory.New()
	dst := &errorCreateFs{memory.New(), false}
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, ErrorMatches, "test error")
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestCopyFileCreateError(c *C) {
	src := memory.New()
	dst := &errorCreateFs{memory.New(), true}
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, ErrorMatches, "test error")
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestCopyFileReaderError(c *C) {
	src := &badReaderFs{memory.New()}
	dst := memory.New()
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, ErrorMatches, "test error")
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestCopyFileSourceCloseError(c *C) {
	var src billy.Filesystem = memory.New()
	dst := memory.New()
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)
	src = &badCloserFs{src}

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, ErrorMatches, "test error")
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestCopyFileDestCloseError(c *C) {
	src := memory.New()
	var dst billy.Filesystem = memory.New()
	path := "path"
	f, err := src.Create(path)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)
	dst = &badCloserFs{dst}

	err = billy.CopyFile(src, dst, path, path)
	c.Assert(err, ErrorMatches, "test error")
	exists, err := billy.Exists(dst, path)
	c.Assert(err, IsNil)
	c.Assert(exists, Equals, false)
}

func (s *UtilsSuite) TestExistsFalse(c *C) {
	fs := memory.New()

	e, err := billy.Exists(fs, "non-existent")
	c.Assert(err, IsNil)
	c.Assert(e, Equals, false)
}

func (s *UtilsSuite) TestExistsTrue(c *C) {
	fs := memory.New()
	f, err := fs.Create("existent")
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	e, err := billy.Exists(fs, "existent")
	c.Assert(err, IsNil)
	c.Assert(e, Equals, true)
}

func (s *UtilsSuite) TestExistsError(c *C) {
	fs := &errorStatFs{memory.New()}

	e, err := billy.Exists(fs, "existent")
	c.Assert(err, ErrorMatches, "test error")
	c.Assert(e, Equals, false)
}

type errorStatFs struct {
	billy.Filesystem
}

func (fs *errorStatFs) Stat(path string) (billy.FileInfo, error) {
	return nil, errors.New("test error")
}

type errorCreateFs struct {
	billy.Filesystem
	do bool
}

func (fs *errorCreateFs) Create(path string) (billy.File, error) {
	if fs.do {
		_, _ = fs.Filesystem.Create(path)
	}

	return nil, errors.New("test error")
}

type badReaderFs struct {
	billy.Filesystem
}

func (fs *badReaderFs) Open(path string) (billy.File, error) {
	f, err := fs.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}

	return &badReadFile{f}, nil
}

type badReadFile struct {
	billy.File
}

func (f *badReadFile) Read(b []byte) (int, error) {
	return 0, errors.New("test error")
}

type badCloserFs struct {
	billy.Filesystem
}

func (fs *badCloserFs) Open(path string) (billy.File, error) {
	f, err := fs.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}

	return &badCloserFile{f}, nil
}

func (fs *badCloserFs) Create(path string) (billy.File, error) {
	f, err := fs.Filesystem.Create(path)
	if err != nil {
		return nil, err
	}

	return &badCloserFile{f}, nil
}

type badCloserFile struct {
	billy.File
}

func (f *badCloserFile) Close() error {
	return errors.New("test error")
}
