package object

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/exp/slog"
)

var (
	minioClient *minio.Client
	bucket      string
	cdnBaseURL  string
)

func Init() error {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("S3_ACCESS_KEY")
	bucket = os.Getenv("S3_BUCKET")
	cdnBaseURL = strings.TrimRight(os.Getenv("CDN_BASE_URL"), "/")
	useSSL := true

	var err error
	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return err
	}

	return nil // minioClient is now set up
}

type UploadResult struct {
	URL string
}

func Upload(ctx context.Context, objectName string, filePath string, objectSize int64) (*UploadResult, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	var contentType string
	mtype, err := mimetype.DetectFile(filePath)
	if err == nil {
		contentType = mtype.String()
	}

	uploadInfo, err := minioClient.PutObject(ctx, bucket, objectName, file, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("upload success", "uploadInfo.key", uploadInfo.Key)

	return &UploadResult{
		URL: fmt.Sprintf("%s/%s", cdnBaseURL, uploadInfo.Key),
	}, nil
}
