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
	"fmt"
	"math/rand"
	"time"

	"github.com/digitalocean/godo"
	"go.temporal.io/sdk/activity"
)

type Activities struct {
	do              *godo.Client
	temporalEnvVars map[string]string
}

type CreateResponse struct {
	App       *godo.App
	TaskQueue string
}

func (a *Activities) Create(ctx context.Context, data ExternalWorkflowRequest) (*CreateResponse, error) {
	logger := activity.GetLogger(ctx)

	// Generate a random 6 digit number
	n := rand.Intn(900000) + 100000

	// Trim to 32 characters
	name := fmt.Sprintf("external-app-%d", n)

	logger.Info("Building DigitalOcean App", "name", name)

	env := []*godo.AppVariableDefinition{
		{
			Key:   "EXTERNAL_TASK_QUEUE",
			Value: name,
			Scope: godo.AppVariableScope_RunTime,
			Type:  godo.AppVariableType_General,
		},
	}
	for k, v := range a.temporalEnvVars {
		if v != "" {
			env = append(env, &godo.AppVariableDefinition{
				Key:   k,
				Value: v,
				Scope: godo.AppVariableScope_RunTime,
				Type:  godo.AppVariableType_General,
			})
		}
	}

	spec := &godo.AppCreateRequest{
		Spec: &godo.AppSpec{
			Name: name,
			Workers: []*godo.AppWorkerSpec{
				{
					Name:             "external-worker",
					InstanceCount:    1,
					InstanceSizeSlug: "basic-xxs",
					Image: &godo.ImageSourceSpec{
						RegistryType: godo.ImageSourceSpecRegistryType_DOCR,
						Registry:     data.Registry,
						Repository:   data.Repository,
						Tag:          data.Tag,
					},
					Envs: env,
				},
			},
		},
	}

	app, _, err := a.do.Apps.Create(ctx, spec)
	if err != nil {
		logger.Error("Error creating app", "error", err)
		return nil, fmt.Errorf("error creating app: %w", err)
	}

	return &CreateResponse{
		App:       app,
		TaskQueue: name,
	}, nil
}

func (a *Activities) Delete(ctx context.Context, appID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Deleting app", "app", appID)

	_, err := a.do.Apps.Delete(ctx, appID)
	if err != nil {
		logger.Error("Error deleting app", "error", err)
		return fmt.Errorf("error deleting app: %w", err)
	}
	return nil
}

func (a *Activities) WaitForAppBuild(ctx context.Context, appID string, heartbeatTimeout time.Duration) error {
	logger := activity.GetLogger(ctx)

	// Start a heartbeat - make it half the heartbeat timeout
	_, cancel := ExecuteEvery(ctx, heartbeatTimeout/2, func(hctx context.Context) {
		logger.Debug("Sending heartbeat")
		activity.RecordHeartbeat(hctx)
	})
	defer cancel()

	for {
		app, _, err := a.do.Apps.Get(ctx, appID)
		if err != nil {
			logger.Error("Error getting app", "error", err, "app", appID)
			return fmt.Errorf("error getting app: %w", err)
		}

		deployment := app.ActiveDeployment
		if deployment != nil {
			switch deployment.Phase {
			case godo.DeploymentPhase_Active:
				logger.Info("App is built", "app", appID)
				return nil
			case godo.DeploymentPhase_Error, godo.DeploymentPhase_Canceled, godo.DeploymentPhase_Superseded:
				logger.Error("Deployment failed", "phase", deployment.Phase, "app", appID)
				return fmt.Errorf("deployment failed with phase: %s", deployment.Phase)
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func NewActivities(doToken string, temporalEnvVars map[string]string) (a *Activities, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %w", err)
		}
	}()

	a = &Activities{
		do:              godo.NewFromToken(doToken),
		temporalEnvVars: temporalEnvVars,
	}

	return
}
