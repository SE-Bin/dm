package streamer

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pingcap/tidb-enterprise-tools/pkg/utils"

	. "github.com/pingcap/check"
)

func (s *testStreamerSuite) TestCollectBinlogFiles(c *C) {
	var (
		valid = []string{
			"mysql-bin.000001",
			"mysql-bin.000002",
			"mysql-bin.000003",
			"mysql-bin.000004",
		}
		invalid = []string{
			"mysql-bin.invalid01",
			"mysql-bin.invalid02",
		}
		meta = []string{
			utils.MetaFilename,
			utils.MetaFilename + ".tmp",
		}
	)

	files, err := CollectAllBinlogFiles("")
	c.Assert(err, NotNil)
	c.Assert(files, IsNil)

	dir, err := ioutil.TempDir("", "test_collect_binlog_files")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	// create all valid binlog files
	for _, fn := range valid {
		f, err2 := os.Create(path.Join(dir, fn))
		c.Assert(err2, IsNil)
		f.Close()
	}
	files, err = CollectAllBinlogFiles(dir)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid)

	// create some invalid binlog files
	for _, fn := range invalid {
		f, err2 := os.Create(path.Join(dir, fn))
		c.Assert(err2, IsNil)
		f.Close()
	}
	files, err = CollectAllBinlogFiles(dir)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid)

	// create some invalid meta files
	for _, fn := range meta {
		f, err2 := os.Create(path.Join(dir, fn))
		c.Assert(err2, IsNil)
		f.Close()
	}
	files, err = CollectAllBinlogFiles(dir)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid)

	// collect newer files, none
	files, err = CollectBinlogFilesCmp(dir, valid[len(valid)-1], FileCmpBigger)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, []string{})

	// collect newer files, some
	files, err = CollectBinlogFilesCmp(dir, valid[0], FileCmpBigger)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid[1:])

	// collect newer or equal files, all
	files, err = CollectBinlogFilesCmp(dir, valid[0], FileCmpBiggerEqual)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid)

	// collect newer or equal files, some
	files, err = CollectBinlogFilesCmp(dir, valid[1], FileCmpBiggerEqual)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid[1:])

	// collect older files, none
	files, err = CollectBinlogFilesCmp(dir, valid[0], FileCmpLess)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, []string{})

	// collect older files, some
	files, err = CollectBinlogFilesCmp(dir, valid[len(valid)-1], FileCmpLess)
	c.Assert(err, IsNil)
	c.Assert(files, DeepEquals, valid[:len(valid)-1])
}
