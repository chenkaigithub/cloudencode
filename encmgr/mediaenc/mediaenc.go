package mediaenc

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cloudencode/common"
	"github.com/cloudencode/configure"
	log "github.com/cloudencode/logging"
)

type MediaEnc struct {
	notify          common.FileNotification
	chanFilename    chan *common.EncodeFileInfo
	chanConcurrence chan int
}

func NewMediaEnc() *MediaEnc {
	ret := &MediaEnc{
		chanFilename: make(chan *common.EncodeFileInfo),
	}
	if configure.EncodeCfgInfo.Procmax == 0 {
		procnum := runtime.NumCPU() - 1
		if procnum <= 0 {
			procnum = 1
		}
		ret.chanConcurrence = make(chan int, procnum)
	} else {
		ret.chanConcurrence = make(chan int, configure.EncodeCfgInfo.Procmax)
	}
	go ret.onwork()
	return ret
}

func (self *MediaEnc) SetUploadNotify(notify common.FileNotification) {
	self.notify = notify
}

func (self *MediaEnc) onwork() {
	for {
		info, ok := <-self.chanFilename
		if !ok {
			break
		}
		log.Infof("MediaEnc start %+v", info)
		self.doencode(info)
	}
}

func (self *MediaEnc) doencode(info *common.EncodeFileInfo) {
	self.chanConcurrence <- 1
	go self.encodefile(info)
}

func (self *MediaEnc) encodefile(info *common.EncodeFileInfo) error {
	defer func() {
		<-self.chanConcurrence
	}()
	var out []byte

	profileinfo, err := self.getProfile(info)
	if err != nil {
		return err
	}
	ffmpegbin := fmt.Sprintf("%s/ffmpeg", configure.EncodeCfgInfo.Bindir)

	resolution := fmt.Sprintf("%dx%d", profileinfo.Width, profileinfo.Height)
	vbitrate := fmt.Sprintf("%dk", profileinfo.Bitrate)
	vframerate := fmt.Sprintf("%d", profileinfo.Framerate)
	arate := fmt.Sprintf("%d", profileinfo.AudioRate)
	abitrate := fmt.Sprintf("%dk", profileinfo.AudioBitrate)
	achannel := fmt.Sprintf("%d", profileinfo.AudioChannels)
	outputFilename, err := self.getOutputFilename(info)
	if err != nil {
		return err
	}
	cmdString := fmt.Sprintf("%s -i %s -c:v libx264 -s %s -profile:v %s -b:v %s -r %s -c:a libfdk_aac -ab %s -ar %s -ac %s -copyts -f mpegts %s",
		ffmpegbin, info.Filename, resolution, profileinfo.Profile, vbitrate, vframerate, abitrate, arate, achannel, outputFilename)
	log.Infof("encodefile cmd:%s", cmdString)

	cmd := exec.Command(ffmpegbin, "-i", info.Filename, "-c:v", "libx264", "-s", resolution,
		"-profile:v", profileinfo.Profile, "-b:v", vbitrate, "-r", vframerate,
		"-c:a", "aac", "-ab", abitrate, "-ar", arate, "-ac", achannel, "-copyts", "-f", "mpegts", outputFilename)
	out, err = cmd.Output()
	if err != nil {
		log.Errorf("runslice error:%v, output:%s", err, string(out))
		return err
	}

	if len(out) > 0 {
		log.Infof("runslice command output:%s", string(out))
	}

	if self.notify != nil {
		self.notify.Notify(outputFilename)
	}
	return nil
}

func (self *MediaEnc) getOutputFilename(info *common.EncodeFileInfo) (outputFilename string, err error) {
	originDir := strings.TrimRight(configure.EncodeCfgInfo.Enctempdir, "/")

	infolist := strings.Split(info.Filename, "/")
	if len(infolist) < 2 {
		log.Errorf("MediaEnc getProfile error, filename=%s", info.Filename)
		err = fmt.Errorf("MediaEnc getProfile error, filename=%s", info.Filename)
		return
	}
	lastindex := len(infolist) - 1

	tempDir := fmt.Sprintf("%s/%s_%s_encode", originDir, info.Profilename, info.Id)

	isExist := common.CheckFileIsExist(tempDir)
	if !isExist {
		log.Infof("StartSlice mkdir alldir=%s", tempDir)
		err := common.MakeDir(tempDir)
		if err != nil {
			return "", err
		}
	}
	outputFilename = fmt.Sprintf("%s/%s", tempDir, infolist[lastindex])
	log.Infof("getOutputFilename outputFilename=%s", outputFilename)
	return
}

//http://oper-img-1.oss-cn-beijing.aliyuncs.com/540p_Xi-whiFUT8SMJc04/003.ts
func (self *MediaEnc) getProfile(info *common.EncodeFileInfo) (profileinfo *configure.EncodeProfile, err error) {
	profileinfo = nil
	err = nil

	profilename := info.Profilename

	log.Infof("getProfile profilename=%s, filename=%s", profilename, info.Filename)

	err = fmt.Errorf("MediaEnc getProfile get %s null", profilename)
	for index, item := range configure.EncodeCfgInfo.Encode_profile {
		log.Infof("MediaEnc profile index=%d, profilename=%s, item.name=%s",
			index, profilename, item.Name)
		if 0 == strings.Compare(item.Name, profilename) {
			profileinfo = &item
			err = nil
			break
		}
	}
	return
}

func (self *MediaEnc) EncodeNotify(info *common.EncodeFileInfo) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("MediaEnc Notify panic:%v, info:%+v", r, info)
		}
	}()
	self.chanFilename <- info

	return nil
}
