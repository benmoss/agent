package main

import (
	"os"
	"time"

	"github.com/buildkite/agent/v3/api"
	"github.com/buildkite/agent/v3/logger"
	"github.com/sanity-io/litter"
)

func main() {
	log := logger.NewConsoleLogger(logger.NewTextPrinter(os.Stderr), os.Exit)
	client := api.NewClient(log, api.Config{
		Endpoint:  "https://agent.buildkite.com/v3",
		Token:     os.Getenv("TOKEN"),
		UserAgent: "buildkite-agent/3.39.0.x (darwin; arm64)",
		// DebugHTTP: true,
	})
	resp, _, err := client.Register(&api.AgentRegisterRequest{
		Name: "bmo",
		OS:   "wtf",
		Arch: "wtf",
		Tags: []string{"role=kaniko"},
	})
	if err != nil {
		log.Fatal("register: %v", err)
	}
	litter.Dump(resp)
	client = client.FromAgentRegisterResponse(resp)
	_, err = client.Connect()
	if err != nil {
		log.Fatal("connect: %v", err)
	}
	defer client.Disconnect()

	for i := 0; i < 10; i++ {
		resp, _, err := client.Ping()
		if err != nil {
			log.Fatal("ping: %v", err)
		}
		litter.Dump(resp)
		time.Sleep(time.Second)
		if resp.Job != nil {
			job, _, err := client.AcceptJob(resp.Job)
			if err != nil {
				log.Fatal("accept: %v", err)
			}
			litter.Dump(job)
			_, err = client.StartJob(resp.Job)
			if err != nil {
				log.Fatal("start: %v", err)
			}
			client.UploadChunk(job.ID, &api.Chunk{
				Data:     "heyo",
				Sequence: 0,
				Offset:   0,
				Size:     len("heyo"),
			})
			time.Sleep(5 * time.Minute)
			_, err = client.FinishJob(job)
			if err != nil {
				log.Fatal("finish: %v", err)
			}
		}
	}
}

var sampleJob = &api.Job{
	ID:       "0183d1b7-29b2-452b-b308-ac5e69394a28",
	Endpoint: "https://agent.buildkite.com/v3",
	State:    "accepted",
	Env: map[string]string{
		"BUILDKITE":                                    "true",
		"BUILDKITE_AGENT_ID":                           "0183d1b8-18fc-4831-bfd2-057763c13a3a",
		"BUILDKITE_AGENT_META_DATA_QUEUE":              "default",
		"BUILDKITE_AGENT_META_DATA_ROLE":               "kaniko",
		"BUILDKITE_AGENT_NAME":                         "bmo",
		"BUILDKITE_ARTIFACT_PATHS":                     "",
		"BUILDKITE_BRANCH":                             "master",
		"BUILDKITE_BUILD_AUTHOR":                       "",
		"BUILDKITE_BUILD_AUTHOR_EMAIL":                 "",
		"BUILDKITE_BUILD_CREATOR":                      "Ben Moss",
		"BUILDKITE_BUILD_CREATOR_EMAIL":                "ben.moss@superorbital.io",
		"BUILDKITE_BUILD_ID":                           "0183d1b5-03ef-4db7-8b74-60d1e3d67c1f",
		"BUILDKITE_BUILD_NUMBER":                       "54",
		"BUILDKITE_BUILD_URL":                          "https://buildkite.com/superorbital/bmo/builds/54",
		"BUILDKITE_COMMAND":                            "",
		"BUILDKITE_COMMIT":                             "9f139aaf7f6e6bc66058a0ca0d9fba1495f9375d",
		"BUILDKITE_JOB_ID":                             "0183d1b7-29b2-452b-b308-ac5e69394a28",
		"BUILDKITE_LABEL":                              ":shell: Shellcheck",
		"BUILDKITE_MESSAGE":                            "explore",
		"BUILDKITE_ORGANIZATION_SLUG":                  "superorbital",
		"BUILDKITE_PIPELINE_DEFAULT_BRANCH":            "master",
		"BUILDKITE_PIPELINE_ID":                        "01835c15-60c7-47fc-a5f0-7e91753c853a",
		"BUILDKITE_PIPELINE_NAME":                      "bmo",
		"BUILDKITE_PIPELINE_PROVIDER":                  "github",
		"BUILDKITE_PIPELINE_SLUG":                      "bmo",
		"BUILDKITE_PLUGINS":                            "[{\"github.com/buildkite-plugins/shellcheck-buildkite-plugin\":{\"files\":[\"hooks/**\",\"lib/**\",\"commands/**\"]}}]",
		"BUILDKITE_PROJECT_PROVIDER":                   "github",
		"BUILDKITE_PROJECT_SLUG":                       "superorbital/bmo",
		"BUILDKITE_PULL_REQUEST":                       "false",
		"BUILDKITE_PULL_REQUEST_BASE_BRANCH":           "",
		"BUILDKITE_PULL_REQUEST_REPO":                  "",
		"BUILDKITE_REBUILT_FROM_BUILD_ID":              "",
		"BUILDKITE_REBUILT_FROM_BUILD_NUMBER":          "",
		"BUILDKITE_REPO":                               "https://github.com/benmoss/docker-compose-buildkite-plugin",
		"BUILDKITE_RETRY_COUNT":                        "0",
		"BUILDKITE_SCRIPT_PATH":                        "",
		"BUILDKITE_SOURCE":                             "ui",
		"BUILDKITE_STEP_ID":                            "0183d1b7-2983-41c3-bf89-88e444aae4ab",
		"BUILDKITE_STEP_KEY":                           "",
		"BUILDKITE_TAG":                                "",
		"BUILDKITE_TIMEOUT":                            "false",
		"BUILDKITE_TRIGGERED_FROM_BUILD_ID":            "",
		"BUILDKITE_TRIGGERED_FROM_BUILD_NUMBER":        "",
		"BUILDKITE_TRIGGERED_FROM_BUILD_PIPELINE_SLUG": "",
		"CI": "true",
	},
	ChunksMaxSizeBytes: 102400,
	Token:              "",
	ExitStatus:         "",
	Signal:             "",
	SignalReason:       "",
	StartedAt:          "",
	FinishedAt:         "",
	RunnableAt:         "2022-10-13T14:19:45.482Z",
	ChunksFailedCount:  0,
}
