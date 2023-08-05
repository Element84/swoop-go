package db

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/utils"
)

const swoopdb = "swoop-db"

var (
	dockersdb = []string{"docker", "compose", "exec", "postgres", "swoop-db"}
	cmdstr    []string
)

func init() {
	cmdstr = resolveSwoopDB()
}

func resolveSwoopDB() []string {
	_, err := exec.LookPath(swoopdb)
	if err == nil {
		return []string{swoopdb}
	}
	return dockersdb
}

type TestingDB struct {
	test testing.TB
	Conf *db.ConnectConfig
}

func NewTestingDB(t testing.TB, prefix string) *TestingDB {
	name := fmt.Sprintf("swoop-%s%s", prefix, t.Name())
	return &TestingDB{
		test: t,
		Conf: &db.ConnectConfig{Database: &name},
	}
}

func (sdb *TestingDB) DatabaseName() string {
	return *sdb.Conf.Database
}

func (sdb *TestingDB) run(ctx context.Context, op []string) error {
	cmd := exec.Command(cmdstr[0], utils.Concat(cmdstr[1:], op)...)
	return cmd.Run()
}

func (sdb *TestingDB) Create(ctx context.Context) {
	sdb.test.Cleanup(sdb.Drop)

	op := []string{"up", "--database", sdb.DatabaseName()}
	err := sdb.run(ctx, op)
	if err != nil {
		sdb.test.Fatalf("failed to create test database: %s", err)
	}
}

func (sdb *TestingDB) LoadFixture(ctx context.Context, fixtureName string) {
	op := []string{"load-fixture", fixtureName, "--database", sdb.DatabaseName()}
	err := sdb.run(ctx, op)
	if err != nil {
		sdb.test.Fatalf("failed to load fixture '%s': %s", fixtureName, err)
	}
}

func (sdb *TestingDB) Drop() {
	op := []string{"drop", "--database", sdb.DatabaseName()}
	_ = sdb.run(context.Background(), op)
}
