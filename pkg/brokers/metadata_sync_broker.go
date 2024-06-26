package brokers

import (
	"arbokcore/core/database"
	"arbokcore/core/notifiers"
	"arbokcore/core/supervisors"
	"arbokcore/pkg/config"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/utils"
	"arbokcore/pkg/workerpool"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type FileUpdateSyncBroker struct {
	name       string
	demandChan chan supervisors.Demand
	producer   *supervisors.MetadataUpdateConsumer
	consumer   *SSEConsumer
}

func NewFileUpdateSyncBroker(
	name string,
	producer *supervisors.MetadataUpdateConsumer,
	consumer *SSEConsumer,
) *FileUpdateSyncBroker {

	return &FileUpdateSyncBroker{
		name:       name,
		demandChan: make(chan supervisors.Demand, 1),
		producer:   producer,
		consumer:   consumer,
	}
}

type SSEConsumer struct {
	Dst *SSEBroker
}

func (r *SSEConsumer) Execute(ctx context.Context, payloads []*queuer.Payload) error {
	// fmt.Print("update event receiver", len(payloads))
	// utils.Dump(payloads)

	for _, payload := range payloads {
		fmt.Println(payload.Message)
		b := bytes.NewBuffer(payload.Message)

		cachedData := notifiers.MetadataUpdateStatusEvent{}

		decoder := gob.NewDecoder(b)
		err := decoder.Decode(&cachedData)
		if err != nil {
			log.Error().Err(err).Msg("failed to decode")
			return err
		}

		fmt.Println("cache data, receiver")
		utils.Dump(cachedData)

		str := "fileID:" + cachedData.FileID
		fmt.Println(str)

		// TODO: get all devices for user

		r.Dst.SendMessage(ctx, Message{
			UserID:   cachedData.UserID,
			DeviceID: cachedData.DeviceID,
			Content:  []byte(str),
		})

		// Now I don't have multiple devices yet.
		// So what I will do is, set another message.

		anotherDevice := "2"
		if cachedData.DeviceID == "2" {
			anotherDevice = "1"
		}

		r.Dst.SendMessage(ctx, Message{
			UserID:   cachedData.UserID,
			DeviceID: anotherDevice,
			Content:  []byte(str),
		})
	}

	return nil
}

func (slf *FileUpdateSyncBroker) HandleDemand(ctx context.Context, demand supervisors.Demand) {
	slf.demandChan <- demand
}

func (slf *FileUpdateSyncBroker) Start(ctx context.Context) {
	// Here what happens is
	// The producer is running, in for loop, waiting for demand to arrive
	// In the slf.producer.Demand() call later down the function
	// send the demand, the producer.Start sends the data on receiveChan
	// The Dispatcher gets this data on receive chan, and then passes it on to a worker
	// which executes the consumer.Exevute function

	// This consumer.Execute is responsible for, fetching the devices for a user
	// And then send them to the SSEBroker channel.
	pool := workerpool.NewWorkerPool(1, slf.consumer.Execute, false)
	recvChan := slf.producer.Produce(ctx)

	go workerpool.Dispatch(ctx, pool, recvChan)

	pool.Start(ctx)

	go func() {

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("stopping pool")
				pool.Stop(ctx)
				return
			case demand, ok := <-slf.demandChan:
				if ok {
					// fmt.Println("demanding device updates")
					slf.producer.Demand(demand)
				}
			}
		}
	}()
}

func SetupFileEventProducer(cfg config.AppConfig) (*supervisors.MetadataUpdateConsumer, error) {
	redisconn, err := database.NewRedisConnection(cfg.RedisURL)

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to redis")
		return nil, err
	}

	nsq := queuer.NewRedisQ(
		redisconn,
		database.MetadataUpdateClientsNotifierQueue,
		1*time.Second,
	)

	//connects to redis q
	// and reads messages on demand
	// The demand is sent via a demand chan
	return supervisors.NewMetadataNotifier(nsq), nil
}

/*

syncer := NewSyncBroker(MetadataNotifier)

syncer.WatchStreamFor(userID1, receiver)
syncer.WatchStreamFor(userID2, receiver)

	redisconn, err := database.NewRedisConnection(cfg.RedisURL)

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to redis")
		return err
	}

	ctx = context.Background()

	nsq := queuer.NewRedisQ(
		redisconn,
		database.MetadataUpdateClientsNotifierQueue,
		1*time.Second,
	)

	producer := supervisors.NewMetadataNotifier(nsq)

ListenFor(userID) {
	resultChan  := make()
	receiver := producer.Produce(ctx)

	go func() {
		for {
			case <-ticker.C:
				producer.Demand(Demand{Count:1, UserID: })
			case data, ok := <-receiver:
				resultChan <- data

		}

	}
}
*/
