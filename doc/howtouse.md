<!-- TOC -->

- [1. 使用说明](#1-使用说明)
    - [1.1. 配置文件](#11-配置文件)
        - [1.1.1. 网络配置](#111-网络配置)
        - [1.1.2. 文件目录配置](#112-文件目录配置)
        - [1.1.3. 阿里OSS配置](#113-阿里oss配置)
        - [1.1.4. redis配置](#114-redis配置)
        - [1.1.5. ts切片时长配置](#115-ts切片时长配置)
        - [1.1.6. 转码profile配置](#116-转码profile配置)
    - [1.2. 启动转码](#12-启动转码)
    - [1.3. 查看转码状态](#13-查看转码状态)

<!-- /TOC -->
# 1. 使用说明
## 1.1. 配置文件
<pre>
{
  "httpport":8000,
  "pprofport":9000,
  "tempdir":"xxxx/temp",
  "enctempdir":"xxxx/enctemp",
  "bindir":"xxxx/bin",
  "hlsinterval":10,
  "osskeyid":"xxxx",
  "osskeysec":"xxxx",
  "ossendpoint":"xxxx",
  "ossbucket":"xxxx",
  "redishost":"127.0.0.1",
  "redisport":6379,
  "redispwd":"Rsmt@123456",
  "encode_profile":[
    {
      "name":"360p",
      "width": 640,
      "height": 360,
      "profile": "baseline",
      "bitrate": 800,
      "framerate": 30,
      "ar": 44100,
      "ab": 64,
      "ac": 2
    }
    ....
  ]
}
</pre>
### 1.1.1. 网络配置
  "httpport":8000, //http服务的端口号，转码业务使用<br/>
  "pprofport":9000, //程序性能监控端口号，仅仅对go pprof监控使用

### 1.1.2. 文件目录配置
  "tempdir":"xxxx/temp", //切片临时存放目录<br/>
  "enctempdir":"xxxx/enctemp", //切编码后临时存放目录<br/>
  "bindir":"xxxx/bin", //ffmpeg程序路径目录<br/>

### 1.1.3. 阿里OSS配置
oss用于存储:
* 转码前的分片ts文件
* 转码后的分片ts文件
* 最终合成的mp4结果文件
  
  "osskeyid":"xxxx", //阿里OSS keyid<br/>
  "osskeysec":"xxxx",//阿里OSS keysec<br/>
  "ossendpoint":"xxxx",//阿里OSS endpoint<br/>
  "ossbucket":"xxxx",  //阿里OSS 分配bucket<br/>

### 1.1.4. redis配置
redis主要用于任务发布，将需要转码的ts文件，放入redis消息队列。<br/>
多个服务器自行消费redis消息队列，启动各自独立的转码工作<br>
  "redishost":"127.0.0.1",//redis服务地址<br/>
  "redisport":6379,//redis服务端口号<br/>
  "redispwd":"Rsmt@123456",//redis服务密码<br/>

### 1.1.5. ts切片时长配置
"hlsinterval":10,//对大视频文件切片，每个ts切片的时长设置，单位：秒

### 1.1.6. 转码profile配置
该profile可以配置多个，数量不限，每个profile的name必须保证唯一。<br/>
<pre>
"encode_profile":[
    {
      "name":"360p",//转码名，必须保证名字字符的唯一性
      "width": 640,//转码后的视频宽度
      "height": 360,//转码后的视频高度
      "profile": "baseline",//转码后x264 profile
      "bitrate": 800,//转码后视频bitrate，单位kbps
      "framerate": 30,//转码后视频framerate，单位帧个数/s
      "ar": 44100,//音频采样率
      "ab": 64,//音频bitrate
      "ac": 2 //音频通道数
    }
</pre>
<br/>

## 1.2. 启动转码
通过发送http消息启动转码:<br/>
* http方式: post
* http路径: /api/startenc
* http post数据格式: json

具体post数据内容如下: <br/>
<pre>
curl -d '{"srcfile":"/a8root/work/service/media.encoder.server/src.flv", "destsubdir":"encoded","destfile":"huangfeiyong_540p.mp4", "profilename":"540p"}' http://10.111.13.140:8000/api/startenc
</pre>
其中:
* srcfile: 转码的源视频文件，可以是http地址的媒体文件
* destsubdir: 最终放入oss中的子路径
* destfile: 转码完成后的文件名
* profilename: 按照配置中的哪个profile来进行转码
  
转码完成后，文件生成在http://bucketname.endpoint/destsubdir/destfile。<br/>

http返回数据:
<pre>
{
    "Ret":200,
    "Dscr":"start encoding ok",
    "Id":"TdefLR2lTAeQbi7p1808"
}
</pre>
* Ret: 返回值，200为正确，其余为错误
* Dscr: 返回描述
* Id: 转码任务的唯一标识，可以用来查询转码状态
  
## 1.3. 查看转码状态
通过发送http消息查看转码状态:<br/>
* http方式: get
* http路径: /api/encstat
* http get参数: id=TdefLR2lTAeQbi7p1808

其中参数id为启动转码中返回的id值。