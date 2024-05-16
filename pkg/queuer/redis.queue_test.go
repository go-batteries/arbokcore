package queuer

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RedisEnqueue(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()

	err := client.Ping(ctx).Err()
	require.NoError(t, err)

	rq := NewRedisQ(
		client,
		"testQ",
		10*time.Second,
	)

	err = rq.Flush(ctx)
	require.NoError(t, err)

	msgs := []string{"message1", "message2", "message3"}
	for _, msg := range msgs {
		err := rq.EnqueueMsg(ctx, &Payload{
			Message: []byte(msg),
		})

		require.NoError(t, err)
	}

	values := []string{}

	for i := 0; i < 3; i++ {
		data, err := rq.ReadMsg(ctx, "does not matter")
		require.NoError(t, err)

		result := string(data.Message)
		values = append(values, result)
	}

	require.Equal(t, 3, len(values))
	assert.Equal(t, msgs, values)
}
