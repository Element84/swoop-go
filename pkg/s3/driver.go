package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// S3Driver is a driver for Object Storage (MinIO, S3, etc)
type S3Driver struct {
	Bucket   string
	Endpoint string
	Region   string
	Session  *session.Session
}

func (s3 *S3Driver) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&s3.Bucket,
		"s3-bucket",
		"",
		"swoop s3 bucket name (required; SWOOP_S3_BUCKET)",
	)
	cobra.MarkFlagRequired(fs, "s3-bucket")
	fs.StringVar(
		&s3.Endpoint,
		"s3-endpoint",
		"",
		"swoop s3 endpoint (required; SWOOP_S3_ENDPOINT)",
	)
}

func (d *S3Driver) AWSCreds() (*credentials.Credentials, error) {
	// see https://pkg.go.dev/github.com/aws/aws-sdk-go/aws/session
	// for details on how this gets creds and the supported env vars
	if d.Session == nil {
		d.Session = session.Must(session.NewSessionWithOptions(session.Options{
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			Config: aws.Config{
				Region: aws.String(d.Region),
			},
		}))
	}

	if d.Region == "" {
		d.Region = *d.Session.Config.Region
	}

	creds, err := d.Session.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	return credentials.NewStaticV4(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	), nil
}

func (d *S3Driver) GetCredentials() (*credentials.Credentials, error) {
	// let's try minio env vars for creds first
	id := os.Getenv("MINIO_ROOT_USER")
	secret := os.Getenv("MINIO_ROOT_PASSWORD")
	token := ""

	if id == "" || secret == "" {
		id = os.Getenv("MINIO_ACCESS_KEY")
		secret = os.Getenv("MINIO_SECRET_KEY")
	}

	// if we end up with something then great, let's use it
	if id != "" || secret != "" {
		return credentials.NewStaticV4(id, secret, token), nil
	}

	// if that didn't work then try the AWS SDK
	return d.AWSCreds()
}

func (d *S3Driver) newS3Client(ctx context.Context) (*s3client, error) {
	var (
		minioClient *minio.Client
		err         error
	)

	secure := true
	endpoint := "s3.amazonaws.com"

	if d.Endpoint != "" {
		endpointURL, err := url.Parse(d.Endpoint)
		if err != nil {
			return nil, err
		}

		// minio wants to add the scheme itself and uses the `secure` flag
		// to determine http vs https, so we drop the scheme off our endpoint
		secure = endpointURL.Scheme == "https"
		endpointURL.Scheme = ""
		// we slice off the first two chars from the endpoint string
		// because .String() includes `//` even with an empty Scheme
		endpoint = endpointURL.String()[2:]
	}

	credentials, err := d.GetCredentials()
	if err != nil {
		return nil, err
	}

	bucketLookupType := minio.BucketLookupAuto
	minioOpts := &minio.Options{
		Creds:        credentials,
		Secure:       secure,
		Region:       d.Region,
		BucketLookup: bucketLookupType,
	}
	minioClient, err = minio.New(endpoint, minioOpts)
	if err != nil {
		return nil, err
	}

	return &s3client{
		S3Driver:    *d,
		minioClient: minioClient,
		context:     ctx,
	}, nil
}

// TODO: check valid config function to use at init

func (d *S3Driver) Get(ctx context.Context, key string) (*minio.Object, error) {
	// TODO: retry transient errors? Or rely on higher-level retry maybe?
	s3, err := d.newS3Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return s3.GetStream(key)
}

func (d *S3Driver) Put(ctx context.Context, key string, stream io.Reader, length int64) error {
	// TODO: retry transient errors? Or rely on higher-level retry maybe?
	s3, err := d.newS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return s3.PutStream(key, stream, length)
}

func (d *S3Driver) makeBucket(ctx context.Context) error {
	s3, err := d.newS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return s3.MakeBucket()
}

func (d *S3Driver) removeBucket(ctx context.Context) error {
	s3, err := d.newS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new S3 client: %v", err)
	}

	return s3.removeBucket()
}
