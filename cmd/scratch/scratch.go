package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const ns = "default"

var defaultBootstrapPod = &corev1.Pod{
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
			},
		},
	},
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
	registerResp, _, err := client.Register(&api.AgentRegisterRequest{
		Name: w.name,
		Tags: []string{"queue=kubernetes"},
	})
	if err != nil {
		w.logger.Error("register: %v", err)
		return
	}
	client = client.FromAgentRegisterResponse(registerResp)
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
			time.Sleep(time.Duration(registerResp.PingInterval) * time.Second)
			// continue
		}
		resp, _, err := client.Ping()
		if err != nil {
			w.logger.Warn("ping: %v", err)
			continue
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
			pod, err := w.podFromJob(job, client)
			if err != nil {
				w.logger.Error("podFromJob: %v", err)
				return
			}
			pod, err = w.client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
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
					return w.client.CoreV1().Pods(ns).Watch(ctx, options)
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
			req := w.client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
			podLogs, err := req.Stream(ctx)
			if err != nil {
				w.logger.Error("error in opening stream: %v", err)
				return
			}
			defer podLogs.Close()

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				w.logger.Error("error in copy information from podLogs to buf: %v", err)
				return
			}
			str := buf.String()
			_, err = client.UploadChunk(job.ID, &api.Chunk{
				Data:     str,
				Sequence: 0,
				Offset:   0,
				Size:     len(str),
			})
			if err != nil {
				w.logger.Error("upload chunk: %v", err)
				return
			}
			if _, err := client.FinishJob(job); err != nil {
				w.logger.Error("failed to finish job: %v", err)
				return
			}
		}
	}
}

func (w *worker) podFromJob(job *api.Job, client *api.Client) (*corev1.Pod, error) {
	var pod *corev1.Pod
	if job.Env["BUILDKITE_PLUGINS"] == "" {
		w.logger.Warn("no plugins specified, using default bootstrap pod")
		pod = defaultBootstrapPod
	} else {
		plugins, err := plugin.CreateFromJSON(job.Env["BUILDKITE_PLUGINS"])
		if err != nil {
			return nil, fmt.Errorf("err converting plugins to json: %w", err)
		} else {
			// create regular pod
			// "BUILDKITE_PLUGINS":                            "[{\"github.com/buildkite-plugins/shellcheck-buildkite-plugin\":{\"files\":[\"hooks/**\",\"lib/**\",\"commands/**\"]}}]",
			for _, plugin := range plugins {
				w.logger.Info("plugin: %v", litter.Sdump(plugin))
				var podSpec corev1.PodSpec
				asJson, err := json.Marshal(plugin.Configuration)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal config: %w", err)
				}
				if err := json.Unmarshal(asJson, &podSpec); err != nil {
					return nil, fmt.Errorf("failed to unmarshal config: %w", err)
				}
				w.logger.Info("podSpec: %v", litter.Sdump(podSpec))
				pod = &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("buildkite-%s", job.ID),
					},
					Spec: podSpec,
				}
			}
		}
	}
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:  "bootstrap",
		Image: "buildkite/agent:latest",
		Args: []string{
			"bootstrap", "--phases=checkout",
		},
	})
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "workspace",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	env := make([]corev1.EnvVar, 0, len(job.Env)+2)
	env = append(env, corev1.EnvVar{
		Name:  "BUILDKITE_BUILD_CHECKOUT_PATH",
		Value: "/workspace",
	}, corev1.EnvVar{
		Name:  "BUILDKITE_AGENT_ACCESS_TOKEN",
		Value: client.Config().Token,
	})
	for k, v := range job.Env {
		env = append(env, corev1.EnvVar{Name: k, Value: v})
	}
	volumeMounts := []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}}
	for i, c := range pod.Spec.Containers {
		c.Env = append(c.Env, env...)
		if c.Name == "" {
			c.Name = fmt.Sprintf("container-%d", i)
		}
		c.VolumeMounts = append(c.VolumeMounts, volumeMounts...)
		pod.Spec.Containers[i] = c
	}
	for i, c := range pod.Spec.InitContainers {
		c.Env = append(c.Env, env...)
		if c.Name == "" {
			c.Name = fmt.Sprintf("container-%d", i)
		}
		c.VolumeMounts = append(c.VolumeMounts, volumeMounts...)
		pod.Spec.InitContainers[i] = c
	}
	return pod, nil
}
