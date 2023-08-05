package s3_test

import (
	"bytes"
	"context"
	"testing"

	testS3 "github.com/element84/swoop-go/pkg/utils/testing/s3"
)

func TestDriver(t *testing.T) {
	var err error
	ctx := context.Background()
	key := "some/key/to/a/file"
	testContent := "some test content"

	t3 := testS3.NewTestingS3(t, "testing-s3-")
	t3.SetupBucket(ctx)

	driver := t3.Driver

	err = driver.CheckConnect(ctx)
	if err != nil {
		t.Fatalf("failed when checking connection: %s", err)
	}

	reader := bytes.NewReader([]byte(testContent))
	err = driver.Put(ctx, key, reader, int64(reader.Len()), nil)
	if err != nil {
		t.Fatalf("failed to put: %s", err)
	}

	stream, err := driver.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get: %s", err)
	}
	defer stream.Close()

	stat, err := stream.Stat()
	if err != nil {
		t.Fatalf("failed to stat stream: %s", err)
	}

	contentBytes := make([]byte, stat.Size)
	b, err := stream.Read(contentBytes)
	if int64(b) != stat.Size && err != nil {
		t.Fatalf("failed to read stream: %s", err)
	}

	content := string(contentBytes)
	if content != testContent {
		t.Fatalf("File contents are not equal: '%s' vs '%s'", content, testContent)
	}
}
