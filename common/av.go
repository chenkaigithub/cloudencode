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

const RET_OK = 200
const RET_ID_NOT_EXIST = 501

const (
	ENCODE_NOT_EXIST = iota
	ENCODE_INIT
	ENCODE_TS_ENCODING
	ENCODE_TS_ENODE_DONE
	ENCODE_MP4_DONE
)

var ENCODE_STATUS_DSCR = []string{"id not exist", "encode init", "encoding ts files", "encode ts done", "encode mp4 done"}

type EncodeStatInfo struct {
	Code        int    `json:"code`
	Id          string `json:"id`
	Statuscode  int    `json:"statuscode`
	Dscr        string `json:"dscr`
	Tstotal     int    `json:"tstotal`
	Tsleftcount int    `json:"tsleftcount`
	Starttime   int64  `json:"starttime`
	Endtime     int64  `json:"endtime`
	Costtime    int64  `json:"costtime`
}

type Writer interface {
	WriteMsg(info *EncodeInfo) (string, error)
	GetEncodeStatInfo(ID string) (info *EncodeStatInfo)
}

type FileNotification interface {
	Notify(filename string) error
}

type EncodeNotification interface {
	EncodeNotify(info *EncodeFileInfo) error
}

type EncodedCheckI interface {
	IsEncodedDone(key string) bool
	UpdateEncodeStat(info *EncodeStatInfo) error
	GetEncodeStat(Id string) (info *EncodeStatInfo, err error)
}

type FileUploadI interface {
	UploadFile(filename string) (url string, err error)
	UploadFileEx(filename string, subpath string, outputFilename string) (url string, err error)
}
