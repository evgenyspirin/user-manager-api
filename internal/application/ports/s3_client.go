package ports

type S3Client interface {
	GetPublicURL(key string) string
	GetBucket() string
}
