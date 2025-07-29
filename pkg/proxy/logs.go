package proxy

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/mhrabovcin/troubleshoot-live/pkg/bundle"
)

// LogsHandler serves logs for k8s `logs` subresource from the provided bundle.
func LogsHandler(b bundle.Bundle, l *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		pod := vars["pod"]
		namespace := vars["namespace"]

		container := r.URL.Query().Get("container")
		previous := r.URL.Query().Get("previous")

		podFilePath := filepath.Join(b.Layout().ClusterResources(), "pods", namespace+".yaml")
		list, err := bundle.LoadResourcesFromFile(b, podFilePath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load pod %s from path %s: %v", pod, podFilePath, err), http.StatusInternalServerError)
			return
		}

		// PRIO1: logs collected from /var/log/pods
		// annotation for the pod logs path is stored in the pod resource[kubernetes.io/config.hash] for etcd/api-server/controller-manager
		var configHash string
		var podUID string
		var restartCount int32
		for _, item := range list.Items {
			if item.GetName() != pod {
				continue
			}

			var pod v1.Pod
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &pod)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to convert unstructured object to pod: %v", err), http.StatusInternalServerError)
				return
			}

			podUID = string(pod.GetUID())
			configHash = pod.GetAnnotations()["kubernetes.io/config.hash"]
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == container {
					restartCount = containerStatus.RestartCount
					break
				}
			}
		}

		// Search for pod logs path in the bundle which could be collected either by the
		// pod logs collector or by the cluster resources collector, which collects pod logs
		// for failing pods. (troubleshoot.sh specific)
		filename := fmt.Sprintf("%s-%s.log", pod, container)
		candidatePaths := []string{
			filepath.Join(b.Layout().PodLogs(), namespace, filename),
			filepath.Join(b.Layout().ClusterResources(), "pods/logs", namespace, pod, container+".log"),
		}

		if previous != "true" {
			// log files to search for
			// 1. k8s/cluster-info/dump/namespace/podName/logs.txt (from the k8s cluster-info dump)
			// 2. k8s/pod-logs/namespace_podName_podUID/container/restartCount.log (/var/log/pods)
			// 3. k8s/pod-logs/namespace_podName_podUID/container/configHash.log (/var/log/pods)

			candidatePaths = append(candidatePaths,
				filepath.Join(b.Layout().ClusterInfo(), "dump", namespace, pod, "logs.txt"),
				filepath.Join(b.Layout().PodLogs(), fmt.Sprintf("%s_%s_%s", namespace, pod, podUID), container, fmt.Sprintf("%d.log", restartCount)),
				filepath.Join(b.Layout().PodLogs(), fmt.Sprintf("%s_%s_%s", namespace, pod, podUID), container, fmt.Sprintf("%s.log", configHash)),
			)
		} else {
			// log files to search for
			// 1. k8s/pod-logs/namespace_podName_podUID/container/restartCount-1.log (/var/log/pods)
			// 2. k8s/previous-pod-logs/namespace/pod_name/previous.log
			if restartCount > 1 {
				candidatePaths = append(candidatePaths,
					filepath.Join(b.Layout().PodLogs(), fmt.Sprintf("%s_%s_%s", namespace, pod, podUID), container, fmt.Sprintf("%d.log", restartCount-1)),
				)
			}
			candidatePaths = append(candidatePaths,
				filepath.Join(b.Layout().PreviousPodLogs(), namespace, pod, "previous.log"),
			)
		}

		podLogsPath := ""
		for _, candidatePath := range candidatePaths {
			if exists, _ := afero.Exists(b, candidatePath); exists {
				podLogsPath = candidatePath
				break
			} else {
				l.Debug("pod logs not found in the bundle", "path", candidatePath)
			}
		}

		if podLogsPath == "" {
			l.Debug("pod logs not found in the bundle", "paths", candidatePaths)
			http.Error(w, "pod logs not found in the bundle", http.StatusInternalServerError)
			return
		}

		l.Debug("pod logs path", "path", podLogsPath)
		data, err := afero.ReadFile(b, podLogsPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		l := l.With("url", r.URL, "logs source", podLogsPath)

		// By default the `k9s` requests logs prefixed with timestamp and in the logs pane
		// only displays a portion without the timestamp, by cutting prefix separated by first
		// space byte(' '). The troubleshoot.sh requests logs without timestamps, which causes
		// issues in the logs pane and for some pods the logs are cut from beginnging.
		// This will backfill zeroed timestamp for each line.
		if r.URL.Query().Get("timestamps") == "true" {
			lines := bytes.Split(data, []byte("\n"))
			timestampPrefixRegexp := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{6})?Z `)
			if !timestampPrefixRegexp.Match(lines[0]) {
				l.Debug("adding timestamp prefix to logs")
				zeroTime := []byte(time.UnixMicro(0).Format(time.RFC3339Nano))
				// Add prefix to each line.
				for i := range lines {
					lines[i] = bytes.Join([][]byte{zeroTime, lines[i]}, []byte{' '})
				}
				data = bytes.Join(lines, []byte("\n"))
			}
		}

		l.Debug("serving logs")
		if _, err := w.Write(data); err != nil {
			slog.Error("failed to write response data", "err", err)
		}
	}
}
