package routes

import (
	"arbokcore/core/supervisors"
	"arbokcore/core/tokens"
	"arbokcore/pkg/brokers"
	"arbokcore/pkg/config"
	"arbokcore/web/middlewares"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func SetupSSEeventResponse(c echo.Context) *echo.Response {
	respHeader := c.Response().Header()

	respHeader.Set(echo.HeaderContentType, "text/event-stream")
	respHeader.Set(echo.HeaderCacheControl, "no-cache")
	respHeader.Set(echo.HeaderConnection, "keep-alive")

	return c.Response()
}

type SSEHandler struct {
	subscriber *brokers.SSEBroker
	syncer     *brokers.FileUpdateSyncBroker
}

func NewSSEHandler(cfg config.AppConfig) *SSEHandler {
	subscriber := brokers.NewSSEBroker("file_events")
	subscriber.Start(context.Background())

	metadataProducer, err := brokers.SetupFileEventProducer(cfg)
	if err != nil {
		panic(err)
	}

	syncer := brokers.NewFileUpdateSyncBroker(
		"update_syncer",
		metadataProducer,
		&brokers.SSEConsumer{Dst: subscriber},
	)
	syncer.Start(context.Background())

	return &SSEHandler{
		subscriber: subscriber,
		syncer:     syncer,
	}
}

func (slf *SSEHandler) EstablishConnection(c echo.Context) error {
	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("token validation not done")
		return c.NoContent(http.StatusUnauthorized)
	}

	ctx := c.Request().Context()
	userID := token.ResourceID
	deviceID := c.QueryParam("deviceID")

	w := SetupSSEeventResponse(c)

	topicName := fmt.Sprintf("%s_%s", userID, deviceID)
	receiver := slf.subscriber.Subscribe(ctx, topicName)

	defer func(_topicName string) {
		fmt.Println("deregistering", topicName, userID, deviceID)
		slf.subscriber.Unsubscribe(ctx, _topicName)
	}(topicName)

	fmt.Println("subscribing", topicName, userID, deviceID)
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("done")
			return c.NoContent(http.StatusRequestTimeout)
		case data, ok := <-receiver:
			// fmt.Println(data, ok)
			if !ok {
				// log.Error().Msg("failed to receive sse data")
				continue
			}

			fmt.Println("==================")
			fmt.Println("==================")
			fmt.Println(string(data.Content), deviceID)
			fmt.Println("==================")
			fmt.Println("==================")

			sseData := fmt.Sprintf(
				"userID:%s,deviceID:%s,%s", userID, deviceID, string(data.Content))

			fmt.Fprintf(w, "data: %s\n\n", sseData)
			w.Flush()
		case <-ticker.C:
			// fmt.Println("checking for file updates to device")
			slf.syncer.HandleDemand(ctx, supervisors.Demand{Count: 1, UserID: userID})
		}
	}

}
