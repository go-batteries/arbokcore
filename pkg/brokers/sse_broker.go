package brokers

import (
	"context"
	"fmt"
	"time"
)

type DeviceConnections map[string]chan Message

type Message struct {
	UserID   string
	DeviceID string
	Content  []byte
}

type EventData struct {
	UserID   string
	DeviceID string
}

func (m Message) Key() string {
	return fmt.Sprintf("%s_%s", m.UserID, m.DeviceID)
}

type Broker struct {
	name          string
	topicNames    []string
	messagesCh    chan Message
	subsciberCh   chan *Topic
	unsubscribeCh chan *Topic
}

type Topic struct {
	Ch      chan Message
	Name    string
	Running bool
	Done    chan bool
}

func NewSSEBroker(name string) *Broker {
	return &Broker{
		name:          name,
		messagesCh:    make(chan Message, 1),
		topicNames:    []string{},
		subsciberCh:   make(chan *Topic, 1),
		unsubscribeCh: make(chan *Topic, 1),
	}
}

func (b *Broker) Subscribe(ctx context.Context, topicName string) chan Message {
	fmt.Println("signal broker to add subscriber")

	receiver := make(chan Message, 1)
	b.subsciberCh <- &Topic{Name: topicName, Ch: receiver}

	fmt.Println("signal broker to add subscriber done")
	return receiver
}

func (b *Broker) Unsubscribe(ctx context.Context, topicName string) {
	fmt.Println("broker to remove subscriber")

	doneCh := make(chan bool)
	b.unsubscribeCh <- &Topic{Name: topicName, Done: doneCh}
	fmt.Println("signal broker to remove subscriber")

	<-doneCh
	fmt.Println("broker unsubscribed")
}

func (b *Broker) SendMessage(ctx context.Context, data Message) {
	select {
	case b.messagesCh <- data:
	case <-time.After(9 * time.Second):
		fmt.Println("failed to send message ", data)
	}
}

func (b *Broker) Start(ctx context.Context) {
	topics := make(DeviceConnections)

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("ctx done")
				return
			case subs, ok := <-b.subsciberCh:
				if !ok {
					fmt.Println("failed to get subscriber channel")
					continue
				}

				rec, ok := topics[subs.Name]
				if ok {
					fmt.Println("existing subscriber found", rec)
					continue
				}

				subs.Running = true
				topics[subs.Name] = subs.Ch
			case subs, ok := <-b.unsubscribeCh:
				ch, ok := topics[subs.Name]
				if !ok {
					fmt.Println("channel not found")
					continue
				}

				subs.Running = false
				subs.Done <- true
				close(subs.Done)

				delete(topics, subs.Name)
				close(ch)
			case msg, ok := <-b.messagesCh:
				if !ok {
					fmt.Println("no messages received")
					continue
				}

				ch, ok := topics[msg.Key()]
				if !ok {
					fmt.Println("no channels active for topic")
					continue
				}

				select {
				case ch <- msg:
				case <-time.After(10 * time.Second):
					fmt.Println("failed to send msg to channel", msg)
				}

			}
		}
	}()
}
