package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

const (
	// MsgReSendSyncReqMaxPayload is a very arbitrary number
	MsgReSendSyncReqMaxPayload = 4000000
)

// MsgReSendSyncReq 用于同步过程中重新发送同步请求
type MsgReSendSyncReq struct {
}

// NewReSendSyncReq create a resend sync request message.
func NewReSendSyncReq() *MsgReSendSyncReq {
	msg := MsgReSendSyncReq{}
	return &msg
}

// BtcDecode decode the message.
func (msg *MsgReSendSyncReq) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgReSendSyncReq) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgReSendSyncReq) Command() string {
	return CmdReSendSyncreq
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgReSendSyncReq) MaxPayloadLength(uint32) uint32 {
	return MsgReSendSyncReqMaxPayload
}
