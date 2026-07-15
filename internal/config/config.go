package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
)

type TencentCloudConfig struct {
	SecretID           string        `json:",optional"`
	SecretKey          string        `json:",optional"`
	Region             string        `json:",default=ap-shanghai"`
	Endpoint           string        `json:",default=ags.tencentcloudapi.com"`
	RequestTimeout     time.Duration `json:",default=10s"`
	StatusPollInterval time.Duration `json:",default=5s"`
	StatusPollTimeout  time.Duration `json:",default=20s"`
}

type ImageSyncConfig struct {
	Endpoint         string        `json:",default=http://172.20.208.115/sync/image"`
	RequestTimeout   time.Duration `json:",default=20s"`
	AttemptTimeout   time.Duration `json:",default=5s"`
	MaxResponseBytes int64         `json:",default=1048576"`
	MaxAttempts      int           `json:",default=3,range=[1:10]"`
	RetryInterval    time.Duration `json:",default=500ms"`
}

type RegistryConfig struct {
	Username       string        `json:",optional"`
	Password       string        `json:",optional"`
	AllowedHosts   []string      `json:",default=[ths-shanghai-tcr.tencentcloudcr.com]"`
	RequestTimeout time.Duration `json:",default=10s"`
}

type Config struct {
	rest.RestConf
	TencentCloud TencentCloudConfig
	ImageSync    ImageSyncConfig
	Registry     RegistryConfig
}
