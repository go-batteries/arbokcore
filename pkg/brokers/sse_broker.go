package brokers

import (
	"arbokcore/pkg/utils"
	"context"
	"fmt"
	"time"
)

type DeviceConnections map[chan *Message]string

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

type SSEBroker struct {
	name          string
	topicNames    []string
	messagesCh    chan Message
	subsciberCh   chan *Topic
	unsubscribeCh chan *Topic
}

type Topic struct {
	Ch        chan *Message
	Partition string
	Name      string
	Running   bool
	Done      chan bool
}

func NewSSEBroker(name string) *SSEBroker {
	return &SSEBroker{
		name:          name,
		messagesCh:    make(chan Message, 1),
		topicNames:    []string{},
		subsciberCh:   make(chan *Topic, 1),
		unsubscribeCh: make(chan *Topic, 1),
	}
}

func (b *SSEBroker) Subscribe(ctx context.Context, topicName string) chan *Message {
	fmt.Println("signal broker to add subscriber")

	receiver := make(chan *Message, 1)
	b.subsciberCh <- &Topic{Name: topicName, Ch: receiver}

	fmt.Println("signal broker to add subscriber done", receiver)
	return receiver
}

func (b *SSEBroker) Unsubscribe(ctx context.Context, deviceChan chan *Message) {
	fmt.Println("broker to remove subscriber")

	doneCh := make(chan bool)
	b.unsubscribeCh <- &Topic{Ch: deviceChan, Done: doneCh}
	fmt.Println("signal broker to remove subscriber")

	<-doneCh
	fmt.Println("broker unsubscribed")
}

func (b *SSEBroker) SendMessage(ctx context.Context, data Message) {
	select {
	case b.messagesCh <- data:
		fmt.Println("sending data")
		utils.Dump(data)
	case <-time.After(9 * time.Second):
		fmt.Println("failed to send message ", data)
	}
}

// Maybe we will change this to a hash
func findTopicByName(topics DeviceConnections, topicName string) chan *Message {
	for ch, name := range topics {
		if name == topicName {
			return ch
		}
	}

	return nil
}

func (b *SSEBroker) Start(ctx context.Context) {
	topics := make(DeviceConnections)

	go func() {
		for {
			fmt.Println("====topics====")
			utils.Dump(topics)
			select {
			case <-ctx.Done():
				fmt.Println("====ctx done====")
				return
			case subs, ok := <-b.subsciberCh:
				if !ok {
					fmt.Println("failed to get subscriber channel")
					continue
				}

				rec, ok := topics[subs.Ch]
				if ok {
					fmt.Println("existing subscriber found", rec)
					continue
				}

				subs.Running = true
				topics[subs.Ch] = subs.Name

			case subs, ok := <-b.unsubscribeCh:
				if !ok {
					fmt.Println("unsubscribe channel malfunction")
					continue
				}

				_, ok = topics[subs.Ch]
				if !ok {
					fmt.Println("channel not found")
					continue
				}

				subs.Running = false
				subs.Done <- true
				close(subs.Done)

				delete(topics, subs.Ch)
				close(subs.Ch)

			case msg, ok := <-b.messagesCh:
				if !ok {
					fmt.Println("no messages received")
					continue
				}

				ch := findTopicByName(topics, msg.Key())
				if ch == nil {
					fmt.Println("no channels active for topic", msg.Key())
					continue
				}

				select {
				case ch <- &msg:
				case <-time.After(10 * time.Second):
					fmt.Println("failed to send msg to channel", msg)
				}

			}
		}
	}()
}
