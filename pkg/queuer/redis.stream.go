package queuer

import (
	"arbokcore/pkg/templating"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisStream struct {
	conn      *redis.Client
	topicName string
	timeout   time.Duration
}

func NewRedisStream(
	conn *redis.Client,
	topicName string,
	timeout time.Duration,
) *RedisStream {

	return &RedisStream{
		conn:      conn,
		topicName: topicName,
		timeout:   timeout,
	}
}

func IsBusyGroup(err error) bool {
	return strings.HasPrefix(err.Error(), "BUSYGROUP")
}

func (rs *RedisStream) EnqueueMsg(ctx context.Context, partition string, data *Payload) error {
	stream := templating.NewTemplateString(rs.topicName)(partition)
	group := fmt.Sprintf("%s-group", stream)

	log.Info().Msg("creating consumer group " + group)

	_, err := rs.conn.XGroupCreateMkStream(ctx, stream, group, "$").Result()
	if err != nil && !errors.Is(err, redis.Nil) && !IsBusyGroup(err) {
		return fmt.Errorf("failed to create consumer group %s: %w", group, err)
	}

	log.Info().Str("q", stream).Msg("pushing to stream")

	_, err = rs.conn.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"message": string(data.Message),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to push message to stream %s: %w", stream, err)
	} else {
		log.Info().Msg("data pushed to stream")
	}

	return nil
}

func (rs *RedisStream) EnqueueMsgs(ctx context.Context, partition string, datas []*Payload) error {
	stream := templating.NewTemplateString(rs.topicName)(partition)
	log.Info().Str("q", stream).Msg("pushing to stream")

	failed := []*Payload{}

	for _, data := range datas {
		_, err := rs.conn.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{
				"message": string(data.Message),
			},
		}).Result()
		if err != nil {
			failed = append(failed, data)
		}
	}

	if len(failed) == 0 {
		return nil
	}

	return PartialError{
		Err:    errors.New("partial_failure"),
		Failed: failed,
	}
}

func (rs *RedisStream) ReadMsg(ctx context.Context, partition string, consumerGroup string) (*Payload, error) {
	stream := templating.NewTemplateString(rs.topicName)(partition)
	log.Info().Str("q", stream).Msg("pushing to stream")

	group := fmt.Sprintf("%s-group", stream)

	// Create the consumer group if it doesn't exist
	_, err := rs.conn.XGroupCreateMkStream(ctx, stream, group, "$").Result()
	if err != nil && !errors.Is(err, redis.Nil) && !IsBusyGroup(err) {
		return nil, fmt.Errorf("failed to create consumer group %s: %w", group, err)
	}

	// Read messages from the stream
	res, err := rs.conn.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumerGroup,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    rs.timeout,
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to read from stream %s: %w", stream, err)
	}

	if len(res) == 0 || len(res[0].Messages) == 0 {
		return nil, nil
	}

	msg := res[0].Messages[0]
	data := &Payload{
		Message: []byte(msg.Values["message"].(string)),
		Key:     msg.ID,
	}

	// Acknowledge the message
	_, err = rs.conn.XAck(ctx, stream, group, msg.ID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge message %s: %w", msg.ID, err)
	}

	return data, nil
}

func (rs *RedisStream) Flush(ctx context.Context) error {
	return rs.conn.FlushDB(ctx).Err()
}
