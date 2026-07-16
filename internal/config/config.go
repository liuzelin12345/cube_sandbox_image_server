package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
)

type AuthConfig struct {
	APIKeys []string
}

type TencentCloudConfig struct {
	SecretID       string        `json:",optional"`
	SecretKey      string        `json:",optional"`
	Region         string        `json:",optional"`
	Endpoint       string        `json:",optional"`
	RequestTimeout time.Duration `json:",optional"`
}

type ImageSyncConfig struct {
	Endpoint         string        `json:",optional"`
	RequestTimeout   time.Duration `json:",optional"`
	AttemptTimeout   time.Duration `json:",optional"`
	MaxResponseBytes int64         `json:",optional"`
	MaxAttempts      int           `json:",optional,range=[1:10]"`
	RetryInterval    time.Duration `json:",optional"`
}

type RegistryConfig struct {
	Username       string        `json:",optional"`
	Password       string        `json:",optional"`
	RequestTimeout time.Duration `json:",optional"`
}

type Config struct {
	rest.RestConf
	Auth         AuthConfig
	TencentCloud TencentCloudConfig
	ImageSync    ImageSyncConfig
	Registry     RegistryConfig
}
