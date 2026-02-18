/*
 * Copyright 2026 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wf

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const TaskQueue = "taskQueueName"

func ConfigureZeroLog() error {
	lvl := zerolog.InfoLevel
	if logLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		var err error
		lvl, err = zerolog.ParseLevel(logLevel)
		if err != nil {
			return err
		}
	}

	log.Trace().Str("log_level", lvl.String()).Msg("Set global log level")
	zerolog.SetGlobalLevel(lvl)

	return nil
}

func ExecuteEvery(ctx context.Context, duration time.Duration, fn func(context.Context)) (cctx context.Context, cancel func()) {
	cctx, cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		l := log.With().Ctx(cctx).Dur("duration", duration).Logger()

		for {
			select {
			case <-ticker.C:
				l.Debug().Msg("Triggering background function")
				fn(cctx)
			case <-cctx.Done():
				l.Debug().Msg("Stopping background function")
				return
			}
		}
	}()

	return cctx, cancel
}
