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

// This workflow exists to create a long-running activity hosted somewhere else
// This could be use to simulate running a long-running Kubernetes job or in
// an AWS BatchJob
func TriggerExternalWorkflow(ctx workflow.Context) error {
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

	logger.Debug("Having a small sleep")
	if err := workflow.Sleep(ctx, 5*time.Second); err != nil {
		return fmt.Errorf("error sleeping: %w", err)
	}

	logger.Debug("Create the external activity runner")
	if err := workflow.ExecuteActivity(ctx, a.Create, heartbeatTimeout).Get(ctx, nil); err != nil {
		logger.Error("Error running create activity", "error", err)
		return fmt.Errorf("error running create activity: %w", err)
	}

	opts := workflow.GetActivityOptions(ctx)
	// @todo(sje): this will be an ID from previous activity
	opts.TaskQueue = "externalTaskQueue"
	actx := workflow.WithActivityOptions(ctx, opts)

	if err := workflow.ExecuteActivity(actx, "long-running-command", time.Minute, heartbeatTimeout).Get(actx, nil); err != nil {
		logger.Error("Error running external activity", "error", err)
		return fmt.Errorf("error running external activity: %w", err)
	}

	return nil
}
