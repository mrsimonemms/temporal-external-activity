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

package main

import (
	"os"

	"external/activities"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

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

func exec() error {
	// Configure logger
	if err := ConfigureZeroLog(); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error configuring logger",
		}
	}

	// Create Temporal connection
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create Temporal client",
		}
	}
	defer c.Close()

	taskQueue, ok := os.LookupEnv("EXTERNAL_TASK_QUEUE")
	if !ok {
		taskQueue = "externalTaskQueue"
	}

	w := worker.New(c, taskQueue, worker.Options{})

	// This only has activities registered
	a := activities.NewActivities()
	w.RegisterActivityWithOptions(a.LongRunningCommand, activity.RegisterOptions{
		Name: "long-running-command",
	})

	log.Info().Msg("Starting worker")
	if err := w.Run(worker.InterruptCh()); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to start the worker",
		}
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}
