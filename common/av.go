package common

type EncodeInfo struct {
	Srcfile     string `json:"srcfile`
	Destfile    string `json:"destfile`
	Profilename string `json:"profilename`
}

type EncodeFileInfo struct {
	Filename    string
	Id          string
	Profilename string
	Timestamp   int64
}
type Writer interface {
	WriteMsg(info *EncodeInfo) (string, error)
}

type FileNotification interface {
	Notify(filename string) error
}

type EncodeNotification interface {
	EncodeNotify(info *EncodeFileInfo) error
}

type EncodedCheckI interface {
	IsEncodedDone(key string) bool
}
