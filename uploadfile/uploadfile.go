package uploadfile

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cloudencode/common"
	log "github.com/cloudencode/logging"
	"github.com/oss"
)

type Uploadfile struct {
	AccessKeyId     string
	AccessKeySecret string
	EndPoint        string
	Bucket          string
	chanFile        chan string
	notify          common.FileNotification
}

func NewUploadfile(keyid string, keysec string, endpoint string, bucket string) *Uploadfile {
	return &Uploadfile{
		AccessKeyId:     keyid,
		AccessKeySecret: keysec,
		EndPoint:        endpoint,
		Bucket:          bucket,
		chanFile:        make(chan string, 20000),
	}
}

func (self *Uploadfile) SetRedisNotify(notify common.FileNotification) {
	self.notify = notify
}

func (self *Uploadfile) Start() {
	go self.onWork()

	return
}

func (self *Uploadfile) onWork() {
	log.Infof("uploadfile is starting: keyid=%s, keysec=%s, endpoint=%s, bucket=%s",
		self.AccessKeyId, self.AccessKeySecret, self.EndPoint, self.Bucket)

	for {
		filename := <-self.chanFile
		log.Infof("upload file:%s", filename)
		go self.doUpload(filename)
	}
}

func (self *Uploadfile) doUpload(filename string) {
	var err error
	var url string

	for index := 0; index < 5; index++ {
		url, err = self.UploadFile(filename)
		if err == nil {
			if self.isTsFile(filename) {
				self.removeFile(filename)
			}
			break
		}
	}
	if err != nil {
		log.Errorf("upload oss error:%v, filename:%s", err, filename)
	} else {
		if self.notify != nil && self.isTsFile(url) {
			self.notify.Notify(url)
		}
	}
}

func (self *Uploadfile) isTsFile(filename string) bool {
	tsPos := strings.LastIndex(filename, ".")
	if tsPos <= 0 {
		log.Errorf("Uploadfile filename(%s) error", filename)
		return false
	}
	if filename[tsPos+1:] != "ts" {
		log.Warningf("Uploadfile filename(%s) not mpegts", filename)
		return false
	}
	return true
}

func (self *Uploadfile) removeFile(filename string) error {
	err := os.Remove(filename)
	if err == nil {
		log.Infof("remove ok file=%s", filename)
	} else {
		log.Errorf("remove file=%s, error=%v", filename, err)
	}

	return err
}

func (self *Uploadfile) Notify(filename string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("uploadfile panic:%v, filename=%s", r, filename)
		}
	}()

	self.chanFile <- filename
	return nil
}

func (self *Uploadfile) UploadFile(filename string) (url string, err error) {
	infolist := strings.Split(filename, "/")
	if len(infolist) < 3 {
		log.Errorf("uploadFile filename(%s) error", filename)
		return "", errors.New(fmt.Sprintf("filename(%s) error", filename))
	}

	lastIndex := len(infolist) - 1
	lastname := infolist[lastIndex]
	profilenameID := infolist[lastIndex-1]

	filekey := fmt.Sprintf("%s/%s", profilenameID, lastname)
	Client, err := oss.New(self.EndPoint, self.AccessKeyId, self.AccessKeySecret)
	if err != nil {
		log.Errorf("oss.New error:%v", err)
		return "", err
	}
	bucket, err := Client.Bucket(self.Bucket)
	if err != nil {
		log.Errorf("Get Bucket error:%v", err)
		return "", err
	}

	err = bucket.PutObjectFromFile(filekey, filename)
	if err != nil {
		log.Errorf("PutObjectFromFile filekey=%s, filepath=%s bucket=%s error:%v",
			filekey, filename, self.Bucket, err)
		return "", err
	}

	url = fmt.Sprintf("http://%s.oss-cn-beijing.aliyuncs.com/%s", self.Bucket, filekey)
	log.Info("PutObjectFromFile ok: http://" + self.Bucket + ".oss-cn-beijing.aliyuncs.com/" + filekey)
	return url, nil
}

func (self *Uploadfile) UploadFileEx(filename string, subpath string, outputFilename string) (url string, err error) {
	filekey := fmt.Sprintf("%s/%s", subpath, outputFilename)
	Client, err := oss.New(self.EndPoint, self.AccessKeyId, self.AccessKeySecret)
	if err != nil {
		log.Errorf("oss.New error:%v", err)
		return "", err
	}
	bucket, err := Client.Bucket(self.Bucket)
	if err != nil {
		log.Errorf("Get Bucket error:%v", err)
		return "", err
	}

	err = bucket.PutObjectFromFile(filekey, filename)
	if err != nil {
		log.Errorf("PutObjectFromFile filekey=%s, filepath=%s bucket=%s error:%v",
			filekey, filename, self.Bucket, err)
		return "", err
	}

	url = fmt.Sprintf("http://%s.oss-cn-beijing.aliyuncs.com/%s", self.Bucket, filekey)
	log.Info("PutObjectFromFile ok: http://" + self.Bucket + ".oss-cn-beijing.aliyuncs.com/" + filekey)
	return url, nil
}
