package s3

import (
	"bytes"
	"context"
	"encoding/json"
)

type JsonClient struct {
	driver *S3Driver
}

func NewJsonClient(driver *S3Driver) *JsonClient {
	return &JsonClient{driver}
}

func (s *JsonClient) GetJsonFromObject(ctx context.Context, key string) (any, error) {
	stream, err := s.driver.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	stat, err := stream.Stat()
	if err != nil {
		return nil, err
	}

	contentBytes := make([]byte, stat.Size)
	b, err := stream.Read(contentBytes)
	if int64(b) != stat.Size && err != nil {
		return nil, err
	}

	var j any
	err = json.Unmarshal(contentBytes, &j)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (s *JsonClient) PutJsonIntoObject(ctx context.Context, key string, j any) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(j)
	if err != nil {
		return err
	}

	opts := &PutOptions{
		// allows us to preview in the minio console
		// application/json would be more appropriate but can't be previewed
		ContentType: "text/plain",
	}

	err = s.driver.Put(ctx, key, b, int64(b.Len()), opts)
	if err != nil {
		return err
	}

	return nil
}
