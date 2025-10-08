package config

import (
	"fmt"
	"net/url"
	"os"
)

type (
	APP struct {
		Name      string
		Host      string
		Port      string
		Env       string
		JWTSecret string
	}
	DB struct {
		User     string
		Password string
		Name     string
		Host     string
		Port     string
	}
	S3 struct {
		Region          string
		AccessKeyID     string
		SecretAccessKey string
		BucketUploads   string
	}
	MQ struct {
		User         string
		Password     string
		Vhost        string
		Host         string
		AmqpPort     string
		Exchange     string
		ExchangeType string
		QueueName    string
	}

	Config struct {
		App APP
		DB  DB
		S3  S3
		MQ  MQ
	}
)

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func Load() Config {
	app := APP{
		Name:      getEnv("SERVICE_NAME", ""),
		Host:      getEnv("SERVICE_HOST", ""),
		Port:      getEnv("SERVICE_PORT", ""),
		Env:       getEnv("SERVICE_ENV", ""),
		JWTSecret: getEnv("SERVICE_JWT_SECRET", ""),
	}
	db := DB{
		User:     getEnv("POSTGRES_USER", ""),
		Password: getEnv("POSTGRES_PASSWORD", ""),
		Name:     getEnv("POSTGRES_DB", ""),
		Host:     getEnv("POSTGRES_HOST", ""),
		Port:     getEnv("POSTGRES_PORT", ""),
	}
	s3 := S3{
		Region:          getEnv("S3_REGION", ""),
		AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
		BucketUploads:   getEnv("S3_BUCKET_UPLOADS", ""),
	}
	mq := MQ{
		User:         getEnv("RABBITMQ_USER", ""),
		Password:     getEnv("RABBITMQ_PASSWORD", ""),
		Vhost:        getEnv("RABBITMQ_VHOST", ""),
		Host:         getEnv("RABBITMQ_HOST", ""),
		AmqpPort:     getEnv("RABBITMQ_AMQP_PORT", ""),
		Exchange:     getEnv("RABBITMQ_EXCHANGE", ""),
		ExchangeType: getEnv("RABBITMQ_EXCHANGE_TYPE", ""),
		QueueName:    getEnv("RABBITMQ_QUEUE_NAME", ""),
	}

	return Config{
		App: app,
		DB:  db,
		S3:  s3,
		MQ:  mq,
	}
}

func (c Config) DBDSN() (string, error) {
	if c.DB.User == "" || c.DB.Name == "" || c.DB.Host == "" || c.DB.Port == "" {
		return "", fmt.Errorf("incomplete DB config")
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.DB.User,
		c.DB.Password,
		c.DB.Host,
		c.DB.Port,
		c.DB.Name,
	), nil
}

func (c Config) AMQPDSN() (string, error) {
	if c.MQ.User == "" || c.MQ.Host == "" || c.MQ.AmqpPort == "" {
		return "", fmt.Errorf("invalid MQ config: user, host and amqp port are required")
	}

	return fmt.Sprintf(
		"%s://%s@%s:%s/%s",
		"amqp",
		url.UserPassword(c.MQ.User, c.MQ.Password).String(),
		c.MQ.Host,
		c.MQ.AmqpPort,
		url.PathEscape(c.MQ.Vhost),
	), nil
}
