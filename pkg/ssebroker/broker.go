package ssebroker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type DeviceConnections map[string]chan string

type Message struct {
	UserID   string
	DeviceID string
	Content  []byte
}

type Broker struct {
	messages    chan Message
	sseEventMap map[string]DeviceConnections
	mu          *sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		messages:    make(chan Message, 1),
		mu:          &sync.RWMutex{},
		sseEventMap: make(map[string]DeviceConnections),
	}
}

func (b *Broker) SendMessage(ctx context.Context, data Message) {
	select {
	case b.messages <- data:
		log.Info().Msg("data sent")
	case <-time.After(10 * time.Second):
		fmt.Println(data)
		log.Info().Msg("failed to send message")
	}
}

func (b *Broker) GetMessage(userID, deviceID string) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	deviceMap, ok := b.sseEventMap[userID]
	if !ok {
		log.Error().Msg("failed to get device map")
		return "", false
	}

	device, ok := deviceMap[deviceID]
	if !ok {
		log.Error().Msg("failed to get device")
		return "", false
	}

	msg, ok := <-device
	return msg, ok
}

func (b *Broker) Start(ctx context.Context) {
	go func() {
		for {
			// fmt.Println("running")
			select {
			case msg, ok := <-b.messages:
				// fmt.Println("reading message ", msg, ok)
				b.mu.RLock()

				devices, ok := b.sseEventMap[msg.UserID]
				if !ok {
					log.Error().Str("userID", msg.UserID).Msg("failed to get devices")
					b.mu.RUnlock()
					continue
				}

				var wg sync.WaitGroup
				// We need to check for active devices.

				for _, device := range devices {
					wg.Add(1)

					go func() {
						log.Info().Msg("sending data to device")

						defer wg.Done()

						select {
						case device <- string(msg.Content):
							// log.Info().Msg("data sent to sse")
						case <-time.After(10 * time.Second):
							log.Warn().Msg("device channel might have expired")
						case <-ctx.Done():
							// log.Info().Msg("stopped by ctx")
						}
					}()

					wg.Wait()

				}
				b.mu.RUnlock()

			case <-ctx.Done():
				log.Info().Msg("sse broker exiting on ctx")
				return
			}
		}
	}()
}

func (b *Broker) GetConnection(userID string) (DeviceConnections, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	deviceMap, ok := b.sseEventMap[userID]
	if !ok {
		return nil, false
	}

	return deviceMap, true
}

func (b *Broker) Print() {
	fmt.Printf("sse veent map %v:\n", b.sseEventMap)
}

func (b *Broker) AddConnection(userID string, deviceID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	deviceMap, ok := b.sseEventMap[userID]
	if !ok {
		b.sseEventMap[userID] = DeviceConnections{deviceID: make(chan string, 1)}
		return true
	}

	if _, ok := deviceMap[deviceID]; ok {
		return false
	}

	deviceMap[deviceID] = make(chan string, 1)
	b.sseEventMap[userID] = deviceMap

	return true
}

func (b *Broker) RemoveConnection(userID string, deviceID string) {
	log.Info().Msg("removing connection")

	b.mu.Lock()
	defer b.mu.Unlock()

	deviceMap, ok := b.sseEventMap[userID]
	if !ok {
		return
	}

	device, ok := deviceMap[deviceID]
	if !ok {
		return
	}

	close(device)
	delete(b.sseEventMap[userID], deviceID)
}

func (b *Broker) FlushUserConns(userID string) {
	_, ok := b.sseEventMap[userID]
	if !ok {
		return
	}

	delete(b.sseEventMap, userID)
}
