package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Define flags
	dir := flag.String("dir", "./", "The directory to change to")
	cmd := flag.String("cmd", "", "The command to run")
	only := flag.String("only", ".*", "Regex to match file paths")
	// Parse flags
	flag.Parse()

	// Validate required flags
	if *dir == "" || *cmd == "" {
		fmt.Println("Both -directory and -cmd flags are required.")
		flag.Usage()
		os.Exit(1)
	}
	absDir, err := getAbsolutePath(*dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	regex, err := regexp.Compile(*only)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Create a channel to listen for interrupt or terminate signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to stop the goroutine
	doneChan := make(chan bool)

	ticker := time.NewTicker(1 * time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var cmdWaitGroup sync.WaitGroup
		var lastCommand *exec.Cmd

		startCommand := func() {
			if lastCommand != nil {
				// If there is a previous command, ensure it has been cleaned up
				cmdWaitGroup.Wait()
			}

			ctx, cancel = context.WithCancel(context.Background())
			cmdWaitGroup.Add(1)
			go func() {
				defer cmdWaitGroup.Done()
				defer cancel()
				lastCommand, err = runCmdWithContext(ctx, *cmd)
				if err != nil {
					fmt.Println(err)
					return
				}
			}()
		}

		// Start the initial command
		startCommand()
		// pause for a few seconds in case new files are edited at start of runup
		time.Sleep(3 * time.Second)
		lastModTime := time.Now()
		for {
			select {
			case <-ticker.C:
				ifChanged, err := areFiledUpdated(lastModTime, absDir, regex)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if !ifChanged {
					continue
				}

				// Cancel the previous command if it was running
				cancel()

				// Start the new command
				startCommand()
				// Update the last modification time
				// and avoid the recursive command creates files loop
				time.Sleep(3 * time.Second)
				lastModTime = time.Now()

			case <-doneChan:
				cancel()
				cmdWaitGroup.Wait()
				return
			}
		}
	}() 
	// Wait for a signal
	sig := <-sigChan
	fmt.Printf("Received signal: %s. Exiting...\n", sig)

	// Signal the goroutine to stop
	doneChan <- true
	fmt.Println("Trying to clean up...")
	wg.Wait()
	fmt.Println("Program terminated.")
}

func runCmdWithContext(ctx context.Context, cmd string) (*exec.Cmd, error) {
	command := exec.CommandContext(ctx, "sh", "-c", cmd)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Create a new process group
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	fmt.Printf("Trying... %s\n", cmd)
	if err := command.Start(); err != nil {
		fmt.Printf("Failed to start command %s: %v\n", cmd, err)
		return nil, err
	}

	cmdDone := make(chan error)

	go func() {
		cmdDone <- command.Wait()
	}()

	select {
	case <-ctx.Done():
		// Cancel the process and its subprocesses
		// -pid means send to a whole group
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil {
			fmt.Printf("Failed to kill process group %d: %v\n", command.Process.Pid, err)
			return nil, err
		}
		fmt.Printf("Process group %d killed\n", command.Process.Pid)
		return command, nil
	case err := <-cmdDone:
		if err != nil {
			fmt.Printf("Command failed: %v\n", err)
			return nil, err
		}
		fmt.Printf("Command ran successfully %s\n", cmd)
		return command, nil
	}
}

// sends files change signal to a channel
// return true when files update
func areFiledUpdated(lastModTime time.Time, dir string, regex *regexp.Regexp) (bool, error) {
	isModded := false
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		// Match the file path against the regex
		if !regex.MatchString(path) {
			return nil // Skip files that don't match the regex
		}
		fStat, err := os.Stat(path)
		if err != nil {
			fmt.Println(err)
			return err
		}
		modTime := fStat.ModTime()
		if modTime.After(lastModTime) {
			fmt.Println(path, " Was saved, running command")
			isModded = true
			return nil
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return isModded, nil
}

func getAbsolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		// If it's already an absolute path, return it
		return path, nil
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Combine the current working directory with the relative path
	absPath, err := filepath.Abs(filepath.Join(cwd, path))
	if err != nil {
		return "", err
	}

	return absPath, nil
}
