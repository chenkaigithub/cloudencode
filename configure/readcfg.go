package configure

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/cloudencode/logging"
)

type EncodeConfigure struct {
	Httpport       int             `json:"httpport"`
	Pprofport      int             `json:"pprofport"`
	Tempdir        string          `json:"tempdir"`
	Enctempdir     string          `json:"enctempdir"`
	Hlsinterval    int             `json:"hlsinterval"`
	Bindir         string          `json:"bindir"`
	Osskeyid       string          `json:"osskeyid"`
	Osskeysec      string          `json:"osskeysec"`
	Ossendpoint    string          `json:"ossendpoint"`
	Ossbucket      string          `json:"ossbucket"`
	Redishost      string          `json:"redishost"`
	Redisport      int             `json:"redisport"`
	Redispwd       string          `json:"redispwd"`
	Procmax        int             `json:"procmax"`
	Encode_profile []EncodeProfile `json:"encode_profile"`
}

type EncodeProfile struct {
	Name          string `json:"name"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Profile       string `json:"profile"`
	Bitrate       int    `json:"bitrate"`
	Framerate     int    `json:"framerate"`
	AudioRate     int    `json:"ar"`
	AudioBitrate  int    `json:"ab"`
	AudioChannels int    `json:"ac"`
}

var EncodeCfgInfo EncodeConfigure

func ReadCfg(cfgfilename string) error {
	log.Info("start load configure file:", cfgfilename)

	data, err := ioutil.ReadFile(cfgfilename)
	if err != nil {
		log.Errorf("ReadFile %s error:%v", cfgfilename, err)
		return err
	}

	err = json.Unmarshal(data, &EncodeCfgInfo)
	if err != nil {
		log.Errorf("json.Unmarshal error:%v", err)
		return err
	}
	log.Infof("get config json data:%+v", EncodeCfgInfo)
	return nil
}
