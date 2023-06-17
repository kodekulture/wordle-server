package badgr

import (
	"log"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
)

var testDB *badger.DB

func TestMain(m *testing.M) {
	// create a tmp dir for badger
	dir, err := os.MkdirTemp("/tmp", "badger_test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// open the db
	testDB, err = badger.Open(badger.DefaultOptions(dir))
	if err != nil {
		log.Fatal(err)
	}
	defer testDB.Close()
	// run the tests
	m.Run()
}
