package watchfiles

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/cloudencode/common"
	log "github.com/cloudencode/logging"
	"github.com/fsnotify"
)

type WatchFiles struct {
	RootDir string
	notify  common.FileNotification
}

func NewWatchFiles(rootdir string) *WatchFiles {
	return &WatchFiles{
		RootDir: rootdir,
	}
}

func (self *WatchFiles) SetUploadNotify(notify common.FileNotification) {
	self.notify = notify
}

func (self *WatchFiles) Start() {
	go self.onWork()
}

func (self *WatchFiles) onWork() {
	Watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("NewWatch error")
		return
	}
	defer Watch.Close()

	self.walkAllDir(Watch, self.RootDir)
	for {
		select {
		case event := <-Watch.Events:
			log.Debugf("file event:%s, name:%s", event.Op, event.Name)
			if event.Op == fsnotify.Create {
				log.Debugf("Create: %s\r\n", event.Name)
				f, err := os.Stat(event.Name)
				if err == nil {
					if f.IsDir() == true {
						go func() {
							Watch.Add(event.Name)
							log.Warningf("Running Watch.Add %s\r\n", event.Name)
						}()
					}
				}
			} else if event.Op == fsnotify.WriteClose {
				f, err := os.Stat(event.Name)
				if err == nil {
					if !f.IsDir() {
						isTs := strings.Contains(event.Name, ".ts")
						if isTs {
							log.Debugf("writeclose filename:%s", event.Name)
							self.notify.Notify(event.Name)
						}
					}
				}
			} else if event.Op == fsnotify.Remove {
				isTs := strings.Contains(event.Name, ".ts")
				isM3u8 := strings.Contains(event.Name, ".m3u8")
				if !isTs && !isM3u8 {
					Watch.Remove(event.Name)
					log.Warningf("Running Watch.Remove %s...\r\n", event.Name)
				}
			}
		case err := <-Watch.Errors:
			log.Errorf("Watch Error %v", err)
		}
	}
}

func (self *WatchFiles) walkAllDir(Watch *fsnotify.Watcher, dirPth string) error {
	Watch.Add(dirPth)

	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		log.Errorf("ReadDir %s error:%v", dirPth, err)
		return err
	}

	for _, fi := range dir {
		if fi.IsDir() {
			filepath := dirPth + "/" + fi.Name()
			self.walkAllDir(Watch, filepath)
			log.Debugf("Watch.Add %s", filepath)
			Watch.Add(filepath)
		}
	}

	return nil
}
