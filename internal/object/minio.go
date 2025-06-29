package object

import (
	"context"
	"fmt"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/exp/slog"

	"github.com/orvice/objr/internal/conf"
)

var (
	minioClient *minio.Client
)

func Init() error {
	var err error
	// Initialize minio client object.
	minioClient, err = minio.New(conf.Conf.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.Conf.S3.AccessKeyID, conf.Conf.S3.SecretAccessKey, ""),
		Secure: conf.Conf.S3.UseSSL,
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

	uploadInfo, err := minioClient.PutObject(ctx, conf.Conf.S3.Bucket, objectName, file, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("upload success", "uploadInfo.key", uploadInfo.Key)

	return &UploadResult{
		URL: fmt.Sprintf("%s/%s", conf.Conf.S3.CDNBaseURL, uploadInfo.Key),
	}, nil
}
