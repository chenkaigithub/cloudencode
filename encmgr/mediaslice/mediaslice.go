package mediaslice

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudencode/common"
	"github.com/cloudencode/concurrent-map"
	"github.com/cloudencode/configure"
	log "github.com/cloudencode/logging"
)

type MediaSlice struct {
	Id           string
	Info         *common.EncodeInfo
	cmd          *exec.Cmd
	IsM3u8Ready  bool
	IsCheckDone  bool
	IsUploadDone bool
	TSTotal      int
	TsIndexMap   cmap.ConcurrentMap
}

func NewMediaSlice(id string, info *common.EncodeInfo) *MediaSlice {
	return &MediaSlice{
		Id:           id,
		Info:         info,
		IsM3u8Ready:  false,
		IsCheckDone:  false,
		IsUploadDone: false,
		TsIndexMap:   cmap.New(),
	}
}

//ffmpeg -i 2.mp4 -c copy -hls_time 10 -hls_list_size 0 -hls_segment_filename './temp/%03d.ts' -f hls ./temp/2.m3u8
func (self *MediaSlice) StartSlice() error {
	originDir := strings.TrimRight(configure.EncodeCfgInfo.Tempdir, "/")
	tempDir := fmt.Sprintf("%s/%s_%s", originDir, self.Info.Profilename, self.Id)

	isExist := common.CheckFileIsExist(tempDir)
	if !isExist {
		log.Infof("StartSlice mkdir alldir=%s", tempDir)
		err := common.MakeDir(tempDir)
		if err != nil {
			return err
		}
	}
	err := self.runSlice(tempDir)
	return err
}

func (self *MediaSlice) runSlice(tempdir string) error {
	tsDscr := fmt.Sprintf("%s", tempdir)
	tsDscr = tsDscr + "/%03d.ts"

	outputfile := fmt.Sprintf("%s/%s.m3u8", tempdir, self.Id)
	intervalStr := fmt.Sprintf("%d", configure.EncodeCfgInfo.Hlsinterval)
	ffmpegbin := fmt.Sprintf("%s/ffmpeg", configure.EncodeCfgInfo.Bindir)

	cmdstring := fmt.Sprintf("%s -i %s -c copy -hls_time %s -hls_list_size 0 -hls_segment_filename %s -f hls %s",
		ffmpegbin, self.Info.Srcfile, intervalStr, tsDscr, outputfile)
	log.Infof("run command:%s", cmdstring)
	self.cmd = exec.Command(ffmpegbin, "-i", self.Info.Srcfile, "-c", "copy", "-hls_time", intervalStr,
		"-hls_list_size", "0", "-hls_segment_filename", tsDscr, "-f", "hls", outputfile)

	out, err := self.cmd.Output()
	if err != nil {
		log.Errorf("runslice error:%v, output:%s", err, string(out))
		return err
	}
	if len(out) > 0 {
		log.Infof("runslice command output:%s", string(out))
	}

	err = self.makeTslist(outputfile)
	return err
}

func (self *MediaSlice) makeTslist(m3u8file string) error {
	fi, err := os.Open(m3u8file)
	if err != nil {
		log.Errorf("makeTslist openfile(%s) error: %s", m3u8file, err)
		return err
	}
	defer fi.Close()

	br := bufio.NewReader(fi)
	for {
		lineData, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		infoStr := string(lineData)
		pos := strings.Index(infoStr, "#EXTINF")
		if pos == 0 {
			lineData, _, err := br.ReadLine()
			if err == io.EOF {
				break
			}
			infoStr := string(lineData)
			index := strings.LastIndex(infoStr, ".ts")
			key := fmt.Sprintf("%s_%s_encode_%s.ts", self.Info.Profilename, self.Id, infoStr[:index])
			log.Infof("makeTslist set key=%s", key)
			self.TsIndexMap.Set(key, m3u8file)
		}
	}
	self.IsCheckDone = false
	self.IsM3u8Ready = true
	self.TSTotal = self.TsIndexMap.Count()

	return nil
}
