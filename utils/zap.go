package utils

import (
	zaprotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap/zapcore"
	"iptv-spider/global"
	"os"
	"path"
	"time"
)

func GetWriteSyncer() (zapcore.WriteSyncer, error) {
	linkName := global.CONFIG.Zap.LinkName
	if linkName != "" && !path.IsAbs(linkName) {
		linkName = path.Join(global.CONFIG.Zap.Director, linkName)
	}
	fileWriter, err := zaprotatelogs.New(
		path.Join(global.CONFIG.Zap.Director, "%Y-%m-%d.log"),
		zaprotatelogs.WithLinkName(linkName),
		zaprotatelogs.WithMaxAge(7*24*time.Hour),
		zaprotatelogs.WithRotationTime(24*time.Hour),
	)
	if global.CONFIG.Zap.LogInConsole {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter)), err
	}
	return zapcore.AddSync(fileWriter), err
}
