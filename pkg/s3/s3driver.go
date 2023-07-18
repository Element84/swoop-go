package s3

import (
	"fmt"

	// "github.com/argoproj/pkg/file"
	// s3client "github.com/argoproj/pkg/s3"

	"github.com/spf13/pflag"
)

// S3Driver is a driver for Object Storage (MinIO, S3, etc)
type S3Driver struct {
	Endpoint              string
	Region                string
	Secure                bool
	AccessKey             string
	SecretKey             string
	RoleARN               string
	UseSDKCreds           bool
	KmsKeyId              string
	KmsEncryptionContext  string
	EnableEncryption      bool
	ServerSideCustomerKey string
}

func (s3 *S3Driver) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&s3.Endpoint,
		"s3-endpoint",
		"",
		"swoop s3 endpoint (required; SWOOP_S3_ENDPOINT)",
	)
	fs.StringVar(
		&s3.Region,
		"s3-region",
		"",
		"swoop s3 region (SWOOP_S3_REGION)",
	)
	fs.BoolVar(
		&s3.Secure,
		"s3-secure",
		false,
		"swoop s3 secure (SWOOP_S3_SECURE)",
	)
	fs.StringVar(
		&s3.AccessKey,
		"s3-access-key",
		"",
		"swoop s3 access key (required; SWOOP_S3_ACCESS_KEY)",
	)
	fs.StringVar(
		&s3.SecretKey,
		"s3-secret-key",
		"",
		"swoop s3 secret key (required; SWOOP_S3_SECRET_KEY)",
	)
	fs.StringVar(
		&s3.RoleARN,
		"s3-role-arn",
		"",
		"swoop s3 role arn (SWOOP_S3_ROLE_ARN)",
	)
	fs.BoolVar(
		&s3.UseSDKCreds,
		"s3-use-sdk-creds",
		false,
		"swoop s3 use SDK creds (SWOOP_USE_SDK_CREDS)",
	)
	fs.StringVar(
		&s3.KmsKeyId,
		"s3-kms-key-id",
		"",
		"swoop s3 kms key id (SWOOP_S3_KMS_KEY_ID)",
	)
	fs.StringVar(
		&s3.KmsEncryptionContext,
		"s3-kms-encryption-context",
		"",
		"swoop s3 kms encryption context (SWOOP_S3_KMS_ENCRYPTION_CONTEXT)",
	)
	fs.BoolVar(
		&s3.EnableEncryption,
		"s3-enable-encryption",
		false,
		"swoop s3 use SDK creds (SWOOP_S3_ENABLE_ENCRYPTION)",
	)
	fs.StringVar(
		&s3.ServerSideCustomerKey,
		"s3-server-side-customer-key",
		"",
		"swoop s3 server side customer key (required; SWOOP_S3_SERVER_SIDE_CUSTOMER_KEY)",
	)
}

// For debugging purposes
func (s3 *S3Driver) S3ClientFlags() string {
	return fmt.Sprintf(
		s3.Endpoint,
		s3.Region,
		s3.Secure,
		s3.AccessKey,
		s3.SecretKey,
		s3.RoleARN,
		s3.UseSDKCreds,
		s3.KmsKeyId,
		s3.KmsEncryptionContext,
		s3.EnableEncryption,
		s3.ServerSideCustomerKey,
	)
}
