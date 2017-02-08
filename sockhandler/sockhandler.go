package sockhandler

import (
	"bytes"
	"fmt"
	"log"
	"minifileserver/filehandler"
	. "minifileserver/protocol"
	"net"
	"sync"

	. "github.com/ecofast/sysutils"
)

type ActiveConns struct {
	mutex sync.Mutex
	conns map[string]net.Conn
}

func (cs *ActiveConns) Initialize() {
	cs.conns = make(map[string]net.Conn)
}

func (cs *ActiveConns) Add(addr string, conn net.Conn) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.conns[addr] = conn
}

func (cs *ActiveConns) Remove(addr string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	delete(cs.conns, addr)
}

func (cs *ActiveConns) Exists(addr string) bool {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	_, ok := cs.conns[addr]
	return ok
}

func (cs *ActiveConns) Count() int {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	return len(cs.conns)
}

const (
	RecvBufLenMax = 16 * 1024
	SendBufLenMax = 32 * 1024
)

var (
	Conns       ActiveConns
	FileHandler filehandler.FileHandler
)

func Run(port int, filepath string) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+IntToStr(port))
	CheckError(err)
	defer listener.Close()

	log.Println("=====文件下载服务器已启动=====")

	FileHandler.Initialize(filepath)
	Conns.Initialize()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting: %s\n", err.Error())
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	Conns.Add(conn.RemoteAddr().String(), conn)
	log.Printf("当前连接数：%d\n", Conns.Count())

	var msg Msg
	var recvBuf []byte
	recvBufLen := 0
	buf := make([]byte, MsgSize)
	for {
		count, err := conn.Read(buf)
		if err != nil {
			Conns.Remove(conn.RemoteAddr().String())
			conn.Close()
			log.Println("连接断开：", err.Error())
			log.Printf("[handleConn] 当前连接数：%d\n", Conns.Count())
			break
		}

		if count+recvBufLen > RecvBufLenMax {
			continue
		}

		recvBuf = append(recvBuf, buf[0:count]...)
		recvBufLen += count
		offsize := 0
		offset := 0
		for recvBufLen-offsize >= MsgSize {
			offset = 0
			msg.Signature = uint32(uint32(recvBuf[offsize+3])<<24 | uint32(recvBuf[offsize+2])<<16 | uint32(recvBuf[offsize+1])<<8 | uint32(recvBuf[offsize+0]))
			offset += 4
			msg.Cmd = uint16(uint16(recvBuf[offsize+offset+1])<<8 | uint16(recvBuf[offsize+offset+0]))
			offset += 2
			msg.Param = int16(int16(recvBuf[offsize+offset+1])<<8 | int16(recvBuf[offsize+offset+0]))
			offset += 2
			copy(msg.FileName[:], recvBuf[offsize+offset+0:offsize+offset+MaxFileNameLen])
			offset += MaxFileNameLen
			msg.Len = int32(int32(recvBuf[offsize+offset+3])<<24 | int32(recvBuf[offsize+offset+2])<<16 | int32(recvBuf[offsize+offset+1])<<8 | int32(recvBuf[offsize+offset+0]))
			offset += 4
			if msg.Signature == CustomSignature {
				pkglen := int(MsgSize + msg.Len)
				if pkglen >= RecvBufLenMax {
					offsize = recvBufLen
					break
				}
				if offsize+pkglen > recvBufLen {
					break
				}

				switch msg.Cmd {
				case CM_PING:
					fmt.Printf("From %s received CM_PING\n", conn.RemoteAddr().String())
					reponsePing(conn)
				case CM_GETFILE:
					fmt.Printf("From %s received CM_GETFILE\n", conn.RemoteAddr().String())
					responseDownloadFile( /*string(msg.FileName[:])*/ msg.FileName, conn)
				default:
					fmt.Printf("From %s received %d\n", conn.RemoteAddr().String(), msg.Cmd)
				}

				offsize += pkglen
			} else {
				offsize++
				fmt.Printf("From %s received %d\n", conn.RemoteAddr().String(), msg.Cmd)
			}
		}

		recvBufLen -= offsize
		if recvBufLen > 0 {
			recvBuf = recvBuf[offsize : offsize+recvBufLen]
		} else {
			recvBuf = nil
		}
	}

	conn.Close()
}

func reponsePing(conn net.Conn) {
	var msg Msg
	msg.Signature = CustomSignature
	msg.Cmd = SM_PING
	msg.Param = 0
	msg.FileName = [MaxFileNameLen]byte{0}
	msg.Len = 0
	conn.Write(msg.Bytes())
}

func responseDownloadFile(filename [MaxFileNameLen]byte, conn net.Conn) {
	var msg Msg
	msg.Signature = CustomSignature
	msg.Cmd = SM_GETFILE
	msg.FileName = filename
	var buf bytes.Buffer
	if data, err := FileHandler.GetFile(BytesToStr(filename[:])); err == nil {
		msg.Param = 0
		msg.Len = int32(len(data))
		buf.Write(msg.Bytes())
		buf.Write(data)
	} else {
		log.Println(err.Error())
		msg.Param = -1
		msg.Len = 0
		buf.Write(msg.Bytes())
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		log.Printf("Write to %s failed: %s", conn.RemoteAddr().String(), err.Error())
	}
}
