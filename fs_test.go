package billy_test

import (
	"testing"

	"fmt"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	. "srcd.works/go-billy.v1"
	"srcd.works/go-billy.v1/memfs"
	"srcd.works/go-billy.v1/osfs"
)

type copyFunc func(Filesystem, Filesystem, string, string) error

var copyFuncs []copyFunc = []copyFunc{
	copyFunc(CopyFile),
	copyFunc(CopyRecursive),
}

func Test(t *testing.T) { TestingT(t) }

type FSSuite struct {
	fsPairs [][]Filesystem
}

var _ = Suite(&FSSuite{})

func (s *FSSuite) SetUpTest(c *C) {
	sameMem := memfs.New()
	sameFs := osfs.New(c.MkDir())
	s.fsPairs = [][]Filesystem{
		[]Filesystem{
			sameMem,
			sameMem,
		},
		[]Filesystem{
			sameFs,
			sameFs,
		},
		[]Filesystem{
			memfs.New(),
			memfs.New(),
		},
		[]Filesystem{
			osfs.New(c.MkDir()),
			memfs.New(),
		},
		[]Filesystem{
			memfs.New(),
			osfs.New(c.MkDir()),
		},
		[]Filesystem{
			osfs.New(c.MkDir()),
			osfs.New(c.MkDir()),
		},
	}
}

func (s *FSSuite) TestCopyFileNonExistent(c *C) {
	for _, cf := range copyFuncs {
		for _, fsPair := range s.fsPairs {
			fromFs := fsPair[0]
			toFs := fsPair[1]
			from := "non-existent1"
			to := "non-existent2"

			err := cf(fromFs, toFs, from, to)
			c.Assert(err, NotNil)

			_, err = fromFs.Stat(from)
			c.Assert(os.IsNotExist(err), Equals, true)

			_, err = toFs.Stat(to)
			c.Assert(os.IsNotExist(err), Equals, true)
		}
	}
}

func (s *FSSuite) TestCopyFileNonExistentSource(c *C) {
	for _, cf := range copyFuncs {
		for _, fsPair := range s.fsPairs {
			fromFs := fsPair[0]
			toFs := fsPair[1]
			from := "non-existent"
			to := "foo"
			toContent := "foo"

			f, err := toFs.Create(to)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, toContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			err = cf(fromFs, toFs, from, to)
			c.Assert(err, NotNil)

			_, err = fromFs.Stat(from)
			c.Assert(os.IsNotExist(err), Equals, true)

			f, err = toFs.Open(to)
			c.Assert(err, IsNil)
			b, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, toContent)
			c.Assert(f.Close(), IsNil)
		}
	}
}

func (s *FSSuite) TestCopyFileNonExistentDestination(c *C) {
	for _, cf := range copyFuncs {
		for _, fsPair := range s.fsPairs {
			fromFs := fsPair[0]
			toFs := fsPair[1]
			from := "foo"
			to := "bar"
			fromContent := "foo"

			f, err := fromFs.Create(from)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, fromContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			err = cf(fromFs, toFs, from, to)
			c.Assert(err, IsNil)

			f, err = fromFs.Open(from)
			c.Assert(err, IsNil)
			b, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)

			f, err = toFs.Open(to)
			c.Assert(err, IsNil)
			b, err = ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)
		}
	}
}

func (s *FSSuite) TestCopyFileOverwrite(c *C) {
	for _, cf := range copyFuncs {
		for _, fsPair := range s.fsPairs {
			fromFs := fsPair[0]
			toFs := fsPair[1]
			from := "foo"
			to := "bar"
			fromContent := "foo"
			toContent := "bar"

			f, err := fromFs.Create(from)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, fromContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			f, err = toFs.Create(to)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, toContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			err = cf(fromFs, toFs, from, to)
			c.Assert(err, IsNil)

			f, err = fromFs.Open(from)
			c.Assert(err, IsNil)
			b, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)

			f, err = toFs.Open(to)
			c.Assert(err, IsNil)
			b, err = ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)
		}
	}
}

// TODO
func (s *FSSuite) TestCopyFileDestDirectory(c *C) {
	for _, cf := range copyFuncs {
		for _, fsPair := range s.fsPairs {
			fromFs := fsPair[0]
			toFs := fsPair[1]
			from := "foo"
			to := "bar"
			fromContent := "foo"
			toContent := "bar"

			f, err := fromFs.Create(from)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, fromContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			f, err = toFs.Create(to)
			c.Assert(err, IsNil)
			_, err = fmt.Fprint(f, toContent)
			c.Assert(err, IsNil)
			c.Assert(f.Close(), IsNil)

			err = cf(fromFs, toFs, from, to)
			c.Assert(err, IsNil)

			f, err = fromFs.Open(from)
			c.Assert(err, IsNil)
			b, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)

			f, err = toFs.Open(to)
			c.Assert(err, IsNil)
			b, err = ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Assert(string(b), Equals, fromContent)
			c.Assert(f.Close(), IsNil)
		}
	}
}
