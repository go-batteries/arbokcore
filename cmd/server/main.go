package main

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/tokens"
	"arbokcore/pkg/blobstore"
	"arbokcore/pkg/config"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/squirtle"
	"arbokcore/pkg/ssebroker"
	"arbokcore/web/middlewares"
	"arbokcore/web/routes"
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/rs/zerolog/log"
)

const (
	ACCESS_TOKEN_HEADER = "X-Access-Token"
	STREAM_TOKEN_HEADER = "X-Stream-Token"
)

func SetupSSEeventResponse(c echo.Context) *echo.Response {
	respHeader := c.Response().Header()

	respHeader.Set(echo.HeaderContentType, "text/event-stream")
	respHeader.Set(echo.HeaderCacheControl, "no-cache")
	respHeader.Set(echo.HeaderConnection, "keep-alive")

	return c.Response()
}

func main() {
	var port string

	flag.StringVar(&port, "port", "9191", "pass the port number")
	flag.Parse()

	cfg := config.Load("./config/")
	_ = cfg

	conn := database.ConnectSqlite(cfg.DbName)

	ctx := context.Background()
	dbconnctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dbconn := conn.Connect(dbconnctx)

	qs := squirtle.LoadAll("./config/querystore.yaml")

	tokensQs, err := qs.HydrateQueryStore("tokens")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize tokens query store")
	}

	tokensRepo := tokens.NewTokensRepository(dbconn, tokensQs)

	authsvc := middlewares.NewAuthMiddleWareService(tokensRepo)

	metadataQueryStore, err := qs.HydrateQueryStore("file_metadatas")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load file metadatas query")
	}

	metadataTokenRepo := files.NewMetadataTokenRepository(
		dbconn,
		metadataQueryStore,
		tokensQs,
	)

	redisConn, err := database.NewRedisConnection(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connected to redis")
	}

	metadataQ := queuer.NewRedisQ(
		redisConn,
		database.MetadataRedisQueue,
		20*time.Second,
	)

	filesrepo := files.NewMetadataRepository(dbconn, metadataQueryStore)
	filesvc := files.NewMetadataService(filesrepo, metadataTokenRepo, metadataQ)

	chunkQs, err := qs.HydrateQueryStore("user_files")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load user files query")
	}

	chunkRepo := files.NewUserFileRespository(dbconn, chunkQs)

	localFs, err := blobstore.NewLocalFS("./tmp/arbokdata")
	if err != nil {
		log.Fatal().Err(err).Msg("failed initialized file system")
	}

	chunkSvc := files.NewFileChunkService(chunkRepo, localFs)

	metadataHandler := &routes.MetadataHandler{FileSvc: filesvc}
	chunkHandler := &routes.ChunkHandler{ChunkSvc: chunkSvc}

	connBrokers := ssebroker.NewBroker()
	connBrokers.Start(ctx)

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderContentDisposition,
			echo.HeaderConnection,
			echo.HeaderCacheControl,
			ACCESS_TOKEN_HEADER,
			STREAM_TOKEN_HEADER,
		},
		ExposeHeaders: []string{
			echo.HeaderContentLength,
			echo.HeaderContentDisposition,
			echo.HeaderContentEncoding,
			echo.HeaderContentType,
			echo.HeaderCacheControl,
			echo.HeaderConnection,
		},
	}))

	// Routes
	router := e.Group("arbokcore")

	router.GET("/ping", hello)

	e.GET("/subscribe/devices", func(c echo.Context) error {
		token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
		if !ok {
			log.Error().Msg("token validation not done")
			return c.NoContent(http.StatusUnauthorized)
		}

		userID := token.ResourceID
		deviceID := c.QueryParam("deviceID")

		ok = connBrokers.AddConnection(userID, deviceID)
		defer connBrokers.RemoveConnection(userID, deviceID)

		connBrokers.Print()

		w := SetupSSEeventResponse(c)

		ctx := c.Request().Context()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("done")
				return c.NoContent(http.StatusRequestTimeout)
			default:
				msg, ok := connBrokers.GetMessage(userID, deviceID)
				if !ok {
					log.Info().Msg("fucked")

					// connBrokers.RemoveConnection(userID, deviceID)
					continue
				}
				log.Info().Msg("sending data")
				fmt.Fprintf(w, "data: %s\n\n", msg)
				w.Flush()
			}
		}

	}, authsvc.ValidateAccessToken)

	go func() {
		ctx := context.Background()

		for {
			time.Sleep(5 * time.Second)

			connBrokers.SendMessage(ctx, ssebroker.Message{
				UserID:   tokens.AdminToken.ResourceID,
				DeviceID: "1",
				Content:  []byte("file_id:1|status:success"),
			})

			connBrokers.SendMessage(ctx, ssebroker.Message{
				UserID:   tokens.AdminToken.ResourceID,
				DeviceID: "2",
				Content:  []byte("file_id:2|status:success"),
			})

			// connBrokers.SendMessage(ctx, ssebroker.Message{
			// 	UserID:   tokens.AnotherToken.ResourceID,
			// 	DeviceID: "1",
			// 	Content:  []byte("file_id:3|status:success"),
			// })
			// connBrokers.SendMessage(ctx, ssebroker.Message{
			// 	UserID:   tokens.AnotherToken.ResourceID,
			// 	DeviceID: "2",
			// 	Content:  []byte("file_id:4|status:success"),
			// })
		}

	}()

	e.PATCH("/my/files/:fileID",
		metadataHandler.UpdateFileMetadata,
		authsvc.ValidateAccessToken,
	)

	e.POST("/my/files",
		metadataHandler.PostFileMetadata,
		authsvc.ValidateAccessToken)

	e.GET("/my/files",
		metadataHandler.GetFileMetadata,
		authsvc.ValidateAccessToken,
	)

	e.PUT("/my/files/:fileID/eof",
		metadataHandler.MarkUploadComplete,
		authsvc.ValidateStreamToken,
	)

	e.PATCH("/my/files/:fileID/chunks",
		chunkHandler.UpsertChunks,
		authsvc.ValidateStreamToken,
	)

	e.GET("/my/files/:fileID/download",
		metadataHandler.DownloadFile,
		authsvc.AddTokenFromUrlToHeader,
		authsvc.ValidateStreamToken,
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: e,
	}

	go func() {
		log.Info().Str("port", port).Msg("server started at")

		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	appCtx := context.Background()
	ctx, stop := signal.NotifyContext(appCtx, os.Interrupt)
	defer stop()

	<-ctx.Done()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error during server shutdown")
	}
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
