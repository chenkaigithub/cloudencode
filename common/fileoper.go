package common

import (
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
