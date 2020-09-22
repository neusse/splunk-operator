package testenv

import (
	"fmt"
	"os/exec"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// CheckMCPodReady check if monitoring pod is ready
func CheckMCPodReady(ns string) bool {
	output, err := exec.Command("kubectl", "get", "pod", "-n", ns).Output()
	if err != nil {
		cmd := fmt.Sprintf("kubectl get pods -n %s", ns)
		logf.Log.Error(err, "Failed to execute command", "command", cmd)
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "monitoring-console") {
			slices := strings.Fields(line)
			logf.Log.Info("MC Pod Found", "POD", slices[0], "READY", slices[1], "STATUS", slices[2])
			return strings.Contains(slices[1], "1/1") && strings.Contains(slices[2], "Running")
		}
	}
	return false
}

// GetMCPodCount get count of Monitoring Console in a namespace
func GetMCPodCount(ns string) int {
	output, err := exec.Command("kubectl", "get", "pods", "-n", ns).Output()
	if err != nil {
		cmd := fmt.Sprintf("kubectl get pods -n %s", ns)
		logf.Log.Error(err, "Failed to execute command", "command", cmd)
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "monitoring-console") {
			count = count + 1
		}
	}
	return count
}

// GetMCPodName returns name of monitoring console pod
func GetMCPodName(ns string) string {
	output, err := exec.Command("kubectl", "get", "pods", "-n", ns).Output()
	if err != nil {
		cmd := fmt.Sprintf("kubectl get pods -n %s", ns)
		logf.Log.Error(err, "Failed to execute command", "command", cmd)
		return ""
	}
	podName := ""
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "monitoring-console") {
			slices := strings.Fields(line)
			logf.Log.Info("MC Pod Found", "POD", slices[0], "READY", slices[1], "STATUS", slices[2])
			return slices[0]
		}
	}
	return podName
}

// GetConfiguredPeers get list of Peers Confiugred on Montioring Console
func GetConfiguredPeers(ns string) []string {
	podName := GetMCPodName(ns)
	var peerList []string
	if len(podName) > 0 {
		peerFile := "/opt/splunk/etc/apps/splunk_monitoring_console/local/splunk_monitoring_console_assets.conf"
		output, err := exec.Command("kubectl", "exec", "-n", ns, podName, "--", "cat", peerFile).Output()
		if err != nil {
			cmd := fmt.Sprintf("kubectl exec -n %s %s -- cat %s", ns, podName, peerFile)
			logf.Log.Error(err, "Failed to execute command", "command", cmd)
		}
		for _, line := range strings.Split(string(output), "\n") {
			// Check for empty lines to prevent an error in logic below
			if len(line) == 0 {
				continue
			}
			// configuredPeers only appear in splunk_monitoring_console_assets.conf when peers are configured.
			if strings.Contains(line, "configuredPeers") {
				// Splitting confiugred peers on "=" and then "," to get list of peers configured
				peerString := strings.Trim(strings.Split(line, "=")[1], "")
				peerList = strings.Split(peerString, ",")
				break
			}
		}
	}
	return peerList
}

// DeleteMCPod delete monitoring console deployment
func DeleteMCPod(ns string) {
	output, err := exec.Command("kubectl", "delete", "deployment", "-n", ns, "splunk-default-monitoring-console").Output()
	if err != nil {
		cmd := fmt.Sprintf("kubectl delete deployment -n %s splunk-default-monitoring-console", ns)
		logf.Log.Error(err, "Failed to execute command", "command", cmd)
	} else {
		logf.Log.Info("Monitoring Console Deployment deleted", "Deployment", "splunk-default-monitoring-console", "stdout", output)
	}
}
