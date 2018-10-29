package encmgr

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/cloudencode/common"
	"github.com/cloudencode/concurrent-map"
	"github.com/cloudencode/configure"
	"github.com/cloudencode/encmgr/mediaslice"
	log "github.com/cloudencode/logging"
)

type EncMgr struct {
	encTaskMap  cmap.ConcurrentMap
	channInfo   chan *EncInfo
	currencChan chan int
	checkDone   common.EncodedCheckI
	notify      common.FileUploadI
}

type EncInfo struct {
	Id        string
	SliceCtrl *mediaslice.MediaSlice
}

func NewEncMgr() *EncMgr {
	ret := &EncMgr{
		encTaskMap:  cmap.New(),
		channInfo:   make(chan *EncInfo, 1000),
		currencChan: make(chan int, runtime.NumCPU()),
	}

	go ret.onWork()
	go ret.onCheck()
	return ret
}

func (self *EncMgr) SetUploadNotify(notify common.FileUploadI) {
	self.notify = notify
}

func (self *EncMgr) SetCheckDone(checkDone common.EncodedCheckI) {
	self.checkDone = checkDone
}

func (self *EncMgr) onCheck() {
	for {
		<-time.After(5 * time.Second)
		for item := range self.encTaskMap.IterBuffered() {
			encinfo := item.Val.(*EncInfo)

			if !encinfo.SliceCtrl.IsM3u8Ready {
				continue
			}

			if encinfo.SliceCtrl.IsCheckDone {
				continue
			}
			var removeList []string
			for tsIndexItem := range encinfo.SliceCtrl.TsIndexMap.IterBuffered() {
				tsIndex := tsIndexItem.Key
				isDone := self.checkDone.IsEncodedDone(tsIndex)
				if !isDone {
					break
				}
				removeList = append(removeList, tsIndex)
			}

			for _, tsIndex := range removeList {
				log.Infof("check remove tsindex:%s", tsIndex)
				encinfo.SliceCtrl.TsIndexMap.Remove(tsIndex)
			}
			if encinfo.SliceCtrl.TsIndexMap.Count() == 0 {
				encinfo.SliceCtrl.IsCheckDone = true
				log.Infof("check done: id=%s, info=%+v", encinfo.SliceCtrl.Id, encinfo.SliceCtrl.Info)
				go self.dofinish(encinfo.SliceCtrl.Info.Profilename, encinfo.SliceCtrl.Id, encinfo.SliceCtrl)
			}
		}
	}
}

func (self *EncMgr) dofinish(profileName string, Id string, sliceObj *mediaslice.MediaSlice) error {
	srcFilename := fmt.Sprintf("%s/%s_%s/%s.m3u8", configure.EncodeCfgInfo.Tempdir, profileName, Id, Id)
	dstFilename := fmt.Sprintf("%s/%s_%s_encode/%s.m3u8", configure.EncodeCfgInfo.Enctempdir, profileName, Id, Id)

	log.Infof("dofinish copy src=%s, dst=%s", srcFilename, dstFilename)

	_, err := common.CopyFile(dstFilename, srcFilename)
	if err != nil {
		log.Errorf("copy src(%s) to dst(%s) error:%v", srcFilename, dstFilename, err)
		return err
	}

	if self.notify != nil {
		var err error
		var url string

		for i := 0; i < 3; i++ {
			url, err = self.notify.UploadFile(dstFilename)
			if err == nil {
				break
			}
		}

		if err == nil {
			var outputFilename string
			outputFilename, err = self.synthesis(url, sliceObj)
			if err == nil {
				self.notify.UploadFileEx(outputFilename, sliceObj.Info.Destsubdir, sliceObj.Info.Destfile)
			}
		}
	}
	return nil
}

func (self *EncMgr) synthesis(m3u8Url string, sliceObj *mediaslice.MediaSlice) (outputFilename string, err error) {
	var out []byte

	ffmpegbin := fmt.Sprintf("%s/ffmpeg", configure.EncodeCfgInfo.Bindir)
	outputDir := fmt.Sprintf("%s/%s", configure.EncodeCfgInfo.Enctempdir, sliceObj.Id)

	if !common.CheckFileIsExist(outputDir) {
		err = common.MakeDir(outputDir)
		if err != nil {
			log.Errorf("EncMgr synthesis mkdir error:%v", err)
			return "", err
		}
	}
	outputFilename = fmt.Sprintf("%s/%s", outputDir, sliceObj.Info.Destfile)
	cmdStr := fmt.Sprintf("%s -i %s -c copy -copyts -f mp4 %s", ffmpegbin, m3u8Url, outputFilename)
	log.Infof("synthesis command: %s", cmdStr)

	cmd := exec.Command(ffmpegbin, "-i", m3u8Url, "-c", "copy", "-copyts", "-f", "mp4", outputFilename)
	out, err = cmd.Output()
	if err != nil {
		log.Errorf("synthesis error:%v, output:%s", err, string(out))
		return "", err
	}

	if len(out) > 0 {
		log.Infof("synthesis command output:%s", string(out))
	}
	return
}

func (self *EncMgr) WriteMsg(info *common.EncodeInfo) (string, error) {
	log.Infof("WriteMsg %+v", info)

	id := common.NewId()

	mediaslice := mediaslice.NewMediaSlice(id, info)
	encinfo := &EncInfo{
		Id:        id,
		SliceCtrl: mediaslice,
	}

	self.channInfo <- encinfo

	return id, nil
}

func (self *EncMgr) onWork() {
	for {
		encinfo := <-self.channInfo
		self.encTaskMap.Set(encinfo.Id, encinfo)

		self.StartSlice(encinfo)
	}
}

func (self *EncMgr) StartSlice(encinfo *EncInfo) {
	defer func() {
		<-self.currencChan
	}()
	self.currencChan <- 1

	encinfo.SliceCtrl.StartSlice()
}
