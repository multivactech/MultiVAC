// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2013-2016 The btcsuite developers

package wire

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/prometheus/common/log"

	"github.com/multivactech/MultiVAC/model/shard"

	"github.com/multivactech/MultiVAC/model/chaincfg/chainhash"
)

// MessageHeaderSize is the number of bytes in a bitcoin message header.
// Bitcoin network (magic) 4 bytes + command 12 bytes + payload length 4 bytes +
// checksum 4 bytes.
const MessageHeaderSize = 24

// CommandSize is the fixed size of all commands in the common bitcoin message
// header.  Shorter commands must be zero padded.
const CommandSize = 12

// MaxMessagePayload is the maximum bytes a message can be regardless of other
// individual limits imposed by messages themselves.
const MaxMessagePayload = (1024 * 1024 * 32) // 32MB

// Commands used in bitcoin message headers which describe the type of message.
const (
	// Cmd string size limit is 12!!!
	CmdVersion                = "version"
	CmdVerAck                 = "verack"
	CmdGetAddr                = "getaddr"
	CmdAddr                   = "addr"
	CmdBlock                  = "block"
	CmdBlockHeader            = "blockheader"
	CmdSlimBlock              = "slimblock"
	CmdTx                     = "tx"
	CmdTxWithProofs           = "txwithproofs"
	CmdHeartBeat              = "heartbeat"
	CmdRelayHash              = "relayhash"
	CmdFetchTxs               = "fetchtxs"
	CmdReturnTxs              = "returntxs"
	CmdFetchDeposit           = "fetchdeposit"
	CmdInitAbciData           = "initabcdata"
	CmdReturnInit             = "returninit"
	CmdReturnDeposit          = "returndep"
	CmdPing                   = "ping"
	CmdPong                   = "pong"
	CmdAlert                  = "alert"
	CmdFilterAdd              = "filteradd"
	CmdFilterClear            = "filterclear"
	CmdFilterLoad             = "filterload"
	CmdReject                 = "reject"
	CmdSendHeaders            = "sendheaders"
	CmdFeeFilter              = "feefilter"
	CmdNewRoundStart          = "new_round"
	CmdLeaderProposal         = "lead_prevote"
	CmdLeaderVote             = "lead_commit"
	CmdGC                     = "gc"
	CmdBinaryBA               = "bba"
	CmdBinaryBAFin            = "bbafin"
	CmdMsgBlockConfirmation   = "blockconfrm"
	CmdSeed                   = "seed"
	CmdStartNet               = "startnet"
	CmdGetShardAddr           = "getshardaddr"
	CmdReturnAddr             = "returnaddr"
	CmdUpdatePeerInfo         = "update_peer"
	CmdTxBatch                = "tx_batch"
	CmdSmartContractInfo      = "smartCInfo"
	CmdFetchSmartContractInfo = "fetchSCInfo"

	// Sync relevant messages
	CmdSyncReq       = "syncreq"
	CmdSyncInv       = "syncinv"
	CmdReSendSyncreq = "resendsync"
	CmdSyncBlock     = "syncblock"
	CmdSyncSlimBlock = "syncslblock"
	CmdSyncHeaders   = "syncheaders"
)

// MessageEncoding represents the wire message encoding format to be used.
type MessageEncoding uint32

const (
	// BaseEncoding encodes all messages in the default format specified
	// for the Bitcoin wire protocol.
	BaseEncoding MessageEncoding = 1 << iota

	// WitnessEncoding encodes all messages other than transaction messages
	// using the default Bitcoin wire protocol specification. For transaction
	// messages, the new encoding format detailed in BIP0144 will be used.
	WitnessEncoding
)

// LatestEncoding is the most recently specified encoding for the Bitcoin wire
// protocol.
var LatestEncoding = WitnessEncoding

// Message is an interface that describes a bitcoin message.  A type that
// implements Message has complete control over the representation of its data
// and may therefore contain additional or fewer fields than those which
// are used directly in the protocol encoded message.
type Message interface {
	BtcDecode(io.Reader, uint32, MessageEncoding) error
	BtcEncode(io.Writer, uint32, MessageEncoding) error
	Command() string
	MaxPayloadLength(uint32) uint32
}

// ShardMessage defines the shard message interface to describe the message about shard.
type ShardMessage interface {
	GetShardIndex() shard.Index
}

// makeEmptyMessage creates a message of the appropriate concrete type based
// on the command.
func makeEmptyMessage(command string) (Message, error) {
	var msg Message
	switch command {
	case CmdVersion:
		msg = &MsgVersion{}

	case CmdVerAck:
		msg = &MsgVerAck{}

	case CmdGetAddr:
		msg = &MsgGetAddr{}

	case CmdAddr:
		msg = &MsgAddr{}

	case CmdBlock:
		msg = &MsgBlock{}

	case CmdBlockHeader:
		msg = &BlockHeader{}

	case CmdTx:
		msg = &MsgTx{}

	case CmdTxBatch:
		msg = &MsgTxBatch{}

	case CmdTxWithProofs:
		msg = &MsgTxWithProofs{}

	case CmdPing:
		msg = &MsgPing{}

	case CmdPong:
		msg = &MsgPong{}

	case CmdAlert:
		msg = &MsgAlert{}

	case CmdFilterAdd:
		msg = &MsgFilterAdd{}

	case CmdFilterClear:
		msg = &MsgFilterClear{}

	case CmdFilterLoad:
		msg = &MsgFilterLoad{}

	case CmdReject:
		msg = &MsgReject{}

	case CmdSendHeaders:
		msg = &MsgSendHeaders{}

	case CmdFeeFilter:
		msg = &MsgFeeFilter{}

	case CmdLeaderProposal:
		msg = &LeaderProposalMessage{}

	case CmdLeaderVote:
		msg = &LeaderVoteMessage{}

	case CmdGC:
		msg = &GCMessage{}

	case CmdUpdatePeerInfo:
		msg = &UpdatePeerInfoMessage{}

	case CmdSeed:
		msg = &MsgSeed{}

	case CmdBinaryBA:
		msg = &MsgBinaryBA{}

	case CmdBinaryBAFin:
		msg = &MsgBinaryBAFin{}

	case CmdMsgBlockConfirmation:
		msg = &MsgBlockConfirmation{}

	case CmdFetchTxs:
		msg = &MsgFetchTxs{}

	case CmdReturnTxs:
		msg = &MsgReturnTxs{}

	case CmdInitAbciData:
		msg = &MsgFetchInit{}

	case CmdFetchDeposit:
		msg = &MsgFetchDeposit{}

	case CmdHeartBeat:
		msg = &HeartBeatMsg{}

	case CmdReturnInit:
		msg = &MsgReturnInit{}

	case CmdReturnDeposit:
		msg = &MsgReturnDeposit{}

	case CmdSyncReq:
		msg = &MsgSyncReq{}

	case CmdSyncInv:
		msg = &MsgSyncInv{}

	case CmdSyncBlock:
		msg = &MsgSyncBlock{}

	case CmdSyncSlimBlock:
		msg = &MsgSyncSlimBlock{}

	case CmdSyncHeaders:
		msg = &MsgSyncHeaders{}

	case CmdSlimBlock:
		msg = &SlimBlock{}

	case CmdFetchSmartContractInfo:
		msg = &MsgFetchSmartContractInfo{}

	case CmdSmartContractInfo:
		msg = &SmartContractInfo{}

	case CmdRelayHash:
		msg = &RelayHashMsg{}

	case CmdGetShardAddr:
		msg = &MsgGetShardAddr{}

	case CmdReturnAddr:
		msg = &MsgReturnShardAddr{}

	case CmdStartNet:
		msg = &MsgStartNet{}

	default:
		return nil, fmt.Errorf("unhandled command [%s]", command)
	}
	return msg, nil
}

// messageHeader defines the header structure for all bitcoin protocol messages.
type messageHeader struct {
	magic    BitcoinNet // 4 bytes
	command  string     // 12 bytes
	length   uint32     // 4 bytes
	checksum [4]byte    // 4 bytes
}

// readMessageHeader reads a bitcoin message header from r.
func readMessageHeader(r io.Reader) (int, *messageHeader, error) {
	// Since readElements doesn't return the amount of bytes read, attempt
	// to read the entire header into a buffer first in case there is a
	// short read so the proper amount of read bytes are known.  This works
	// since the header is a fixed size.
	var headerBytes [MessageHeaderSize]byte
	n, err := io.ReadFull(r, headerBytes[:])
	if err != nil {
		return n, nil, err
	}
	hr := bytes.NewReader(headerBytes[:])

	// Create and populate a messageHeader struct from the raw header bytes.
	hdr := messageHeader{}
	var command [CommandSize]byte
	err = readElements(hr, &hdr.magic, &command, &hdr.length, &hdr.checksum)
	if err != nil {
		log.Errorf("failed to read elements,err:%v", err)
	}
	// Strip trailing zeros from command string.
	hdr.command = string(bytes.TrimRight(command[:], string(0)))

	return n, &hdr, nil
}

// discardInput reads n bytes from reader r in chunks and discards the read
// bytes.  This is used to skip payloads when various errors occur and helps
// prevent rogue nodes from causing massive memory allocation through forging
// header length.
func discardInput(r io.Reader, n uint32) {
	maxSize := uint32(10 * 1024) // 10k at a time
	numReads := n / maxSize
	bytesRemaining := n % maxSize
	if n > 0 {
		buf := make([]byte, maxSize)
		for i := uint32(0); i < numReads; i++ {
			_, err := io.ReadFull(r, buf)
			if err != nil {
				log.Errorf("failed to read from reader,err:%v", err)
			}

		}
	}
	if bytesRemaining > 0 {
		buf := make([]byte, bytesRemaining)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			log.Errorf("failed to read from reader,err:%v", err)
		}
	}
}

// WriteMessageN writes a bitcoin Message to w including the necessary header
// information and returns the number of bytes written.    This function is the
// same as WriteMessage except it also returns the number of bytes written.
func WriteMessageN(w io.Writer, msg Message, pver uint32, btcnet BitcoinNet) (int, error) {
	return WriteMessageWithEncodingN(w, msg, pver, btcnet, BaseEncoding)
}

// WriteMessage writes a bitcoin Message to w including the necessary header
// information.  This function is the same as WriteMessageN except it doesn't
// doesn't return the number of bytes written.  This function is mainly provided
// for backwards compatibility with the original API, but it's also useful for
// callers that don't care about byte counts.
func WriteMessage(w io.Writer, msg Message, pver uint32, btcnet BitcoinNet) error {
	_, err := WriteMessageN(w, msg, pver, btcnet)
	return err
}

// WriteMessageWithEncodingN writes a bitcoin Message to w including the
// necessary header information and returns the number of bytes written.
// This function is the same as WriteMessageN except it also allows the caller
// to specify the message encoding format to be used when serializing wire
// messages.
func WriteMessageWithEncodingN(w io.Writer, msg Message, pver uint32,
	btcnet BitcoinNet, encoding MessageEncoding) (int, error) {

	totalBytes := 0

	// Enforce max command size.
	var command [CommandSize]byte
	cmd := msg.Command()
	if len(cmd) > CommandSize {
		str := fmt.Sprintf("command [%s] is too long [max %v]",
			cmd, CommandSize)
		return totalBytes, messageError("WriteMessage", str)
	}
	copy(command[:], []byte(cmd))

	// Encode the message payload.
	var bw bytes.Buffer
	err := msg.BtcEncode(&bw, pver, encoding)
	if err != nil {
		return totalBytes, err
	}
	payload := bw.Bytes()
	lenp := len(payload)

	// Enforce maximum overall message payload.
	if lenp > MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload is %d bytes",
			lenp, MaxMessagePayload)
		return totalBytes, messageError("WriteMessage", str)
	}

	// Enforce maximum message payload based on the message type.
	mpl := msg.MaxPayloadLength(pver)
	if uint32(lenp) > mpl {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload size for "+
			"messages of type [%s] is %d.", lenp, cmd, mpl)
		return totalBytes, messageError("WriteMessage", str)
	}

	// Create header for the message.
	hdr := messageHeader{}
	hdr.magic = btcnet
	hdr.command = cmd
	hdr.length = uint32(lenp)
	copy(hdr.checksum[:], chainhash.DoubleHashB(payload)[0:4])

	// Encode the header for the message.  This is done to a buffer
	// rather than directly to the writer since writeElements doesn't
	// return the number of bytes written.
	hw := bytes.NewBuffer(make([]byte, 0, MessageHeaderSize))
	err = writeElements(hw, hdr.magic, command, hdr.length, hdr.checksum)
	if err != nil {
		log.Errorf("faild to write elements,err:%v", err)
	}
	// Write header.
	n, err := w.Write(hw.Bytes())
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}

	// Write payload.
	n, err = w.Write(payload)
	totalBytes += n
	return totalBytes, err
}

// ReadMessageWithEncodingN reads, validates, and parses the next bitcoin Message
// from r for the provided protocol version and bitcoin network.  It returns the
// number of bytes read in addition to the parsed Message and raw bytes which
// comprise the message.  This function is the same as ReadMessageN except it
// allows the caller to specify which message encoding is to to consult when
// decoding wire messages.
func ReadMessageWithEncodingN(r io.Reader, pver uint32, btcnet BitcoinNet,
	enc MessageEncoding) (int, Message, []byte, error) {

	totalBytes := 0
	n, hdr, err := readMessageHeader(r)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

	// Enforce maximum message payload.
	if hdr.length > MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - header "+
			"indicates %d bytes, but max message payload is %d "+
			"bytes.", hdr.length, MaxMessagePayload)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Check for messages from the wrong bitcoin network.
	if hdr.magic != btcnet {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("message from other network [%v]", hdr.magic)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Check for malformed commands.
	command := hdr.command
	if !utf8.ValidString(command) {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("invalid command %v", []byte(command))
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Create struct of appropriate message type based on the command.
	msg, err := makeEmptyMessage(command)
	if err != nil {
		discardInput(r, hdr.length)
		return totalBytes, nil, nil, messageError("ReadMessage",
			err.Error())
	}

	// Check for maximum length based on the message type as a malicious client
	// could otherwise create a well-formed header and set the length to max
	// numbers in order to exhaust the machine's memory.
	mpl := msg.MaxPayloadLength(pver)
	if hdr.length > mpl {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("payload exceeds max length - header "+
			"indicates %v bytes, but max payload size for "+
			"messages of type [%v] is %v.", hdr.length, command, mpl)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Read payload.
	payload := make([]byte, hdr.length)
	n, err = io.ReadFull(r, payload)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

	// Test checksum.
	checksum := chainhash.DoubleHashB(payload)[0:4]
	if !bytes.Equal(checksum[:], hdr.checksum[:]) {
		str := fmt.Sprintf("payload checksum failed - header "+
			"indicates %v, but actual checksum is %v.",
			hdr.checksum, checksum)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Unmarshal message.  NOTE: This must be a *bytes.Buffer since the
	// MsgVersion BtcDecode function requires it.
	pr := bytes.NewBuffer(payload)
	err = msg.BtcDecode(pr, pver, enc)
	if err != nil {
		return totalBytes, nil, nil, err
	}

	return totalBytes, msg, payload, nil
}

// ReadMessageN reads, validates, and parses the next bitcoin Message from r for
// the provided protocol version and bitcoin network.  It returns the number of
// bytes read in addition to the parsed Message and raw bytes which comprise the
// message.  This function is the same as ReadMessage except it also returns the
// number of bytes read.
func ReadMessageN(r io.Reader, pver uint32, btcnet BitcoinNet) (int, Message, []byte, error) {
	return ReadMessageWithEncodingN(r, pver, btcnet, BaseEncoding)
}

// ReadMessage reads, validates, and parses the next bitcoin Message from r for
// the provided protocol version and bitcoin network.  It returns the parsed
// Message and raw bytes which comprise the message.  This function only differs
// from ReadMessageN in that it doesn't return the number of bytes read.  This
// function is mainly provided for backwards compatibility with the original
// API, but it's also useful for callers that don't care about byte counts.
func ReadMessage(r io.Reader, pver uint32, btcnet BitcoinNet) (Message, []byte, error) {
	_, msg, buf, err := ReadMessageN(r, pver, btcnet)
	return msg, buf, err
}

// MakeMessageHash calculate Message hash.
func MakeMessageHash(msg Message, pver uint32, btcnet BitcoinNet) chainhash.Hash {
	var buf bytes.Buffer
	_, err := WriteMessageWithEncodingN(&buf, msg, pver,
		btcnet, LatestEncoding)
	if err != nil {
		log.Errorf("fail to write message with encoding,err:%v", err)
	}

	return chainhash.HashH(buf.Bytes())
}

// MsgShouldRelay checks if the message needs to relay.
func MsgShouldRelay(msg Message) bool {
	msgs := []string{
		//TODO(nanlin): Modify msg need relay.
		CmdBinaryBA,
		CmdBinaryBAFin,
		CmdMsgBlockConfirmation,
		CmdLeaderProposal,
		CmdLeaderVote,
		CmdGC,
		CmdSeed,
		//CmdNewRoundStart,
		//CmdSyncInv,
		CmdBlockHeader,
		//CmdTx,
		//CmdTxBatch,
		//CmdSyncReq,
		//CmdSyncInv,
		//CmdSyncHeaders,
		//CmdVote,
		//CmdStopShard,
		//CmdStartShard,
		// HeartBeat and whiteList
		CmdSlimBlock,
		CmdHeartBeat,
		CmdRelayHash,
		//CmdReturnWhiteList,
		//CmdFetchWhiteList,
	}

	for _, cmd := range msgs {
		if msg.Command() == cmd {
			return true
		}
	}
	return false
}
