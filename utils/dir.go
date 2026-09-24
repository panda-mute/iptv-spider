package utils

import "os"

// PathExists 路径是否存在
func PathExists(path string) (isExists, isDir bool) {
	pathInfo, err := os.Stat(path)
	if err == nil {
		return true, pathInfo.IsDir()
	}
	return false, false
}
