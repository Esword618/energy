//----------------------------------------
//
// Copyright © yanghy. All Rights Reserved.
//
// Licensed under Apache License Version 2.0, January 2004
//
// https://www.apache.org/licenses/LICENSE-2.0
//
//----------------------------------------

package ipc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/energye/energy/common"
	. "github.com/energye/energy/consts"
	"github.com/energye/energy/logger"
	"github.com/energye/energy/pkgs/json"
	"github.com/energye/golcl/lcl/rtl/version"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
)

var (
	protocolHeader       = []byte{0x01, 0x09, 0x08, 0x07, 0x00, 0x08, 0x02, 0x02}                           //协议头
	protocolHeaderLength = int32(len(protocolHeader))                                                       //协议头长度
	messageTypeLength    = int32(1)                                                                         //消息类型 int8
	channelIdLength      = int32(8)                                                                         //通道Id int64
	dataByteLength       = int32(4)                                                                         //数据长度 int32
	headerLength         = int(protocolHeaderLength + messageTypeLength + channelIdLength + dataByteLength) //协议头长度
)

var (
	memoryAddress    = "energy.sock"
	ipcSock          string
	useNetIPCChannel = false
	Channel          = &ipcChannel{
		browser: &browserChannel{
			channel: sync.Map{},
			mutex:   sync.Mutex{},
		},
		render: &renderChannel{
			mutex: sync.Mutex{},
		},
	}
	ipcWriteBuf = new(bytes.Buffer)
)

//消息类型
type mt int8

const (
	mt_invalid    mt = iota - 1 //无效类型
	mt_connection               //建立链接消息
	mt_common                   //普通消息
)

const (
	key_channelId = "key_channelId"
)

type IPCCallback func(context IIPCContext)

func init() {
	ipcSock = filepath.Join(os.TempDir(), memoryAddress)
}

type ipcChannel struct {
	port    int
	browser *browserChannel
	render  *renderChannel
}

type channel struct {
	IPCType IPC_TYPE
	Conn    net.Conn
}

func removeMemory() {
	os.Remove(ipcSock)
}

func UseNetIPCChannel() bool {
	return useNetIPCChannel
}

func MemoryAddress() string {
	return memoryAddress
}

func isUseNetIPC() bool {
	if common.IsDarwin() || common.IsLinux() {
		return false
	}
	ov := version.OSVersion
	if (ov.Major > 10) || (ov.Major == 10 && ov.Build >= 17063) {
		return false
	}
	return true
}

// conn 返回通道链接
func (m *channel) conn() net.Conn {
	return m.Conn
}

// Port 获取并返回net socket端口
func (m *ipcChannel) Port() int {
	if m.port != 0 {
		return m.port
	}
	//主进程获取端口号
	if common.Args.IsMain() {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			panic("Failed to Get unused Port number Error: " + err.Error())
		}
		listen, err := net.ListenTCP("tcp", addr)
		if err != nil {
			panic("Failed to Get unused Port number Error: " + err.Error())
		}
		defer listen.Close()
		m.port = listen.Addr().(*net.TCPAddr).Port
	}
	return m.port
}

// Browser 返回 browser 通道
func (m *ipcChannel) Browser() *browserChannel {
	return m.browser
}

// Render 返回 render 通道
func (m *ipcChannel) Render() *renderChannel {
	return m.render
}

// IIPCChannel browser
type IIPCChannel interface {
	Close()
	Channel(channelId int64) *channel //IPC 获取指定的通道
	ChannelIds() (result []int64)     //IPC 获取所有通道
}

// IIPCContext IPC通信回调上下文
type IIPCContext interface {
	Connect() net.Conn // IPC 通道链接
	ChannelId() int64  // 通道ID
	Message() IMessage // 消息
	Free()             //
}

// IMessage 消息内容接口
type IMessage interface {
	Type() mt
	Length() uint32
	Data() []byte
	JSON() json.JSON
	clear()
}

// ipcReadHandler ipc 消息读取处理
type ipcReadHandler struct {
	ipcType IPC_TYPE
	ct      ChannelType
	connect net.Conn
	handler IPCCallback
}

// ipcMessage 消息内容
type ipcMessage struct {
	t mt
	s uint32
	v []byte
}

// IPCContext IPC 上下文
type IPCContext struct {
	channelId int64    //render channelId
	ipcType   IPC_TYPE //
	connect   net.Conn //
	message   IMessage //
}

// Close 关闭当前ipc通道链接
func (m *ipcReadHandler) Close() {
	if m.connect != nil {
		m.connect.Close()
		m.connect = nil
	}
}

// Read 读取内容
func (m *ipcReadHandler) Read(b []byte) (n int, err error) {
	if m.ipcType == IPCT_NET {
		return m.connect.Read(b)
	} else {
		n, _, err := m.connect.(*net.UnixConn).ReadFromUnix(b)
		return n, err
	}
}

// Free 释放消息内存空间
func (m *IPCContext) Free() {
	if m.message != nil {
		m.message.clear()
		m.message = nil
	}
}

// ChannelId 返回通道ID
func (m *IPCContext) ChannelId() int64 {
	return m.channelId
}

// Message 返回消息内容
func (m *IPCContext) Message() IMessage {
	return m.message
}

// Connect 返回当前通道链接
func (m *IPCContext) Connect() net.Conn {
	return m.connect
}

// Type 消息类型
func (m *ipcMessage) Type() mt {
	return m.t
}

// Data 消息[]byte数据
func (m *ipcMessage) Data() []byte {
	return m.v
}

// Length 消息[]byte长度
func (m *ipcMessage) Length() uint32 {
	return m.s
}

// JSON 消息转为JSON对象
func (m *ipcMessage) JSON() json.JSON {
	return json.NewJSON(m.v)
}

// clear 清空内容
func (m *ipcMessage) clear() {
	m.t = mt_invalid
	m.v = nil
	m.s = 0
}

// ipcWrite 写入消息
func ipcWrite(messageType mt, channelId int64, data []byte, conn net.Conn) (n int, err error) {
	defer func() {
		data = nil
	}()
	if conn == nil {
		return 0, errors.New("链接未建立成功")
	}
	var (
		dataByteLen = len(data)
	)
	if dataByteLen > math.MaxUint32 {
		return 0, errors.New("超出最大消息长度")
	}
	binary.Write(ipcWriteBuf, binary.BigEndian, protocolHeader)      //协议头
	binary.Write(ipcWriteBuf, binary.BigEndian, int8(messageType))   //消息类型
	binary.Write(ipcWriteBuf, binary.BigEndian, channelId)           //通道Id
	binary.Write(ipcWriteBuf, binary.BigEndian, uint32(dataByteLen)) //数据长度
	binary.Write(ipcWriteBuf, binary.BigEndian, data)                //数据
	n, err = conn.Write(ipcWriteBuf.Bytes())
	ipcWriteBuf.Reset()
	return n, err
}

// ipcRead 读取消息
func ipcRead(handler *ipcReadHandler) {
	var ipcType, chnType string
	if handler.ipcType == IPCT_NET {
		ipcType = "[net]"
	} else {
		ipcType = "[unix]"
	}
	if handler.ct == Ct_Server {
		chnType = "[server]"
	} else {
		chnType = "[client]"
	}
	defer func() {
		logger.Debug("IPC Read Disconnect type:", ipcType, "ChannelType:", chnType, "processType:", common.Args.ProcessType())
		handler.Close()
	}()
	for {
		header := make([]byte, headerLength)
		size, err := handler.Read(header)
		if err != nil {
			logger.Debug("IPC Read【Error】IPCType:", ipcType, "ChannelType:", chnType, "Error:", err)
			return
		} else if size == 0 {
			logger.Debug("IPC Read【Size == 0】IPCType:", ipcType, "ChannelType:", chnType, "header:", header, "Error:", err)
			return
		}
		if size == headerLength {
			for i, protocol := range protocolHeader {
				if header[i] != protocol {
					return
				}
			}
			var (
				t         int8 //消息类型
				channelId int64
				dataLen   uint32 //数据长度
				low, high int32  //
			)
			//消息类型
			low = protocolHeaderLength
			high = protocolHeaderLength + messageTypeLength
			err = binary.Read(bytes.NewReader(header[low:high]), binary.BigEndian, &t)
			if err != nil {
				logger.Debug("binary.Read.length: ", err)
				return
			}
			//通道ID
			low = high
			high = high + channelIdLength
			err = binary.Read(bytes.NewReader(header[low:high]), binary.BigEndian, &channelId)
			if err != nil {
				logger.Debug("binary.Read.length: ", err)
				return
			}

			//数据长度
			low = high
			high = high + dataByteLength
			err = binary.Read(bytes.NewReader(header[low:high]), binary.BigEndian, &dataLen)
			if err != nil {
				logger.Debug("binary.Read.length: ", err)
				return
			}
			//数据
			dataByte := make([]byte, dataLen)
			if dataLen > 0 {
				size, err = handler.Read(dataByte)
			}
			if err != nil {
				logger.Debug("binary.Read.data: ", err)
				return
			}
			handler.handler(&IPCContext{
				channelId: channelId,
				ipcType:   handler.ipcType,
				connect:   handler.connect,
				message: &ipcMessage{
					t: mt(t),
					s: dataLen,
					v: dataByte,
				},
			})
		} else {
			logger.Debug("无效的 != headerLength")
			break
		}
	}
}
