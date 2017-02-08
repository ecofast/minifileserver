package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	MaxFileNameLen = 24
)

type Msg struct {
	Signature uint32
	Cmd       uint16
	Param     int16
	FileName  [MaxFileNameLen]byte
	Len       int32
}

func (m *Msg) String() string {
	return fmt.Sprintf("Signature:%d Cmd:%d Param:%d FileName:%s Len:%d", m.Signature, m.Cmd, m.Param, m.FileName, m.Len)
}

func (m *Msg) Bytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, m)
	return buf.Bytes()
}

const (
	MsgSize         = 4 + 2 + 2 + 24 + 4
	CustomSignature = 0xFAFBFCFD

	CM_PING = 100
	SM_PING = 200

	CM_GETFILE = 1000
	SM_GETFILE = 2000
)
