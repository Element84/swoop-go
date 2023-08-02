package s3

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestDriver(t *testing.T) {
	var err error
	ctx := context.Background()
	bucket := "testbucket"
	key := "some/key/to/a/file"
	testContent := "some test content"

	driver := &S3Driver{
		Bucket:   bucket,
		Endpoint: os.Getenv("SWOOP_S3_ENDPOINT"),
	}

	err = driver.CheckConnect(ctx)
	if err != nil {
		t.Fatalf("failed when checking connection: %s", err)
	}

	defer func() {
		err := driver.RemoveBucket(ctx)
		if err != nil {
			t.Fatalf("failed to remove bucket: %s", err)
		}
	}()

	err = driver.MakeBucket(ctx)
	if err != nil {
		t.Fatalf("failed to make bucket: %s", err)
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
