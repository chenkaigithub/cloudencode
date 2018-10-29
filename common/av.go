package common

type EncodeInfo struct {
	Srcfile     string `json:"srcfile`
	Destsubdir  string `json:"destsubdir"`
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

type FileUploadI interface {
	UploadFile(filename string) (url string, err error)
	UploadFileEx(filename string, subpath string, outputFilename string) (url string, err error)
}
