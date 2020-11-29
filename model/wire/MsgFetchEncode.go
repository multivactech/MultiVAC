package wire

import (
	"encoding/gob"
	"io"

	"github.com/multivactech/MultiVAC/model/chaincfg/multivacaddress"
)

// MsgFetchEncode is used to encode a fetch msg.
type MsgFetchEncode struct {
	Msg         Message
	ShardHeight int64
	UserAddress multivacaddress.Address
}

// BtcDecode decode the message.
func (m *MsgFetchEncode) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(m)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (m *MsgFetchEncode) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	encoder := gob.NewEncoder(w)
	return encoder.Encode(*m)
}
