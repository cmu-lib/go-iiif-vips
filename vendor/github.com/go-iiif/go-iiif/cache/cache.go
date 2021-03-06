package cache

import (
	iiifconfig "github.com/go-iiif/go-iiif/config"
	"strings"
)

type Cache interface {
	Exists(string) bool
	Get(string) ([]byte, error)
	Set(string, []byte) error
	Unset(string) error
}

func NewImagesCacheFromConfig(config *iiifconfig.Config) (Cache, error) {

	cfg := config.Images.Cache
	return NewCacheFromConfig(cfg)
}

func NewDerivativesCacheFromConfig(config *iiifconfig.Config) (Cache, error) {

	cfg := config.Derivatives.Cache
	return NewCacheFromConfig(cfg)
}

func NewCacheFromConfig(config iiifconfig.CacheConfig) (Cache, error) {

	var cache Cache
	var err error

	switch strings.ToLower(config.Name) {
	case "blob":
		cache, err = NewBlobCache(config)
	case "disk":
		cache, err = NewDiskCache(config)
	case "memory":
		cache, err = NewMemoryCache(config)
	case "s3":
		cache, err = NewS3Cache(config)
	default:
		cache, err = NewNullCache(config)
	}

	return cache, err
}
