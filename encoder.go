package main

import (
	"flag"

	"github.com/cloudencode/configure"
	"github.com/cloudencode/encmgr"
	"github.com/cloudencode/encmgr/mediaenc"
	"github.com/cloudencode/httphandle"
	log "github.com/cloudencode/logging"
	"github.com/cloudencode/redisoper"
	"github.com/cloudencode/uploadfile"
	"github.com/cloudencode/watchfiles"
)

var (
	loglevel = flag.String("l", "info", "log level")
	logfile  = flag.String("f", "logs/encode.log", "log file path")
	cfgfile  = flag.String("c", "conf/encode.json", "encode configure filename")
)

func init() {
	flag.Parse()
	log.SetOutputByName(*logfile)
	log.SetRotateByDay()
	log.SetLevelByString(*loglevel)
}

func main() {
	err := configure.ReadCfg(*cfgfile)
	if err != nil {
		return
	}
	encmgrObj := encmgr.NewEncMgr()

	uploadfileObj := uploadfile.NewUploadfile(configure.EncodeCfgInfo.Osskeyid,
		configure.EncodeCfgInfo.Osskeysec,
		configure.EncodeCfgInfo.Ossendpoint,
		configure.EncodeCfgInfo.Ossbucket)

	mediaEncObj := mediaenc.NewMediaEnc()

	redisObj := redisoper.NewRedisOper(configure.EncodeCfgInfo.Redishost,
		configure.EncodeCfgInfo.Redisport,
		configure.EncodeCfgInfo.Redispwd)
	watchfileObj := watchfiles.NewWatchFiles(configure.EncodeCfgInfo.Tempdir)

	mediaEncObj.SetUploadNotify(uploadfileObj)
	uploadfileObj.SetRedisNotify(redisObj)
	redisObj.SetEncodeNotify(mediaEncObj)
	watchfileObj.SetUploadNotify(uploadfileObj)
	encmgrObj.SetCheckDone(redisObj)
	encmgrObj.SetUploadNotify(uploadfileObj)

	watchfileObj.Start()
	uploadfileObj.Start()

	err = redisObj.Start()
	if err != nil {
		return
	}

	httpserver := httphandle.NewHttpHandle(configure.EncodeCfgInfo.Httpport, encmgrObj)

	httpserver.StartHttpServer()
}
