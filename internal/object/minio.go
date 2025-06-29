package object

import (
	"context"
	"fmt"
	"os"
	"time"

	"butterfly.orx.me/core/log"
	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/orvice/objr/internal/conf"
	"golang.org/x/exp/slog"
)

var (
	minioClient *minio.Client
)

func Init() error {
	logger := log.FromContext(context.Background())
	var err error
	// Initialize minio client object.
	logger.Info("init minio client",
		"endpoint", conf.Conf.S3.Endpoint,
		"accessKeyID", conf.Conf.S3.AccessKeyID,
		"secretAccessKey", conf.Conf.S3.SecretAccessKey,
		"useSSL", conf.Conf.S3.UseSSL,
	)
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
	logger := log.FromContext(ctx)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		logger.Error("open file failed", "error", err)
		return nil, err
	}

	var contentType string
	mtype, err := mimetype.DetectFile(filePath)
	if err == nil {
		contentType = mtype.String()
	}

	start := time.Now()

	logger.Info("upload file",
		"bucket", conf.Conf.S3.Bucket,
		"objectName", objectName, "filePath", filePath, "objectSize", objectSize, "contentType", contentType)

	uploadInfo, err := minioClient.PutObject(ctx, conf.Conf.S3.Bucket, objectName, file, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logger.Error("upload failed",
			"error", err)
		return nil, err
	}
	slog.Info("upload success",
		"duration", time.Since(start),
		"uploadInfo.key", uploadInfo.Key)

	return &UploadResult{
		URL: fmt.Sprintf("%s/%s", conf.Conf.S3.CDNBaseURL, uploadInfo.Key),
	}, nil
}
