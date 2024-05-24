package queuer

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a separate database for testing to avoid conflicts
	})
}

func TestRedisStream_EnqueueMsg(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic", 5*time.Second)

	err := rs.EnqueueMsg(context.Background(), "partition1", &Payload{Message: []byte("test message")})
	assert.NoError(t, err, "expected no error while enqueuing message")
}

func TestRedisStream_EnqueueMsgs(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic", 5*time.Second)

	messages := []*Payload{
		{Message: []byte("message 1")},
		{Message: []byte("message 2")},
	}

	err := rs.EnqueueMsgs(context.Background(), "partition1", messages)
	assert.NoError(t, err, "expected no error while enqueuing messages")
}

func TestRedisStream_ReadMsg(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic::%s", 5*time.Second)

	partition := "partition1"
	consumer := "consumer1"

	// Enqueue a message
	err := rs.EnqueueMsg(context.Background(), partition, &Payload{Message: []byte("test message")})
	require.NoError(t, err, "expected no error while enqueuing message")

	// Read the message
	data, err := rs.ReadMsg(context.Background(), partition, consumer)
	require.NoError(t, err, "expected no error while reading message")
	assert.NotNil(t, data, "expected a message to be read")
	assert.Equal(t, []byte("test message"), data.Message, "expected message content to match")
}

func TestRedisStream_ReadMsg_NoMessage(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic::%s", 5*time.Second)

	partition := "partition1"
	consumer := "consumer1"

	// Attempt to read a message when there are none
	data, err := rs.ReadMsg(context.Background(), partition, consumer)
	assert.NoError(t, err, "expected no error while reading message")
	assert.Nil(t, data, "expected no message to be read")
}

func TestRedisStream_MultiplePartitions(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic::%s", 5*time.Second)

	partition1 := "partition1"
	partition2 := "partition2"

	// Enqueue messages to different partitions
	err := rs.EnqueueMsg(context.Background(), partition1, &Payload{Message: []byte("message for partition1")})
	require.NoError(t, err, "expected no error while enqueuing message to partition1")

	err = rs.EnqueueMsg(context.Background(), partition2, &Payload{Message: []byte("message for partition2")})
	require.NoError(t, err, "expected no error while enqueuing message to partition2")

	// Read messages from different partitions
	data1, err := rs.ReadMsg(context.Background(), partition1, "consumer1")
	require.NoError(t, err, "expected no error while reading message from partition1")
	assert.NotNil(t, data1, "expected a message to be read from partition1")
	assert.Equal(t, []byte("message for partition1"), data1.Message, "expected message content to match for partition1")

	data2, err := rs.ReadMsg(context.Background(), partition2, "consumer1")
	require.NoError(t, err, "expected no error while reading message from partition2")
	assert.NotNil(t, data2, "expected a message to be read from partition2")
	assert.Equal(t, []byte("message for partition2"), data2.Message, "expected message content to match for partition2")
}

func TestRedisStream_MultipleConsumers(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic::%s", 5*time.Second)

	partition := "partition1"
	consumer1 := "consumer1"
	consumer2 := "consumer2"

	// Enqueue a message
	err := rs.EnqueueMsg(context.Background(), partition, &Payload{Message: []byte("test message")})
	require.NoError(t, err, "expected no error while enqueuing message")

	// Read the message with the first consumer
	data1, err := rs.ReadMsg(context.Background(), partition, consumer1)
	require.NoError(t, err, "expected no error while reading message")

	assert.NotNil(t, data1, "expected a message to be read by consumer1")
	assert.Equal(t, []byte("test message"), data1.Message, "expected message content to match for consumer1")

	data2, err := rs.ReadMsg(context.Background(), partition, consumer2)
	require.NoError(t, err, "expected no error while reading message")
	assert.Nil(t, data2, "expected no new message to be read by consumer2")
}

func TestRedisStream_Flush(t *testing.T) {
	client := setupTestRedisClient()
	defer client.FlushDB(context.Background())

	rs := NewRedisStream(client, "testtopic", 5*time.Second)

	// Enqueue a message
	err := rs.EnqueueMsg(context.Background(), "partition1", &Payload{Message: []byte("test message")})
	assert.NoError(t, err, "expected no error while enqueuing message")

	// Flush the database
	err = rs.Flush(context.Background())
	assert.NoError(t, err, "expected no error while flushing database")

	// Attempt to read a message after flushing
	data, err := rs.ReadMsg(context.Background(), "partition1", "consumer1")
	assert.NoError(t, err, "expected no error while reading message")
	assert.Nil(t, data, "expected no message to be read after flushing database")
}
