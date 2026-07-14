package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
)

type TencentCloudConfig struct {
	SecretID           string        `json:",env=TENCENTCLOUD_SECRET_ID"`
	SecretKey          string        `json:",env=TENCENTCLOUD_SECRET_KEY"`
	Region             string        `json:",env=TENCENTCLOUD_REGION,default=ap-shanghai"`
	Endpoint           string        `json:",env=TENCENTCLOUD_ENDPOINT,default=ags.tencentcloudapi.com"`
	RequestTimeout     time.Duration `json:",default=10s"`
	StatusPollInterval time.Duration `json:",default=5s"`
	StatusPollTimeout  time.Duration `json:",default=20s"`
}

type ImageSyncConfig struct {
	Endpoint         string        `json:",env=IMAGE_SYNC_ENDPOINT,default=http://172.20.208.115/sync/image"`
	RequestTimeout   time.Duration `json:",default=20s"`
	AttemptTimeout   time.Duration `json:",default=5s"`
	MaxResponseBytes int64         `json:",default=1048576"`
	MaxAttempts      int           `json:",default=3,range=[1:10]"`
	RetryInterval    time.Duration `json:",default=500ms"`
}

type RegistryConfig struct {
	Username       string        `json:",env=REGISTRY_USERNAME"`
	Password       string        `json:",env=REGISTRY_PASSWORD"`
	AllowedHosts   []string      `json:",default=[ths-shanghai-tcr.tencentcloudcr.com]"`
	RequestTimeout time.Duration `json:",default=10s"`
}

type Config struct {
	rest.RestConf
	TencentCloud TencentCloudConfig
	ImageSync    ImageSyncConfig
	Registry     RegistryConfig
}
