/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Copyright (c) 2014-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	"encoding/json"
	"errors"
	"github.com/multivactech/MultiVAC/rpc/btcjson"
)

var (
	// ErrWebsocketsRequired is an error to describe the condition where the
	// caller is trying to use a websocket-only feature, such as requesting
	// notifications or other websocket requests when the client is
	// configured to run in HTTP POST mode.
	ErrWebsocketsRequired = errors.New("a websocket connection is required " +
		"to use this feature")
)

// notificationState is used to track the current state of successfully
// registered notification so the state can be automatically re-established on
// reconnect.
type notificationState struct {
	notifyBlocks       bool
	notifyNewTx        bool
	notifyNewTxVerbose bool
	notifyReceived     map[string]struct{}
	notifySpent        map[btcjson.OutPoint]struct{}
}

// Copy returns a deep copy of the receiver.
func (s *notificationState) Copy() *notificationState {
	var stateCopy notificationState
	stateCopy.notifyBlocks = s.notifyBlocks
	stateCopy.notifyNewTx = s.notifyNewTx
	stateCopy.notifyNewTxVerbose = s.notifyNewTxVerbose
	stateCopy.notifyReceived = make(map[string]struct{})
	for addr := range s.notifyReceived {
		stateCopy.notifyReceived[addr] = struct{}{}
	}
	stateCopy.notifySpent = make(map[btcjson.OutPoint]struct{})
	for op := range s.notifySpent {
		stateCopy.notifySpent[op] = struct{}{}
	}

	return &stateCopy
}

// newNotificationState returns a new notification state ready to be populated.
func newNotificationState() *notificationState {
	return &notificationState{
		notifyReceived: make(map[string]struct{}),
		notifySpent:    make(map[btcjson.OutPoint]struct{}),
	}
}

// newNilFutureResult returns a new future result channel that already has the
// result waiting on the channel with the reply set to nil.  This is useful
// to ignore things such as notifications when the caller didn't specify any
// notification handlers.
/*func newNilFutureResult() chan *response {
	responseChan := make(chan *response, 1)
	responseChan <- &response{result: nil, err: nil}
	return responseChan
}*/

// NotificationHandlers defines callback function pointers to invoke with
// notifications.  Since all of the functions are nil by default, all
// notifications are effectively ignored until their handlers are set to a
// concrete callback.
//
// NOTE: Unless otherwise documented, these handlers must NOT directly call any
// blocking calls on the client instance since the input reader goroutine blocks
// until the callback has completed.  Doing so will result in a deadlock
// situation.
type NotificationHandlers struct {
	// OnClientConnected is invoked when the client connects or reconnects
	// to the RPC server.  This callback is run async with the rest of the
	// notification handlers, and is safe for blocking client requests.
	OnClientConnected func()

	// OnBlockConnected is invoked when a block is connected to the longest
	// (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	//
	// NOTE: Deprecated. Use OnFilteredBlockConnected instead.
	//OnBlockConnected func(hash *chainhash.Hash, height int32, t time.Time)

	// OnFilteredBlockConnected is invoked when a block is connected to the
	// longest (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.  Its parameters differ from OnBlockConnected: it
	// receives the block's height, header, and relevant transactions.
	//OnFilteredBlockConnected func(height int32, header *wire.BlockHeader,
	//	txs []*btcutil.Tx)

	// OnBlockDisconnected is invoked when a block is disconnected from the
	// longest (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	//
	// NOTE: Deprecated. Use OnFilteredBlockDisconnected instead.
	//OnBlockDisconnected func(hash *chainhash.Hash, height int32, t time.Time)

	// OnFilteredBlockDisconnected is invoked when a block is disconnected
	// from the longest (best) chain.  It will only be invoked if a
	// preceding NotifyBlocks has been made to register for the notification
	// and the call to function is non-nil.  Its parameters differ from
	// OnBlockDisconnected: it receives the block's height and header.
	//OnFilteredBlockDisconnected func(height int32, header *wire.BlockHeader)

	// OnRecvTx is invoked when a transaction that receives funds to a
	// registered address is received into the memory pool and also
	// connected to the longest (best) chain.  It will only be invoked if a
	// preceding call to NotifyReceived, Rescan, or RescanEndHeight has been
	// made to register for the notification and the function is non-nil.
	//
	// NOTE: Deprecated. Use OnRelevantTxAccepted instead.
	//OnRecvTx func(transaction *btcutil.Tx, details *btcjson.BlockDetails)

	// OnRedeemingTx is invoked when a transaction that spends a registered
	// outpoint is received into the memory pool and also connected to the
	// longest (best) chain.  It will only be invoked if a preceding call to
	// NotifySpent, Rescan, or RescanEndHeight has been made to register for
	// the notification and the function is non-nil.
	//
	// NOTE: The NotifyReceived will automatically register notifications
	// for the outpoints that are now "owned" as a result of receiving
	// funds to the registered addresses.  This means it is possible for
	// this to invoked indirectly as the result of a NotifyReceived call.
	//
	// NOTE: Deprecated. Use OnRelevantTxAccepted instead.
	//OnRedeemingTx func(transaction *btcutil.Tx, details *btcjson.BlockDetails)

	// OnRelevantTxAccepted is invoked when an unmined transaction passes
	// the client's transaction filter.
	//
	// NOTE: This is a btcsuite extension ported from
	// github.com/decred/dcrrpcclient.
	//OnRelevantTxAccepted func(transaction []byte)

	// OnRescanFinished is invoked after a rescan finishes due to a previous
	// call to Rescan or RescanEndHeight.  Finished rescans should be
	// signaled on this notification, rather than relying on the return
	// result of a rescan request, due to how btcd may send various rescan
	// notifications after the rescan request has already returned.
	//
	// NOTE: Deprecated. Not used with RescanBlocks.
	//OnRescanFinished func(hash *chainhash.Hash, height int32, blkTime time.Time)

	// OnRescanProgress is invoked periodically when a rescan is underway.
	// It will only be invoked if a preceding call to Rescan or
	// RescanEndHeight has been made and the function is non-nil.
	//
	// NOTE: Deprecated. Not used with RescanBlocks.
	//OnRescanProgress func(hash *chainhash.Hash, height int32, blkTime time.Time)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to false has been
	// made to register for the notification and the function is non-nil.
	//OnTxAccepted func(hash *chainhash.Hash, amount btcutil.Amount)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to true has been
	// made to register for the notification and the function is non-nil.
	//OnTxAcceptedVerbose func(txDetails *btcjson.TxRawResult)

	// OnBtcdConnected is invoked when a wallet connects or disconnects from
	// btcd.
	//
	// This will only be available when client is connected to a wallet
	// server such as btcwallet.
	//OnBtcdConnected func(connected bool)

	// OnAccountBalance is invoked with account balance updates.
	//
	// This will only be available when speaking to a wallet server
	// such as btcwallet.
	//OnAccountBalance func(account string, balance btcutil.Amount, confirmed bool)

	// OnWalletLockState is invoked when a wallet is locked or unlocked.
	//
	// This will only be available when client is connected to a wallet
	// server such as btcwallet.
	//OnWalletLockState func(locked bool)

	// OnUnknownNotification is invoked when an unrecognized notification
	// is received.  This typically means the notification handling code
	// for this package needs to be updated for a new notification type or
	// the caller is using a custom notification this package does not know
	// about.
	OnUnknownNotification func(method string, params []json.RawMessage)
}

// handleNotification examines the passed notification type, performs
// conversions to get the raw notification types into higher level types and
// delivers the notification to the appropriate On<X> handler registered with
// the client.
func (c *Client) handleNotification(ntfn *rawNotification) {
	// Ignore the notification if the client is not interested in any
	// notifications.
	if c.ntfnHandlers == nil {
		return
	}

	switch ntfn.Method {

	// OnUnknownNotification
	default:
		if c.ntfnHandlers.OnUnknownNotification == nil {
			return
		}

		c.ntfnHandlers.OnUnknownNotification(ntfn.Method, ntfn.Params)
	}
}
