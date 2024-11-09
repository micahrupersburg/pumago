package index

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"pumago/config"
	"strconv"
	"syscall"
)

func fork(binary string, args []string) error {
	cmd := exec.Command(binary, args...)
	if flag.Lookup("verbose").Value.String() == "true" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	log.Printf("Launching LLama: %v", cmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting LLama: %v", err)
		return err
	}

	// Handle termination signals to ensure llama-server is terminated
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		<-sigChan
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}()

	// this is important, otherwise the process becomes in S mode
	go func() {
		err = cmd.Wait()
		fmt.Printf("Cmd %+v finished with error: %v", cmd, err)
	}()

	return nil
}
func (index *Index) Launch() error {
	binary := filepath.Join(config.BinDir(), "llama-server")
	args := []string{"--model", filepath.Join(config.Dir(), "model.gguf"), "--host", "localhost", "--port", strconv.Itoa(index.port), "--embedding"}
	args = append(args, "--batch-size", strconv.Itoa(index.maxChunkSize))
	args = append(args, "--ubatch-size", strconv.Itoa(index.maxChunkSize))
	return fork(binary, args)
}
