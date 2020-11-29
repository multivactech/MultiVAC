/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package message

// PubSubManager represents a computation unit for pub-sub pattern: https://en.wikipedia.org/wiki/Publish–subscribe_pattern
// This is applied to internal in-shard broadcast communication.
type PubSubManager struct {
	// subscribers receives the Event the PubSubManager concerns with.
	subscribers []Subscriber
}

// NewPubSubManager gives a new PubSubManager.
func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		subscribers: make([]Subscriber, 0),
	}
}

// Pub publishes the given Event to the subscribers.
func (pubSubMgr *PubSubManager) Pub(evt Event) {
	for _, subscr := range pubSubMgr.subscribers {
		go subscr.Recv(evt)
	}
}

// Sub subscribes the given EventTopic with the given Subscriber.
//
// TODO (wangruichao@mtv.ac): need to consider concurrent safety.
func (pubSubMgr *PubSubManager) Sub(subscr Subscriber) {
	pubSubMgr.subscribers = append(pubSubMgr.subscribers, subscr)
}

// Subscriber is a interface that contains a method for receiving Event.
type Subscriber interface {
	Recv(evt Event)
}
