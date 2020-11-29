package wire

import (
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// Timestamp defines the type of time.
type Timestamp int64

// HeartBeatMsg is a type of message heartbeat.
type HeartBeatMsg struct {
	Pk        []byte // the miner's public key
	TimeStamp Timestamp
	Proof     []byte // the proof of a mine
	Signature []byte // the signature of TimeStamp
}

// BtcDecode decode the message.
// TODO(jylu)
func (msg *HeartBeatMsg) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// Deserialize deserialize the data.
// TODO(jylu)
func (msg *HeartBeatMsg) Deserialize(r io.Reader) error {
	return msg.BtcDecode(r, 0, BaseEncoding)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// TODO(jylu)
func (msg *HeartBeatMsg) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Serialize will serialize the message.
// TODO(jylu)
func (msg *HeartBeatMsg) Serialize(w io.Writer) error {
	return msg.BtcEncode(w, 0, BaseEncoding)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *HeartBeatMsg) Command() string {
	//TODO(jylu)
	return CmdHeartBeat
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *HeartBeatMsg) MaxPayloadLength(pver uint32) uint32 {
	//TODO(jylu)
	return MaxBlockPayload
}
