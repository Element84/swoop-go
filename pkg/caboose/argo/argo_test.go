package argo_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/utils/testing/config"
	"github.com/element84/swoop-go/pkg/utils/testing/db"
	"github.com/element84/swoop-go/pkg/utils/testing/k8s"
	"github.com/element84/swoop-go/pkg/utils/testing/s3"

	"github.com/element84/swoop-go/pkg/caboose/argo"
)

var ac *argo.ArgoCaboose

func initTest(ctx context.Context, t *testing.T, uuids []uuid.UUID) {
	// test bucket init
	t3 := s3.NewTestingS3(t, "caboose-argo-")
	t3.SetupBucket(ctx)
	for _, _uuid := range uuids {
		t3.PutInput(ctx, _uuid)
		t3.PutOutput(ctx, _uuid)
	}

	// test namespace init
	testConfigFlags := k8s.TestNamespaceAndConfigFlags(ctx, t, "caboose-argo-")

	// test db init
	testdb := db.NewTestingDB(t, "caboose_argo_")
	testdb.Create(ctx)

	// test caboose
	ac = &argo.ArgoCaboose{
		S3Driver:       t3.Driver,
		SwoopConfig:    config.LoadConfigFixture(t),
		K8sConfigFlags: testConfigFlags,
		DbConfig:       testdb.PoolConfig(),
	}
}

func TestCabooseRuns(t *testing.T) {
	rootctx := context.Background()

	ctx, cancel := context.WithTimeout(rootctx, 10*time.Second)
	initTest(ctx, t, []uuid.UUID{})

	ctx, cancel = context.WithTimeout(rootctx, 10*time.Second)
	go ac.Run(ctx, cancel)
}
