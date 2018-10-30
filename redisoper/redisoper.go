package redisoper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudencode/common"
	log "github.com/cloudencode/logging"
	"github.com/garyburd/redigo/redis"
)

type RedisOper struct {
	hostname     string
	port         int
	password     string
	redispool    *redis.Pool
	chanFilename chan string
	encNotify    common.EncodeNotification
}

type EncItem struct {
	Profilename string `json:"profilename"`
	Id          string `json:"id"`
	Url         string `json:"url"`
	Timestamp   string `json:"timestamp"`
}

const MaxIdle = 100
const IdleTimeout = 300000
const MaxActive = 5000
const ConnectTimeout = 5000
const ReadTimeout = 0
const WriteTimeout = 0

const LPUSH_KEY = "origin_file"

func NewRedisOper(hostname string, port int, password string) *RedisOper {
	return &RedisOper{
		hostname:     hostname,
		port:         port,
		password:     password,
		redispool:    nil,
		chanFilename: make(chan string, MaxActive-1),
	}
}

func (self *RedisOper) SetEncodeNotify(encNotify common.EncodeNotification) {
	self.encNotify = encNotify
}

func (self *RedisOper) getFileinfo(filename string) (profileName string, idString string, err error) {
	infolist := strings.Split(filename, "/")
	lastindex := len(infolist) - 1
	if len(infolist) < 2 {
		log.Errorf("redis get filename(%s) error", filename)
		err = fmt.Errorf("filename(%s) error", filename)
		return
	}
	tsPos := strings.LastIndex(filename, ".")
	if tsPos <= 0 {
		log.Errorf("redis get filename(%s) error", filename)
		err = fmt.Errorf("filename(%s) error", filename)
		return
	}
	if filename[tsPos+1:] != "ts" {
		log.Warningf("redis get filename(%s) not mpegts", filename)
		err = fmt.Errorf("redis get filename(%s) not mpegts", filename)
		return
	}
	tempinfo := infolist[lastindex-1]
	tempIndex := strings.Index(tempinfo, "_")
	if tempIndex <= 0 {
		log.Errorf("RedisOper get error filename:%s", tempinfo)
		err = fmt.Errorf("RedisOper get error filename:%s", tempinfo)
		return
	}
	profileName = tempinfo[:tempIndex]
	idString = tempinfo[tempIndex+1:]
	log.Infof("redisopen get profilename=%s, id=%s", profileName, idString)
	return
}

func (self *RedisOper) onwork() {
	log.Info("redisoper onwork is starting....")
	for {
		filename := <-self.chanFilename
		log.Infof("RedisOper get filename=%s", filename)
		profileName, idString, err := self.getFileinfo(filename)
		if err != nil {
			continue
		}

		if self.isEncodedFile(idString) {
			self.doSet(filename, profileName, idString)
		} else {
			self.doLpush(filename, profileName, idString)
		}
	}
}

func (self *RedisOper) doSet(filename string, profileName string, idString string) (err error) {
	err = fmt.Errorf("redis set error: filename=%s", filename)
	for index := 0; index < 3; index++ {
		err := self.setValue(filename, profileName, idString)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (self *RedisOper) isEncodedFile(idStr string) bool {
	pos := strings.LastIndex(idStr, "_")
	if pos <= 0 {
		return false
	}

	dscr := idStr[pos+1:]
	if dscr == "encode" {
		return true
	}
	return false
}

func (self *RedisOper) doLpush(filename string, profileName string, idString string) (err error) {
	err = fmt.Errorf("redis lpush error: filename=%s", filename)
	for index := 0; index < 3; index++ {
		err := self.lpush(filename, profileName, idString)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return
}

func (self *RedisOper) onPopWork() {
	log.Info("redisoper rpop work is starting....")
	for {
		filename, id, profilename, timestamp, err := self.rpop()
		if err != nil {
			continue
		}
		log.Infof("RedisOper rpop: filename=%s, id=%s, profilename=%s, timestamp=%d",
			filename, id, profilename, timestamp)
		info := &common.EncodeFileInfo{
			Filename:    filename,
			Id:          id,
			Profilename: profilename,
			Timestamp:   timestamp,
		}
		self.encNotify.EncodeNotify(info)
	}
}

func (self *RedisOper) rpop() (filename string, id string, profilename string, timestamp int64, err error) {
	var value interface{}

	client := self.redispool.Get()
	if client == nil {
		log.Error("redis rpop get client error")
		return "", "", "", 0, errors.New("redis pool get client error")
	}
	defer client.Close()

	value, err = client.Do("BRPOP", LPUSH_KEY, 60)
	if err != nil {
		log.Errorf("reids rpop get brpop(%s) error:%v", LPUSH_KEY, err)
		return "", "", "", 0, err
	}

	infolist, retErr := redis.Strings(value, err)
	if retErr != nil {
		log.Errorf("redisoper rpop error:%v", retErr)
		err = retErr
		return
	}

	log.Infof("rpop value:%v", infolist)

	if len(infolist) != 2 {
		err = fmt.Errorf("redis rpop decoder error: string=%v", infolist)
		log.Errorf("redis rpop decoder error: string=%v", infolist)
		return
	}
	dataStr := infolist[1]
	log.Infof("redis rpop: data=%s", dataStr)
	var item EncItem

	err = json.Unmarshal([]byte(dataStr), &item)
	if err != nil {
		log.Errorf("RedisOper rpop json decode error:%v, value=%s", err, dataStr)
		return
	}

	filename = item.Url
	id = item.Id
	profilename = item.Profilename
	ret, retErr := strconv.ParseInt(item.Timestamp, 10, 64)
	if retErr == nil {
		timestamp = int64(ret)
	} else {
		timestamp = 0
	}

	err = nil
	log.Infof("filename=%s, id=%s, profilename=%s, timestamp=%d", filename, id, profilename, timestamp)

	return
}

func (self *RedisOper) IsEncodedDone(key string) bool {
	client := self.redispool.Get()
	if client == nil {
		return false
	}
	defer client.Close()

	value, err := client.Do("GET", key)
	if err != nil || value == nil {
		//log.Errorf("redis get error:%v, key=%s", err, key)
		return false
	}
	log.Infof("redis get key(%s) ok", key)
	return true
}

func (self *RedisOper) UpdateEncodeStat(info *common.EncodeStatInfo) error {
	client := self.redispool.Get()
	if client == nil {
		return errors.New("get redis client error")
	}
	defer client.Close()

	data, err := json.Marshal(info)
	if err != nil {
		log.Errorf("redis UpdateEncodeStat error:%v", err)
		return err
	}

	dataStr := string(data)
	timeoutString := fmt.Sprintf("%d", 60*60)
	log.Infof("redis UpdateEncodeStat key=%s, value=%s, timeout=%s", info.Id, dataStr, timeoutString)
	_, err = client.Do("SETEX", info.Id, timeoutString, dataStr)
	if err != nil {
		log.Errorf("redis UpdateEncodeStat set error:%v", err)
		return err
	}
	return nil
}

func (self *RedisOper) GetEncodeStat(Id string) (info *common.EncodeStatInfo, err error) {
	err = nil
	info = &common.EncodeStatInfo{
		Code:        common.RET_ID_NOT_EXIST,
		Id:          Id,
		Statuscode:  common.ENCODE_NOT_EXIST,
		Dscr:        common.ENCODE_STATUS_DSCR[common.ENCODE_NOT_EXIST],
		Tstotal:     0,
		Tsleftcount: 0,
	}

	client := self.redispool.Get()
	if client == nil {
		log.Error("GetEncodeStat get redis client error")
		err = errors.New("get redis client error")
		return
	}
	defer client.Close()

	value, err := client.Do("GET", Id)
	if err != nil || value == nil {
		log.Errorf("redis GetEncodeStat get error:%v, key=%s", err, Id)
		return
	}
	log.Infof("redis get key(%s) return ", Id)

	var retStr string
	retStr, err = redis.String(value, err)
	if err != nil {
		log.Errorf("redis GetEncodeStat get error:%v, key=%s", err, Id)
		return
	}

	log.Infof("redis GetEncodeStat key=%s, value=%s", Id, retStr)
	data := []byte(retStr)
	err = json.Unmarshal(data, info)
	if err != nil {
		log.Errorf("redis GetEncodeStat json decode error:%v, key=%s", err, Id)
		return
	}
	log.Infof("redis GetEncodeStat key=%s, info=%+v", Id, info)
	return
}

func (self *RedisOper) setValue(filename string, profileName string, idString string) error {
	client := self.redispool.Get()
	if client == nil {
		return errors.New("redis pool get client error")
	}
	defer client.Close()

	pos := strings.LastIndex(filename, "/")
	if pos <= 0 {
		return fmt.Errorf("redis set error: filename(%s) invalid", filename)
	}
	tsName := filename[pos+1:]
	key := fmt.Sprintf("%s_%s_%s", profileName, idString, tsName)
	value := filename
	timeoutString := fmt.Sprintf("%d", 60*30)
	_, err := client.Do("SETEX", key, timeoutString, value)
	if err != nil {
		log.Errorf("redis set error:%v", err)
	}

	log.Infof("redis set key=%s, value=%s", key, value)
	return err
}

func (self *RedisOper) lpush(filename string, profilename string, id string) error {
	client := self.redispool.Get()
	if client == nil {
		return errors.New("redis pool get client error")
	}
	defer client.Close()

	nowT := time.Now().UnixNano() / (1000 * 1000)
	value := fmt.Sprintf("{\"profilename\":\"%s\", \"id\":\"%s\", \"url\":\"%s\", \"timestamp\":\"%d\"}",
		profilename, id, filename, nowT)
	_, err := client.Do("LPUSH", LPUSH_KEY, value)
	log.Infof("lpush key=%s, value=%s, err=%v", LPUSH_KEY, value, err)

	return err
}

func (self *RedisOper) Start() error {
	server := fmt.Sprintf("%s:%d", self.hostname, self.port)
	self.redispool = self.redisPoolInit(server, self.password)

	if self.redispool == nil {
		log.Error("redis init pool error")
		return errors.New("redis init pool error")
	}

	log.Infof("redisoper start ok: server=%s, password=%s", server, self.password)
	go self.onwork()
	go self.onPopWork()
	return nil
}

func (self *RedisOper) Notify(filename string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("RedisOper Notify panic:%v, filename:%s", r, filename)
		}
	}()
	self.chanFilename <- filename
	return nil
}

func (self *RedisOper) redisPoolInit(server string, password string) *redis.Pool {
	opts := []redis.DialOption{}
	if ConnectTimeout > 0 {
		opts = append(opts, redis.DialConnectTimeout(time.Duration(ConnectTimeout)*time.Millisecond))
	}

	if ReadTimeout > 0 {
		opts = append(opts, redis.DialReadTimeout(time.Duration(ReadTimeout)*time.Millisecond))
	}
	if WriteTimeout > 0 {
		opts = append(opts, redis.DialWriteTimeout(time.Duration(WriteTimeout)*time.Millisecond))
	}

	return &redis.Pool{
		MaxIdle:     MaxIdle,
		IdleTimeout: time.Duration(IdleTimeout) * time.Second,
		MaxActive:   MaxActive,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server, opts...)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
