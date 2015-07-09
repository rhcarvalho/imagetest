package imagetest

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// reuseImages ...
var reuseImages bool

func init() {
	flag.BoolVar(&reuseImages, "reuseimages", false,
		"reuse existing Docker images of test applications")

}

// TestImageFromSource ...
func TestImageFromSource(t *testing.T, builderImage, source, contextDir, outputImage string) {
	if !reuseImages {
		b, err := buildApp(builderImage, source, contextDir, outputImage)
		if err != nil {
			if b != nil {
				t.Fatalf("\n%s\n%s", b, err)
			}
			t.Fatal(err)
		}
	}
	containerID, err := runApp(outputImage)
	if err != nil {
		t.Fatal(err)
	}
	defer stopTestApp(containerID)
	testInParallel(t, []func() error{
		func() error {
			containerIP, err := inspectContainerIP(containerID)
			if err != nil {
				t.Fatal(err)
			}
			url := fmt.Sprintf("http://%s:8080", containerIP)
			return checkHTTPConnectivity(url)
		},
		func() error {
			return checkSCLEnabled(outputImage, containerID)
		},
	})
}

// testInParallel ...
func testInParallel(t *testing.T, fs []func() error) {
	var wg sync.WaitGroup
	for _, f := range fs {
		f := f
		wg.Add(1)
		go func() {
			err := f()
			if err != nil {
				t.Error(err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

// buildApp ...
func buildApp(builderImage, source, contextDir, outputImage string) ([]byte, error) {
	cmd := exec.Command("sti", "build",
		"--force-pull=false",
		fmt.Sprintf("--context-dir=%s", contextDir),
		source, builderImage, outputImage)
	b, err := cmd.CombinedOutput()
	return b, err
}

// runApp ...
func runApp(imageName string) (string, error) {
	cmd := exec.Command("docker", "run",
		"--user=12345", "-p", "8080", "-d", imageName)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, b)
	}
	containerID := strings.TrimSpace(string(b))
	return containerID, nil
}

// stopTestApp ...
func stopTestApp(containerID string) error {
	cmd := exec.Command("docker", "rm", "-f", containerID)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, b)
	}
	return nil
}

// inspectContainerIP ...
func inspectContainerIP(containerID string) (string, error) {
	cmd := exec.Command("docker", "inspect",
		"--format='{{ .NetworkSettings.IPAddress }}'", containerID)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, b)
	}
	return strings.TrimSpace(string(b)), nil
}

// checkHTTPConnectivity ...
func checkHTTPConnectivity(url string) error {
	var (
		resp *http.Response
		err  error
	)
	const maxAttempts = 10
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err = client.Head(url)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP status: got %v, want %v",
				resp.StatusCode, http.StatusOK)
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %v", maxAttempts, err)
}

// CheckSCLEnabled ...
func CheckSCLEnabled(command, expectedOutput string) func(string, string) error {
	return func(imageName, containerID string) error {
		err := checkDockerExecOutputContains(containerID,
			[]string{"bash", "-c", command},
			expectedOutput)
		if err != nil {
			return err
		}
		err = checkDockerRunOutputContains(imageName,
			[]string{"bash", "-c", command},
			expectedOutput)
		return err
	}
}

// checkSCLEnabled ...
func checkSCLEnabled(imageName, containerID string) error {
	err := checkDockerExecOutputContains(containerID,
		[]string{"bash", "-c", "ruby --version"},
		"ruby 2.0.0")
	if err != nil {
		return err
	}
	err = checkDockerRunOutputContains(imageName,
		[]string{"bash", "-c", "ruby --version"},
		"ruby 2.0.0")
	return err
}

func checkDockerRunOutputContains(imageName string, command []string, expectedOutput string) error {
	cmd := exec.Command("docker",
		append([]string{"run", "--rm", imageName}, command...)...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, b)
	}
	if !bytes.Contains(b, []byte(expectedOutput)) {
		return fmt.Errorf("Docker run output: got '%s', want '%s'",
			b, expectedOutput)
	}
	return nil
}

func checkDockerExecOutputContains(containerID string, command []string, expectedOutput string) error {
	cmd := exec.Command("docker",
		append([]string{"exec", containerID}, command...)...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, b)
	}
	if !bytes.Contains(b, []byte(expectedOutput)) {
		return fmt.Errorf("Docker exec output: got '%s', want '%s'",
			b, expectedOutput)
	}
	return nil
}
