package s3

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"user-manager-api/config"
)

type Client struct {
	logger *zap.Logger
	region string
	bucket string
}

func New(
	ctx context.Context,
	logger *zap.Logger,
	cfg config.S3,

) (*Client, error) {
	// ...

	return &Client{
		logger: logger,
		region: cfg.Region,
		bucket: cfg.BucketUploads,
	}, nil
}

func (c *Client) GetPublicURL(key string) string {
	return fmt.Sprintf("https://%s.example.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
}

func (c *Client) GetBucket() string { return c.bucket }
