package workers

// func MetadataUpdateNotification(ctx context.Context, cfg config.AppConfig) error {
// 	redisconn, err := database.NewRedisConnection(cfg.RedisURL)
//
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to connect to redis")
// 		return err
// 	}
//
// 	ctx = context.Background()
//
// 	nsq := queuer.NewRedisQ(
// 		redisconn,
// 		database.MetadataUpdateClientsNotifierQueue,
// 		1*time.Second,
// 	)
//
// 	producer := supervisors.NewMetadataNotifier(nsq)
//
// 	return nil
// }
