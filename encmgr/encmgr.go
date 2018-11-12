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

const FINISH_TIMEOUT = 300 * 1000

type EncMgr struct {
	encTaskMap       cmap.ConcurrentMap
	channInfo        chan *EncInfo
	channConcurrence chan int
	checkDone        common.EncodedCheckI
	notify           common.FileUploadI
}

type EncInfo struct {
	Id        string
	StartT    int64
	EndT      int64
	SliceCtrl *mediaslice.MediaSlice
}

func NewEncMgr() *EncMgr {
	ret := &EncMgr{
		encTaskMap: cmap.New(),
		channInfo:  make(chan *EncInfo, 1000),
	}

	if configure.EncodeCfgInfo.Procmax == 0 {
		procnum := runtime.NumCPU() - 1
		if procnum <= 0 {
			procnum = 1
		}
		ret.channConcurrence = make(chan int, procnum)
	} else {
		ret.channConcurrence = make(chan int, configure.EncodeCfgInfo.Procmax)
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
		var removeList []string

		for item := range self.encTaskMap.IterBuffered() {
			encinfo := item.Val.(*EncInfo)

			if !encinfo.SliceCtrl.IsM3u8Ready {
				continue
			}

			if encinfo.SliceCtrl.IsCheckDone {
				nowT := time.Now().UnixNano() / (1000 * 1000)
				if nowT-encinfo.EndT > FINISH_TIMEOUT {
					removeList = append(removeList, encinfo.Id)
				}
				continue
			}
			var removeList []string
			for tsIndexItem := range encinfo.SliceCtrl.TsIndexMap.IterBuffered() {
				tsIndex := tsIndexItem.Key
				isDone := self.checkDone.IsEncodedDone(tsIndex)
				if !isDone {
					continue
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
				go self.dofinish(encinfo)
			}
			self.updateEncodeStatInfo(encinfo)
		}
		for _, ID := range removeList {
			log.Errorf("encMgr remove finished ID:=%s", ID)
			self.encTaskMap.Remove(ID)
		}
	}
}

func (self *EncMgr) dofinish(encinfo *EncInfo) error {
	profileName := encinfo.SliceCtrl.Info.Profilename
	Id := encinfo.SliceCtrl.Id
	sliceObj := encinfo.SliceCtrl

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
			outputFilename, err = self.synthesis(url, encinfo)
			if err == nil {
				var outputURL string
				outputURL, err = self.notify.UploadFileEx(outputFilename, sliceObj.Info.Destsubdir, sliceObj.Info.Destfile)
				if err == nil {
					encinfo.EndT = time.Now().UnixNano() / (1000 * 1000)
					encinfo.SliceCtrl.IsUploadDone = true
					costT := encinfo.EndT - encinfo.StartT
					log.Infof("finished: it takes %dms to finish encode, url=%s", costT, outputURL)
				}
			}
		}
	}
	return nil
}

func (self *EncMgr) synthesis(m3u8Url string, encinfo *EncInfo) (outputFilename string, err error) {
	var out []byte
	sliceObj := encinfo.SliceCtrl

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
	cmdStr := fmt.Sprintf("%s -i %s -c copy -copyts -movflags faststart -f mp4 %s", ffmpegbin, m3u8Url, outputFilename)
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

	nowT := time.Now().UnixNano() / (1000 * 1000)
	mediaslice := mediaslice.NewMediaSlice(id, info)
	encinfo := &EncInfo{
		Id:        id,
		StartT:    nowT,
		SliceCtrl: mediaslice,
	}

	self.channInfo <- encinfo

	return id, nil
}

func (self *EncMgr) updateEncodeStatInfo(encinfo *EncInfo) {
	ID := encinfo.Id

	info := &common.EncodeStatInfo{
		Code:        common.RET_ID_NOT_EXIST,
		Id:          ID,
		Statuscode:  common.ENCODE_NOT_EXIST,
		Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_NOT_EXIST],
		Tstotal:     0,
		Tsleftcount: 0,
	}

	if encinfo.SliceCtrl.IsM3u8Ready && !encinfo.SliceCtrl.IsCheckDone {
		costT := time.Now().UnixNano()/int64(1000*1000) - encinfo.StartT
		info = &common.EncodeStatInfo{
			Code:        common.RET_OK,
			Id:          ID,
			Statuscode:  common.ENCODE_TS_ENCODING,
			Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_TS_ENCODING],
			Tstotal:     encinfo.SliceCtrl.TSTotal,
			Tsleftcount: encinfo.SliceCtrl.TsIndexMap.Count(),
			Starttime:   encinfo.StartT,
			Endtime:     0,
			Costtime:    costT,
		}
	} else if encinfo.SliceCtrl.IsCheckDone && !encinfo.SliceCtrl.IsUploadDone {
		costT := time.Now().UnixNano()/int64(1000*1000) - encinfo.StartT
		info = &common.EncodeStatInfo{
			Code:        common.RET_OK,
			Id:          ID,
			Statuscode:  common.ENCODE_TS_ENODE_DONE,
			Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_TS_ENODE_DONE],
			Tstotal:     encinfo.SliceCtrl.TSTotal,
			Tsleftcount: encinfo.SliceCtrl.TsIndexMap.Count(),
			Starttime:   encinfo.StartT,
			Endtime:     0,
			Costtime:    costT,
		}
	} else if encinfo.SliceCtrl.IsUploadDone {
		info = &common.EncodeStatInfo{
			Code:        common.RET_OK,
			Id:          ID,
			Statuscode:  common.ENCODE_MP4_DONE,
			Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_MP4_DONE],
			Tstotal:     encinfo.SliceCtrl.TSTotal,
			Tsleftcount: encinfo.SliceCtrl.TsIndexMap.Count(),
			Starttime:   encinfo.StartT,
			Endtime:     encinfo.EndT,
			Costtime:    encinfo.EndT - encinfo.StartT,
		}

	} else {
		costT := time.Now().UnixNano()/int64(1000*1000) - encinfo.StartT
		info = &common.EncodeStatInfo{
			Code:        common.RET_OK,
			Id:          ID,
			Statuscode:  common.ENCODE_INIT,
			Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_INIT],
			Tstotal:     0,
			Tsleftcount: 0,
			Starttime:   encinfo.StartT,
			Endtime:     0,
			Costtime:    costT,
		}
	}
	self.checkDone.UpdateEncodeStat(info)
	return
}

func (self *EncMgr) GetEncodeStatInfo(ID string) (info *common.EncodeStatInfo) {
	info, _ = self.checkDone.GetEncodeStat(ID)

	return
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
		<-self.channConcurrence
	}()
	self.channConcurrence <- 1

	encinfo.SliceCtrl.StartSlice()
}
