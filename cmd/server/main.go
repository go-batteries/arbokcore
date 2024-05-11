package main

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/tokens"
	"arbokcore/pkg/blobstore"
	"arbokcore/pkg/config"
	"arbokcore/pkg/squirtle"
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

func main() {
	var port string

	flag.StringVar(&port, "port", "9191", "pass the port number")
	flag.Parse()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  []string{"http://localhost"},
		AllowHeaders:  []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, ACCESS_TOKEN_HEADER, STREAM_TOKEN_HEADER},
		ExposeHeaders: []string{echo.HeaderContentLength, echo.HeaderContentDisposition, echo.HeaderContentEncoding},
	}))

	cfg := config.Load("./config/")
	_ = cfg

	conn := database.ConnectSqlite(cfg.DbName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbconn := conn.Connect(ctx)

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
	filesrepo := files.NewMetadataRepository(dbconn, metadataQueryStore)
	filesvc := files.NewMetadataService(filesrepo, metadataTokenRepo)

	chunkQs, err := qs.HydrateQueryStore("user_files")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load user files query")
	}

	localFs, err := blobstore.NewLocalFS("./tmp/arbokdata")
	if err != nil {
		log.Fatal().Err(err).Msg("failed initialized file system")
	}

	chunkRepo := files.NewUserFileRespository(dbconn, chunkQs)
	chunkSvc := files.NewFileChunkService(chunkRepo, localFs)

	metadataHandler := &routes.MetadataHandler{FileSvc: filesvc}
	chunkHandler := &routes.ChunkHandler{ChunkSvc: chunkSvc}

	// Routes
	router := e.Group("arbokcore")

	router.GET("/ping", hello)

	e.PATCH("/my/files/:fileID",
		metadataHandler.UpdateFileMetadata,
		authsvc.ValidateStreamToken,
	)

	e.POST("/my/files",
		metadataHandler.PostFileMetadata,
		authsvc.ValidateAccessToken)

	e.GET("/my/files",
		metadataHandler.GetFileMetadata,
		authsvc.ValidateAccessToken,
	)

	e.PATCH("/my/files/:fileID/chunks",
		chunkHandler.UpsertChunks,
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
