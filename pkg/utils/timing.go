package utils

import (
	"time"

	"github.com/rs/zerolog/log"
)

func Bench(start time.Time, msg string) {
	elapesed := time.Since(start)

	log.Debug().Str("time elapsed", elapesed.String()).Msg("end tracking" + msg)
}

func Bench2(msg string) func() {
	start := time.Now()

	log.Debug().Str("start", start.String()).Msg("start tracking" + msg)

	return func() {
		elapesed := time.Since(start)

		log.Debug().Str("time elapsed", elapesed.String()).Msg("end tracking" + msg)
	}
}
