package global

import (
	"github.com/astaxie/beego/cache"
	"github.com/minio/minio-go/v7"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"iptv-spider/config"
)

var (
	DB                 *gorm.DB
	CACHE              cache.Cache
	COS                *cos.Client
	MinioClient        *minio.Client
	VIPER              *viper.Viper
	CONFIG             *config.Server
	LOG                *zap.Logger
	CRON               *cron.Cron
	ConcurrencyControl = &singleflight.Group{}
)
