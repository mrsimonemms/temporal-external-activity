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
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ExternalWorkflowRequest struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

// This workflow exists to create a long-running activity hosted somewhere else
// This could be use to simulate running a long-running Kubernetes job or in
// an AWS BatchJob
func TriggerExternalWorkflow(ctx workflow.Context, data ExternalWorkflowRequest) (retErr error) {
	name := workflow.GetInfo(ctx).WorkflowType.Name
	logger := workflow.GetLogger(ctx)

	logger.Info("Trigger External workflow started", "name", name)

	heartbeatTimeout := time.Second * 10

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour,
		HeartbeatTimeout:    heartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	})

	var a *Activities

	// Run any activities before we need the long-running activity - this example
	// just has a sleep
	logger.Debug("Having a small sleep")
	if err := workflow.Sleep(ctx, 5*time.Second); err != nil {
		return fmt.Errorf("error sleeping: %w", err)
	}

	// Now create the long running app
	logger.Debug("Create the external activity runner")
	var result *CreateResponse
	if err := workflow.ExecuteActivity(ctx, a.Create, data).Get(ctx, &result); err != nil {
		logger.Error("Error running create activity", "error", err)
		return fmt.Errorf("error running create activity: %w", err)
	}

	// Once we get to here, we need to clean up the activity runner - the
	// defer function ensures we always run this before this workflow ends
	defer func() {
		logger.Debug("Cleaning up the external runner")
		if err := workflow.ExecuteActivity(ctx, a.Delete, result.App.ID).Get(ctx, nil); err != nil {
			logger.Error("Error running delete activity", "error", err)
			if retErr == nil {
				retErr = err
			}
		}

		logger.Info("Trigger External workflow finished", "name", name)
	}()

	// Wait until the app is deployed and ready
	logger.Debug("Wait until app is built", "app", result.App.ID)
	if err := workflow.ExecuteActivity(ctx, a.WaitForAppBuild, result.App.ID, heartbeatTimeout).Get(ctx, nil); err != nil {
		logger.Error("Error waiting for app to be built", "error", err, "app", result.App.ID)
		return fmt.Errorf("error waiting for app to be built: %w", err)
	}

	// Configure the activity to call the activities on the long-running app
	opts := workflow.GetActivityOptions(ctx)
	opts.TaskQueue = result.TaskQueue
	appCtx := workflow.WithActivityOptions(ctx, opts)

	// Call the long-running command on the app
	if err := workflow.ExecuteActivity(appCtx, "long-running-command", time.Minute, heartbeatTimeout).Get(appCtx, nil); err != nil {
		logger.Error("Error running external activity", "error", err)
		return fmt.Errorf("error running external activity: %w", err)
	}

	// Run any activities before we need the long-running activity - this example
	// just has a sleep. Notice how we use the `ctx` context rather than the
	// `appCtx`
	logger.Debug("Having a final small sleep")
	if err := workflow.Sleep(ctx, 5*time.Second); err != nil {
		return fmt.Errorf("error sleeping: %w", err)
	}

	return
}
