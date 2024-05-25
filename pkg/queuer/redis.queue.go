package queuer

import (
	"arbokcore/pkg/templating"
	"context"
	"errors"
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
	EnqueueMsg(ctx context.Context, partition string, data *Payload) error
	EnqueueMsgs(ctx context.Context, partition string, data []*Payload) error
	ReadMsg(ctx context.Context, partition string, key string) (*Payload, error)
}

type RedisQ struct {
	conn      *redis.Client
	topicName string
	timeout   time.Duration
}

func NewRedisQ(
	conn *redis.Client,
	queueName string,
	timeout time.Duration,
) *RedisQ {
	return &RedisQ{
		conn:      conn,
		topicName: queueName,
		timeout:   timeout,
	}
}

// The redis queue is implemented using RPUSH and BLPOP
// BLPOP returns one message at a time.
// Here we are using a sequential fetch, even if we did a parallel fetch
// For 10 records it wouldn't matter and would cause contention or extra headache

// The other thing is, for maintaining integrity, we need to have serializability
// Understanding database serializability is important here

func (slf *RedisQ) EnqueueMsgs(ctx context.Context, partition string, datas []*Payload) (err error) {
	log.Info().Str("q", slf.topicName).Msg("pushing message to queue")

	failed := []*Payload{}

	for _, data := range datas {
		err = slf.conn.RPush(ctx, slf.topicName, string(data.Message)).Err()
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

func (slf *RedisQ) EnqueueMsg(ctx context.Context, partition string, data *Payload) (err error) {
	queue := templating.NewTemplateString(slf.topicName)(partition)
	log.Info().Str("q", queue).Msg("pushing message to queue")

	err = slf.conn.RPush(ctx, queue, string(data.Message)).Err()
	if err != nil {
		log.Error().Err(err).Msg("failed to push message to queue")
	}

	return err
}

var (
	ErrEmptyRecords = errors.New("empty_records")
)

func (slf *RedisQ) ReadMsg(ctx context.Context, partition string, key string) (data *Payload, err error) {
	queue := templating.NewTemplateString(slf.topicName)(partition)
	log.Info().Str("q", queue).Msg("reading messages from queue")

	results, err := slf.conn.BLPop(ctx, slf.timeout, queue).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// log.Info().Msg("no results in queue")
			return nil, nil
		}

		log.Error().Err(err).Msg("failed to fetch results from q")
		return nil, err
	}

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
