package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type LogFile struct {
	ServiceName string `json:"service_name"`
	LogFilePath string `json:"log_file_path"`
}

// 1. Use a run() function to ensure defers execute before os.Exit
func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	// Parse flags early so we can pass the rest to go test
	isJson := flag.Bool("json", false, "Output test results in json format")
	flag.Parse()

	// Remaining args (e.g., -run TestName) are passed to the test command
	passthroughArgs := flag.Args()

	adminDatabaseURL, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		fmt.Println("Error: Missing ADMIN_DATABASE_URL")
		return 1
	}

	client, err := sql.Open("postgres", adminDatabaseURL)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		return 1
	}
	// This defer will now actually run!
	defer client.Close()

	logFiles := createLogFiles()

	// Ensure logs are saved even if runTests panics or fails
	defer saveLogs(logFiles)

	// Ensure DB is cleaned up at the end
	defer func() {
		if err := runTearDown(client); err != nil {
			fmt.Printf("Error during tear down: %v\n", err)
		}
	}()

	if err := runTests(*isJson, passthroughArgs); err != nil {
		fmt.Printf("Tests failed: %v\n", err)
		return 1
	}

	fmt.Println("Tests completed successfully")
	return 0
}

func createLogFiles() []LogFile {
	logFiles := []LogFile{}
	raw := os.Getenv("LOG_META")
	if raw == "" {
		return logFiles
	}

	// Splitting by ; then =
	entries := strings.SplitSeq(raw, ";")
	for entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			logFiles = append(logFiles, LogFile{
				LogFilePath: strings.TrimSpace(parts[0]),
				ServiceName: strings.TrimSpace(parts[1]),
			})
		}
	}
	return logFiles
}

func saveLogs(logFiles []LogFile) {
	for _, logFile := range logFiles {
		if err := saveLog(logFile); err != nil {
			fmt.Printf("Error saving log for %s: %v\n", logFile.ServiceName, err)
		}
	}
}

func runTests(isJson bool, extraArgs []string) error {
	args := []string{"test", "-timeout", "0", "-v"}

	if isJson {
		args = append(args, "-json")
	}

	// Add user provided args (like -run or -count)
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	// Always append the package specifier last
	args = append(args, "./...")

	fmt.Printf("Running: go %s\n", strings.Join(args, " "))
	cmd := exec.Command("go", args...)

	outputFile := os.Getenv("TEST_OUT")
	if outputFile == "" {
		outputFile = "test_output.json"
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("mkdir error: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create file error: %w", err)
	}
	defer file.Close()

	// Capture output
	cmd.Stdout = io.MultiWriter(os.Stdout, file)
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func saveLog(logFile LogFile) error {
	dir := filepath.Dir(logFile.LogFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(logFile.LogFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Add a timeout to docker logs so it doesn't hang forever
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "logs", logFile.ServiceName)
	cmd.Stdout = file
	cmd.Stderr = file
	return cmd.Run()
}

func runTearDown(client *sql.DB) error {
	fmt.Println("Cleaning database...")

	// 2. Use TRUNCATE CASCADE.
	// This is faster and handles Foreign Keys automatically.
	// You don't need to worry about delete order.
	query := `
        TRUNCATE TABLE 
            chat_user.chat_user, 
            media.asset, 
            conversation.conversation
        RESTART IDENTITY CASCADE;
    `

	_, err := client.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clean db: %w", err)
	}

	return nil
}
