package wire

import (
	"fmt"
	"github.com/multivactech/MultiVAC/model/shard"
	"io"

	"github.com/multivactech/MultiVAC/base/rlp"
)

// TODO Add more logic
// TODO support TxWithProof

// MsgTxBatch is a message is specifically used to batch Txs.
type MsgTxBatch struct {
	ShardIndex shard.Index
	Txs        []MsgTx
}

// BtcDecode decode the message.
func (msg *MsgTxBatch) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
func (msg *MsgTxBatch) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the protocol command string for the message.
func (msg *MsgTxBatch) Command() string {
	return CmdTxBatch
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.
func (msg *MsgTxBatch) MaxPayloadLength(_ uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

func (msg *MsgTxBatch) String() string {
	return fmt.Sprintf("%s Index:%v, size:%d",
		msg.Command(), msg.ShardIndex, len(msg.Txs))
}

// GetShardIndex returns the shardIndex.
func (msg *MsgTxBatch) GetShardIndex() shard.Index {
	return msg.ShardIndex
}