package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// CURRENT ONLY WORKS IF THE COMMAND YOU ARE RUNNING TERMINATES
// TODO is to cancel command that dont terminate and manage them somehow

func main() {
	// Define flags
	dir := flag.String("dir", "./", "The directory to change to")
	cmd := flag.String("cmd", "", "The command to run")
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

	fmt.Println("Command executed successfully.")
	// Create a channel to listen for interrupt or terminate signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to stop the goroutine
	doneChan := make(chan bool)
	// this needs to be a buffered channel to avoid deadlock (should really be a return not a channel but whatever haha)
	fileModChan := make(chan time.Time, 100)
	lastModTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	// Start a goroutine that runs indefinitely
	go func() {
		for {
			select {
			case lastModTime = <-fileModChan:
				err := runCmd(*cmd)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			case <-ticker.C:
				err := notifyChanFilesUpdated(lastModTime, absDir, fileModChan)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			case <-doneChan:
				fmt.Println("Stopping goroutine...")
				return
			}
		}
	}()
	// run the cmd the first time
	err = runCmd(*cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for a signal
	sig := <-sigChan
	fmt.Printf("Received signal: %s. Exiting...\n", sig)

	// Signal the goroutine to stop
	doneChan <- true

	fmt.Println("Program terminated.")
}

func runCmd(cmd string) error {
	command := exec.Command("sh", "-c", cmd)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fmt.Printf("Failed to execute command %s: %v\n", cmd, err)
		return err
	}
	return nil
}

// sends files change signal to a channel
// I dont need a channel but lets do it anyway
func notifyChanFilesUpdated(lastModTime time.Time, dir string, ch chan time.Time) error {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		fStat, err := os.Stat(path)
		if err != nil {
			fmt.Println(err)
			return err
		}
		modTime := fStat.ModTime()
		if modTime.After(lastModTime) {
			fmt.Println(path, " Was saved, running command")
			ch <- modTime
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
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
