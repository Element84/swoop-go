package s3

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/sse"
)

// using an interface here so we can eventually replace
// the client with a mock for testing without docker
type minioClient interface {
	BucketExists(ctx context.Context, bucketName string) (found bool, err error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	RemoveBucketWithOptions(ctx context.Context, bucketName string, opts minio.RemoveBucketOptions) error
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	SetBucketEncryption(ctx context.Context, bucketname string, config *sse.Configuration) error
}

type s3client struct {
	S3Driver
	minioClient minioClient
	context     context.Context
}

func (s3 *s3client) GetStream(key string) (*minio.Object, error) {
	stream, err := s3.minioClient.GetObject(
		s3.context,
		s3.Bucket,
		key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	// NoSuchKey not raised until accessing stream, so we call Stat
	_, err = stream.Stat()
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s3 *s3client) PutStream(key string, stream io.Reader, length int64) error {
	_, err := s3.minioClient.PutObject(
		s3.context,
		s3.Bucket,
		key,
		stream,
		length,
		minio.PutObjectOptions{},
	)

	return err
}

func (s3 *s3client) BucketExists() (bool, error) {
	return s3.minioClient.BucketExists(
		s3.context,
		s3.Bucket,
	)
}

func (s3 *s3client) MakeBucket() error {
	return s3.minioClient.MakeBucket(
		s3.context,
		s3.Bucket,
		minio.MakeBucketOptions{Region: s3.Region},
	)
}

func (s3 *s3client) removeBucket() error {
	return s3.minioClient.RemoveBucketWithOptions(
		s3.context,
		s3.Bucket,
		// we only use this for testing cleanup so ForceDelete is safe
		minio.RemoveBucketOptions{ForceDelete: true},
	)
}
