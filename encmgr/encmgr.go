package encmgr

import (
	"runtime"
	"time"

	"github.com/cloudencode/common"
	"github.com/cloudencode/concurrent-map"
	"github.com/cloudencode/encmgr/mediaslice"
	log "github.com/cloudencode/logging"
)

type EncMgr struct {
	encTaskMap  cmap.ConcurrentMap
	channInfo   chan *EncInfo
	currencChan chan int
	checkDone   common.EncodedCheckI
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
			}
		}
	}
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
