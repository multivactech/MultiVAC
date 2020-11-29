/**
 * Copyright (c) 2018-present, MultiVAC Foundation.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package message

// Actor represents a computation unit for actor pattern: https://www.brianstorti.com/the-actor-model/
type Actor interface {
	// Act is to act the callback function when received the e event with the instance of Actor.
	Act(e *Event, callback func(m interface{}))
}

// ActorContext is the context of the Actors.
type ActorContext struct {
	// mailboxes are instruction channels for sending message to the Actors.
	mailboxes map[Actor]chan *instruction
}

type instruction struct {
	e        *Event
	callback func(m interface{})
}

// EventTopic represents the subscription topic
type EventTopic int

// Event has a topic and a kind of message that needs to be processed by actors.
type Event struct {
	Topic EventTopic
	Extra interface{}
}

// NewEvent returns a new instance of Event.
func NewEvent(t EventTopic, e interface{}) *Event {
	return &Event{Topic: t, Extra: e}
}

// NewActorContext makes a new instance of ActorContext.
func NewActorContext() *ActorContext {
	actorContext := &ActorContext{}
	actorContext.mailboxes = make(map[Actor]chan *instruction)
	return actorContext
}

// StartActor starts the process of the given Actor.
func (ctx ActorContext) StartActor(a Actor) {
	// If the given Actor is not started, create instruction chan and put it into mailboxes
	if !ctx.ActorStarted(a) {
		m := make(chan *instruction)
		ctx.mailboxes[a] = m
	}
	// Listen to the relevant mailbox and delegates to the given Actor
	go ctx.startProc(a)
}

// Send sends instruction to the given Actor.
func (ctx ActorContext) Send(a Actor, e *Event, callback func(m interface{})) {
	ctx.mailboxes[a] <- &instruction{e: e, callback: callback}
}

// SendAndWait sends the given Event to the given Actor and wait until getting response.
func (ctx ActorContext) SendAndWait(a Actor, e *Event) interface{} {
	c := make(chan interface{}, 1)
	ctx.Send(a, e, func(r interface{}) { c <- r })
	return <-c
}

// ActorStarted checks whether the given Actor is started.
//
// TODO(wangruichao@mtv.ac): need to consider concurrent safety.
func (ctx ActorContext) ActorStarted(a Actor) bool {
	_, ok := ctx.mailboxes[a]
	return ok
}

// startProc starts the listening process for the given Actor.
//
// When received message from mailbox, the given Actor will Act. Otherwise block the process.
func (ctx ActorContext) startProc(a Actor) {
	go func() {
		mailbox := ctx.mailboxes[a]
		for mail := range mailbox {
			a.Act(mail.e, mail.callback)
		}
	}()
}
