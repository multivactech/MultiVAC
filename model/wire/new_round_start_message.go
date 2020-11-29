// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2018-present, MultiVAC dev team.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package wire

import (
	"github.com/multivactech/MultiVAC/model/shard"
	"io"
	"time"

	"github.com/multivactech/MultiVAC/base/rlp"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

// NewRoundStartMessage is the message for starting new round.
type NewRoundStartMessage struct {
	ShardIndex  shard.Index
	Round       int
	Timestamp   time.Time
	Seed        chainhash.Hash
	Transaction *MsgTx `rlp:"nil"`
}

// BtcDecode decode the message.
func (msg *NewRoundStartMessage) BtcDecode(r io.Reader, _ uint32, _ MessageEncoding) error {
	return rlp.Decode(r, msg)
}

// BtcEncode encode the message.
func (msg *NewRoundStartMessage) BtcEncode(w io.Writer, _ uint32, _ MessageEncoding) error {
	return rlp.Encode(w, msg)
}

// Command returns the command string.
func (msg *NewRoundStartMessage) Command() string {
	return CmdNewRoundStart
}

// MaxPayloadLength returns the max payload length
func (msg *NewRoundStartMessage) MaxPayloadLength(_ uint32) uint32 {
	// TODO Change it to a proper value.
	return LargeMaxPayloadLength
}

// GetRound returns the round of the message.
func (msg *NewRoundStartMessage) GetRound() int {
	return msg.Round
}
