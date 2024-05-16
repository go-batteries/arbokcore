package queuer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Payload struct {
	Message []byte
	Key     string
	TTL     time.Duration
}

type PartialError struct {
	Err    error
	Failed []*Payload
}

func (perr PartialError) Error() string {
	return perr.Err.Error()
}

type Queuer interface {
	EnqueueMsg(ctx context.Context, data *Payload) error
	EnqueueMsgs(ctx context.Context, data []*Payload) error
	ReadMsg(ctx context.Context, key string) (*Payload, error)
}

type RedisQ struct {
	conn      *redis.Client
	queueName string
	timeout   time.Duration
}

func NewRedisQ(
	conn *redis.Client,
	queueName string,
	timeout time.Duration,
) *RedisQ {
	return &RedisQ{
		conn:      conn,
		queueName: queueName,
		timeout:   timeout,
	}
}

func (slf *RedisQ) EnqueueMsgs(ctx context.Context, datas []*Payload) (err error) {
	log.Info().Str("q", slf.queueName).Msg("pushing message to queue")

	failed := []*Payload{}

	for _, data := range datas {
		err = slf.conn.RPush(ctx, slf.queueName, string(data.Message)).Err()
		if err != nil {
			log.Error().Err(err).Msg("failed to push message to queue")
			failed = append(failed, data)
		}
	}

	if len(failed) == 0 {
		log.Info().Msg("enqueued all value")
		return nil
	}

	return PartialError{
		Err:    errors.New("partial_failure"),
		Failed: failed,
	}
}

func (slf *RedisQ) EnqueueMsg(ctx context.Context, data *Payload) (err error) {
	log.Info().Str("q", slf.queueName).Msg("pushing message to queue")

	err = slf.conn.RPush(ctx, slf.queueName, string(data.Message)).Err()
	if err != nil {
		log.Error().Err(err).Msg("failed to push message to queue")
	}

	return err
}

var (
	ErrEmptyRecords = errors.New("empty_records")
)

func (slf *RedisQ) ReadMsg(ctx context.Context, key string) (data *Payload, err error) {
	log.Info().Str("q", slf.queueName).Msg("reading messages from queue")

	results, err := slf.conn.BLPop(ctx, slf.timeout, slf.queueName).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// log.Info().Msg("no results in queue")
			return nil, nil
		}

		log.Error().Err(err).Msg("failed to fetch results from q")
		return nil, err
	}

	fmt.Println(results)

	if len(results) < 2 {
		return nil, ErrEmptyRecords
	}

	qname, result := results[0], results[1]
	data = &Payload{Message: []byte(result), Key: qname}

	log.Info().Int("len(data)", len(results)).
		Msg("fetched n record from q")

	return
}

func (slf *RedisQ) Flush(ctx context.Context) error {
	return slf.conn.FlushDB(ctx).Err()
}
