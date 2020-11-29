package connection

import (
	"sync"
	"time"

	"github.com/multivactech/MultiVAC/logger"
	"github.com/multivactech/MultiVAC/model/shard"
	"github.com/multivactech/MultiVAC/model/wire"
	"github.com/multivactech/MultiVAC/p2p/peer"
)

// Multiplexer defines the data structure for handler.
type Multiplexer struct {
	msgHandlers map[Tag][]chan *MessageAndReply
}

// Tag is the key of multiplexer map.
type Tag struct {
	Msg   string
	Shard shard.Index
}

// MessagesAndReceiver defines the data structure for Register handler.
type MessagesAndReceiver struct {
	Tags     []Tag
	Channels chan *MessageAndReply
}

// MessageAndReply defines the data structure that the p2p network modules will
// send to other modules by channel.
type MessageAndReply struct {
	Msg   wire.Message
	Reply peer.Reply
}

// MakeMultiplexer creates an empty Multiplexer.
func MakeMultiplexer() *Multiplexer {
	var m Multiplexer
	m.msgHandlers = make(map[Tag][]chan *MessageAndReply)
	return &m
}

// Handle will find the channels which the message should be transmitted
// according to the command of message and the given shard index,
// and then put the message into the channels.
func (m *Multiplexer) Handle(message wire.Message, pr peer.Reply) bool {
	shard := getMsgShard(message)
	channels := m.getChannels(Tag{
		Msg:   message.Command(),
		Shard: shard,
	})
	if channels != nil {
		logger.ConnServerLogger().Debugf("success get channel")
		messageAndReply := &MessageAndReply{
			Msg:   message,
			Reply: pr,
		}
		for _, h := range channels {
			select {
			case h <- messageAndReply:
				logger.ConnServerLogger().Debugf("successfully send message to the given channel")
			case <-time.After(3 * time.Second):
				logger.ConnServerLogger().Errorf("channel seems full,time out!"+
					"len:%v,cap:%v,message:%v", len(h), cap(h), messageAndReply.Msg)
				return false
			}
		}
		return true
	}
	logger.ConnServerLogger().Debugf("failed to get channel,message:%v,shard:%v", message.Command(), shard)
	return false
}

var mtx sync.RWMutex

// RegisterChannels is used to register the channels for message reception to the Map
// so that the network provider can easily find the channel and then put the message
// into the channel and pass it to other modules.
func (m *Multiplexer) RegisterChannels(dispatch *MessagesAndReceiver) {
	mtx.Lock()
	defer mtx.Unlock()
	for _, tag := range dispatch.Tags {
		if channels, ok := m.msgHandlers[tag]; ok {
			if exist(dispatch.Channels, channels) {
				logger.ConnServerLogger().Debugf("channel for message %v is already exist", dispatch.Tags)
			} else {
				channels = append(channels, dispatch.Channels)
				m.msgHandlers[tag] = channels
			}
		} else {
			m.msgHandlers[tag] = []chan *MessageAndReply{dispatch.Channels}
		}
	}
}

// exist will check if the message channel has already registered
func exist(handler chan *MessageAndReply, handlers []chan *MessageAndReply) bool {
	if len(handlers) == 0 {
		return false
	}
	for _, h := range handlers {
		if h == handler {
			return true
		}
	}
	return false
}

func (m *Multiplexer) getChannels(tag Tag) []chan *MessageAndReply {
	mtx.RLock()
	defer mtx.RUnlock()
	channels, ok := m.msgHandlers[tag]
	if ok {
		return channels
	}
	return nil
}
