package initialize

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/tencentyun/cos-go-sdk-v5"
	"iptv-spider/global"
	"net/http"
	"net/url"
)

func COS() *cos.Client {
	cosConfig := global.CONFIG.OSS
	if !cosConfig.Enable {
		return nil
	}
	u, err := url.Parse(cosConfig.EndPoint)
	if err != nil {
		panic("存储地址解析失败！")
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cosConfig.AccessKey,
			SecretKey: cosConfig.SecretKey,
		},
	})
	return c
}

func Minio() *minio.Client {
	minioConfig := global.CONFIG.OSS
	if !minioConfig.Enable {
		return nil
	}
	_, err := url.Parse(minioConfig.EndPoint)
	if err != nil {
		panic("存储地址解析失败！")
	}
	c, err := minio.New(minioConfig.EndPoint, &minio.Options{
		Creds:        credentials.NewStaticV4(minioConfig.AccessKey, minioConfig.SecretKey, ""),
		Secure:       minioConfig.UseSSL,
		BucketLookup: minio.BucketLookupDNS,
	})
	if err != nil {
		return nil
	}
	return c
}
