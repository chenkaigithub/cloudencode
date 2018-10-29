package common

import (
	"io"
	"os"

	log "github.com/cloudencode/logging"
)

func CheckFileIsExist(filename string) bool {
	exist := true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func MakeDirAll(dirpath string) error {
	err := os.MkdirAll(dirpath, os.ModePerm)
	if err != nil {
		log.Errorf("MakeDirAll error:%v", err)
		return err
	}
	return nil
}

func MakeDir(dirpath string) error {
	err := os.Mkdir(dirpath, os.ModePerm)
	if err != nil {
		log.Errorf("MakeDirAll error:%v", err)
		return err
	}
	return nil
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
