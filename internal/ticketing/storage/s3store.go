package storage

import (
	"context"
	"fmt"
	"log"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Store struct {
	Client   *s3.Client
	Presign  *s3.PresignClient
	Bucket   string
	KeyPrefix string // e.g. "tickets/"
}

type Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	KeyPrefix string
}

func NewS3Store(cfg Config) (*S3Store, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	presign := s3.NewPresignClient(client)

	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		prefix = "tickets/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	log.Printf("S3Store ready bucket=%s endpoint=%s prefix=%s", cfg.Bucket, cfg.Endpoint, prefix)

	return &S3Store{
		Client:    client,
		Presign:   presign,
		Bucket:    cfg.Bucket,
		KeyPrefix: prefix,
	}, nil
}


func DetectContentType(fileName, provided string) string {
	ct := strings.TrimSpace(provided)
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(fileName))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	return ct
}

func (s *S3Store) MakeTicketKey(ticketID string, fileName string) string {
	safe := sanitizeFilename(fileName)
	return fmt.Sprintf("%s%s/%d-%s", s.KeyPrefix, ticketID, time.Now().UnixNano(), safe)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" {
		return "file"
	}
	return name
}
