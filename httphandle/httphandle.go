package httphandle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/cloudencode/common"
	log "github.com/cloudencode/logging"
)

type encodeResp struct {
	Ret  int    `json:"ret`
	Dscr string `json:"dscr`
	Id   string `json:"id`
}

const (
	START_ENC_OK    = 200
	ERROR_JSON_BODY = 501
	START_ENC_ERROR = 502
)

type HttpHandleServer struct {
	port   int
	writer common.Writer
}

func NewHttpHandle(port int, writer common.Writer) *HttpHandleServer {
	return &HttpHandleServer{
		port:   port,
		writer: writer,
	}
}

func (self *HttpHandleServer) StartHttpServer() error {
	httpFlvAddr := fmt.Sprintf(":%d", self.port)
	flvListen, err := net.Listen("tcp", httpFlvAddr)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/startenc", func(w http.ResponseWriter, r *http.Request) {
		self.startEncodeHandle(w, r)
	})
	mux.HandleFunc("/api/encstat", func(w http.ResponseWriter, r *http.Request) {
		self.queryStatHandle(w, r)
	})
	http.Serve(flvListen, mux)
	return nil
}

func (self *HttpHandleServer) encodeResponse(w http.ResponseWriter, ret int, dscr string, id string) {
	responseBody := &encodeResp{
		Ret:  ret,
		Dscr: dscr,
		Id:   id,
	}
	data, _ := json.Marshal(responseBody)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (self *HttpHandleServer) startEncodeHandle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("session start readall error:%v", err)
		self.encodeResponse(w, ERROR_JSON_BODY, "request body json error", "-1")
		return
	}
	defer r.Body.Close()

	var info common.EncodeInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		log.Errorf("json.Unmarshal error:%v", err)
		self.encodeResponse(w, ERROR_JSON_BODY, "request body json error", "-1")
		return
	}
	log.Infof("get config json data:%+v", info)
	id, err := self.writer.WriteMsg(&info)
	if err != nil {
		log.Errorf("json.Unmarshal error:%v", err)
		self.encodeResponse(w, START_ENC_ERROR, "start encoding error", "-1")
		return
	}
	self.encodeResponse(w, START_ENC_OK, "start encoding ok", id)

	return
}

func (self *HttpHandleServer) queryStatHandle(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	idList := req.Form["id"]
	if len(idList) == 0 {
		http.Error(w, "no id inputed", http.StatusBadRequest)
		return
	}
	ID := idList[0]

	info := self.writer.GetEncodeStatInfo(ID)

	retData, err := json.Marshal(info)
	if err != nil {
		log.Errorf("queryStatHandle json encode error:%v", err)
		http.Error(w, fmt.Sprintf("queryStatHandle json encode error:%v", err), http.StatusBadRequest)
		return
	}

	w.Write(retData)
	return
}
