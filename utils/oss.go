package utils

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"iptv-spider/global"
	"strings"
)

func UploadToOSS(key string, data []byte) {
	if global.COS != nil {
		r := bytes.NewReader(data)
		_, err := global.COS.Object.Put(context.Background(), key, r, nil)
		if err != nil {
			panic(err)
		}
	} else if global.MinioClient != nil {
		bucket := global.CONFIG.OSS.Bucket
		r := bytes.NewReader(data)
		if strings.HasPrefix(key, "/") {
			key = strings.TrimPrefix(key, "/")
		}
		var opts minio.PutObjectOptions
		if strings.HasSuffix(key, ".xml") {
			opts.ContentType = "application/xml"
		}
		_, err := global.MinioClient.PutObject(context.Background(), bucket, key, r, int64(len(data)), opts)
		if err != nil {
			global.LOG.Error("上传到存储服务失败", zap.Error(err))
		}
	} else {
		global.LOG.Error("未配置存储服务")
	}
}
