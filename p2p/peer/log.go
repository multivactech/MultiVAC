// Copyright (c) 2018-present, MultiVAC Foundation.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"fmt"
	"strings"

	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/logger/btclog"
	"github.com/multivactech/MultiVAC/model/wire"
)

const (
	// maxRejectReasonLen is the maximum length of a sanitized reject reason
	// that will be logged.
	maxRejectReasonLen = 250
)

var log btclog.Logger

func init() {
	var exists bool
	log, exists = logger.GetLogger(logger.PeerLoggerTag)
	if !exists {
		panic("Fail to get logger for peer package")
	}
	log.SetLevel(logger.PeerLogLevel)
}

// SuppressLogForTest is disable log, this is only for testing.
func SuppressLogForTest() {
	log = btclog.Disabled
}

// LogClosure is a closure that can be printed with %v to be used to
// generate expensive-to-create data for a detailed log level and avoid doing
// the work if the data isn't printed.
type logClosure func() string

func (c logClosure) String() string {
	return c()
}

func newLogClosure(c func() string) logClosure {
	return logClosure(c)
}

// directionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

const (
	// LockTimeThreshold is the number below which a lock time is
	// interpreted to be a block number.  Since an average of one block
	// is generated per 10 minutes, this allows blocks for about 9,512
	// years.
	LockTimeThreshold = 5e8 // Tue Nov 5 00:53:20 1985 UTC
)

// sanitizeString strips any characters which are even remotely dangerous, such
// as html control characters, from the passed string.  It also limits it to
// the passed maximum size, which can be 0 for unlimited.  When the string is
// limited, it will also add "..." to the string to indicate it was truncated.
func sanitizeString(str string, maxLength uint) string {
	const safeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXY" +
		"Z01234567890 .,;_/:?@"

	// Strip any characters not in the safeChars string removed.
	str = strings.Map(func(r rune) rune {
		if strings.ContainsRune(safeChars, r) {
			return r
		}
		return -1
	}, str)

	// Limit the string to the max allowed length.
	if maxLength > 0 && uint(len(str)) > maxLength {
		str = str[:maxLength]
		str = str + "..."
	}
	return str
}

// messageSummary returns a human-readable string which summarizes a message.
// Not all messages have or need a summary.  This is used for debug logging.
func messageSummary(msg wire.Message) string {
	switch msg := msg.(type) {
	case *wire.MsgVersion:
		return fmt.Sprintf("agent %s, pver %d, block %d",
			msg.UserAgent, msg.ProtocolVersion, msg.LastBlock)

	case *wire.MsgVerAck:
		// No summary.

	case *wire.MsgGetAddr:
		// No summary.

	case *wire.MsgAddr:
		return fmt.Sprintf("%d addr", len(msg.AddrList))

	case *wire.MsgPing:
		// No summary - perhaps add nonce.

	case *wire.MsgPong:
		// No summary - perhaps add nonce.

	case *wire.MsgAlert:
		// No summary.

	case *wire.MsgTx:
		return fmt.Sprintf("hash %s, %d inputs",
			msg.TxHash(), len(msg.TxIn))

	case *wire.MsgBlock:
		header := &msg.Header
		return fmt.Sprintf("hash %s, ver %d, %d tx", header.BlockBodyHash,
			header.Version, len(msg.Body.Transactions))

	case *wire.MsgReject:
		// Ensure the variable length strings don't contain any
		// characters which are even remotely dangerous such as HTML
		// control characters, etc.  Also limit them to sane length for
		// logging.
		rejCommand := sanitizeString(msg.Cmd, wire.CommandSize)
		rejReason := sanitizeString(msg.Reason, maxRejectReasonLen)
		summary := fmt.Sprintf("cmd %v, code %v, reason %v", rejCommand,
			msg.Code, rejReason)
		if rejCommand == wire.CmdBlock || rejCommand == wire.CmdTx {
			summary += fmt.Sprintf(", hash %v", msg.Hash)
		}
		return summary
	}

	// No summary for other messages.
	return ""
}
