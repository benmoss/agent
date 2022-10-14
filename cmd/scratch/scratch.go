package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/buildkite/agent/v3/agent/plugin"
	"github.com/buildkite/agent/v3/api"
	"github.com/buildkite/agent/v3/logger"
	"github.com/sanity-io/litter"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	toolswatch "k8s.io/client-go/tools/watch"
)

type worker struct {
	name   string
	logger logger.Logger
	client *kubernetes.Clientset
}

func main() {
	log := logger.NewConsoleLogger(logger.NewTextPrinter(os.Stderr), os.Exit)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		<-c
		cancel()
	}()

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		log.Error("failed to create client config: %v", err)
		return
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Error("failed to create clienset: %v", err)
		return
	}
	var wg sync.WaitGroup
	workers := 1
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		name := fmt.Sprintf("worker-%d", i)
		w := worker{
			client: clientset,
			logger: log.WithFields(logger.StringField("worker", name)),
			name:   name,
		}
		go w.run(ctx, &wg)
	}
	wg.Wait()
}

func (w *worker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	client := api.NewClient(w.logger, api.Config{
		Endpoint:  "https://agent.buildkite.com/v3",
		Token:     os.Getenv("BUILDKITE_TOKEN"),
		UserAgent: "buildkite-agent/3.39.0.x (darwin; arm64)",
	})
	resp, _, err := client.Register(&api.AgentRegisterRequest{
		Name: w.name,
		OS:   "wtf",
		Arch: "wtf",
		Tags: []string{"queue=kubernetes"},
	})
	if err != nil {
		w.logger.Error("register: %v", err)
		return
	}
	w.logger.Info("register: %v", litter.Sdump(resp))
	client = client.FromAgentRegisterResponse(resp)
	_, err = client.Connect()
	if err != nil {
		w.logger.Error("connect: %v", err)
		return
	}
	defer client.Disconnect()
	for {
		select {
		case <-ctx.Done():
			w.logger.Error("context cancelled: %v", ctx.Err())
			return
		default:
			time.Sleep(time.Second)
			// continue
		}
		resp, _, err := client.Ping()
		if err != nil {
			w.logger.Error("ping: %v", err)
			return
		}
		if resp.Job != nil {
			job, _, err := client.AcceptJob(resp.Job)
			if err != nil {
				w.logger.Error("accept: %v", err)
				return
			}
			if job.State == "running" {
				continue
			}
			_, err = client.StartJob(resp.Job)
			if err != nil {
				w.logger.Error("start: %v", err)
			}
			w.logger.Info("start: %v", litter.Sdump(job))
			plugins, err := plugin.CreateFromJSON(job.Env["BUILDKITE_PLUGINS"])
			if err != nil {
				w.logger.Warn("err converting plugins to json: %v", err)
				var env []corev1.EnvVar
				for k, v := range job.Env {
					env = append(env, corev1.EnvVar{Name: k, Value: v})
				}
				env = append(env, corev1.EnvVar{
					Name:  "BUILDKITE_BUILD_PATH",
					Value: "/buildkite/builds",
				})
				env = append(env, corev1.EnvVar{
					Name:  "BUILDKITE_AGENT_ACCESS_TOKEN",
					Value: client.Config().Token,
				})
				pod, err := w.client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "agent-",
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Name:  "agent",
								Image: "buildkite/agent:latest",
								Args: []string{
									"bootstrap",
								},
								Env: env,
							},
						},
					},
				}, metav1.CreateOptions{})
				if err != nil {
					w.logger.Error("failed to create pod: %v", err)
					return
				}
				w.logger.Info("created pod: %s", pod.Name)
				fs := fields.OneTermEqualSelector(metav1.ObjectNameField, pod.Name)
				lw := &cache.ListWatch{
					ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
						options.FieldSelector = fs.String()
						return w.client.CoreV1().Pods(pod.Namespace).List(context.TODO(), options)
					},
					WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
						options.FieldSelector = fs.String()
						return w.client.CoreV1().Pods("default").Watch(ctx, options)
					},
				}
				_, err = toolswatch.UntilWithSync(ctx, lw, &corev1.Pod{}, nil, func(ev watch.Event) (bool, error) {
					if pod, ok := ev.Object.(*corev1.Pod); ok {
						if pod.Status.Phase == corev1.PodSucceeded {
							w.logger.Info("pod success!")
							return true, nil
						}
						w.logger.Info("pod not success! status: %s", pod.Status.Phase)
						job.ExitStatus = "0"
						return false, nil
					}
					return false, errors.New("event object not of type v1.Node")
				})
				if err != nil {
					w.logger.Error("failed to watch pod: %v", err)
					return
				}
				if _, err := client.FinishJob(job); err != nil {
					w.logger.Error("failed to finish job: %v", err)
					return
				}
			}

			// "BUILDKITE_PLUGINS":                            "[{\"github.com/buildkite-plugins/shellcheck-buildkite-plugin\":{\"files\":[\"hooks/**\",\"lib/**\",\"commands/**\"]}}]",
			for _, plugin := range plugins {
				w.logger.Info("plugin: %v", litter.Sdump(plugin))
				var podSpec corev1.PodSpec
				asJson, err := json.Marshal(plugin.Configuration)
				if err != nil {
					w.logger.Error("failed to marshal config: %v", err)
					return
				}
				if err := json.Unmarshal(asJson, &podSpec); err != nil {
					w.logger.Error("failed to unmarshal config: %v", err)
					return
				}
				w.logger.Info("podSpec: %v", litter.Sdump(podSpec))
			}

			// 	w.logger.Info("accept job: %v", litter.Sdump(job))
			// 	_, err = client.StartJob(resp.Job)
			// 	if err != nil {
			// 		w.logger.Error("start: %v", err)
			// 	}
			// 	_, err = client.UploadChunk(job.ID, &api.Chunk{
			// 		Data:     "heyo",
			// 		Sequence: 0,
			// 		Offset:   0,
			// 		Size:     len("heyo"),
			// 	})
			// 	if err != nil {
			// 		w.logger.Error("upload chunk: %v", err)
			// return
			// 	}
			// 	_, err = client.FinishJob(job)
			// 	if err != nil {
			// 		w.logger.Error("finish: %v", err)
			// return
			// 	}
		}
	}
}

type Kubernetes struct {
	Name string
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
