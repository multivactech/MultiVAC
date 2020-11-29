/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package message

const (
	// EvtSyncStart is the flag of SyncStart Event.
	EvtSyncStart EventTopic = iota

	// EvtSyncComplete is the flag of SyncComplete Event.
	EvtSyncComplete

	// EvtSyncAbort is the flag of SyncAbort Event.
	EvtSyncAbort

	// EvtLedgerFallBehind is the flag of UpdateLedger Event.
	// It is triggered when the consensus module detects that the round is behind.
	EvtLedgerFallBehind

	// EvtTxPoolInitFinished is the flag of TxPoolInitFinished Event.
	EvtTxPoolInitFinished
)
