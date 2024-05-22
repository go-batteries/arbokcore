package ssebroker

import (
	"arbokcore/pkg/rho"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_BrokerSubscription(t *testing.T) {
	ctx := context.Background()

	broker := NewBroker("test_sse")
	broker.Start(ctx)

	receiver1 := broker.Subscribe(ctx, "user1_device1")
	receiver2 := broker.Subscribe(ctx, "user1_device2")

	generator := func(n int) chan bool {
		done := make(chan bool)

		go func() {
			defer close(done)

			for i := 0; i < n; i++ {
				if i%2 == 0 {
					broker.SendMessage(ctx, Message{
						UserID:   "user1",
						DeviceID: "device1",
					})
				} else {
					broker.SendMessage(ctx, Message{
						UserID:   "user1",
						DeviceID: "device2",
					})
				}
			}
		}()

		return done
	}

	collect := func(done chan bool) chan Message {
		results := make(chan Message)

		go func() {
			defer close(results)

			for {
				select {
				case <-done:
					return
				case data := <-receiver1:
					results <- data
				case data := <-receiver2:
					results <- data
				}
			}
		}()

		return results
	}

	messages := []Message{}
	done := make(chan bool)

	generator(10)
	result := collect(done)

	for i := 0; i < 10; i++ {
		messages = append(messages, <-result)
	}
	close(done)

	require.Equal(t, 10, len(messages))

	topics := rho.Map(messages, func(message Message, _ int) string {
		return message.Key()
	})

	topicMap := map[string]bool{}
	for _, topic := range topics {
		if _, ok := topicMap[topic]; !ok {
			topicMap[topic] = true
		}
	}

	require.Equal(t, map[string]bool{"user1_device1": true, "user1_device2": true}, topicMap)
}
