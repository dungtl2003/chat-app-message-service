package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq" // postgresql driver support
)

type LogFile struct {
	ServiceName string `json:"service_name"`
	LogFilePath string `json:"log_file_path"`
}

func main() {
	adminDatabaseURL, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		fmt.Println("Missing ADMIN_DATABASE_URL")
		os.Exit(1)
	}

	client, err := sql.Open("postgres", adminDatabaseURL)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	logFiles := createLogFiles()

	if err := runTests(); err != nil {
		fmt.Printf("Error running tests: %v\n", err)
		if err := runTearDown(client); err != nil {
			fmt.Printf("Error during tear down: %v\n", err)
		}
		exit(1, logFiles)
	}
	fmt.Println("Tests completed successfully")

	if err := runTearDown(client); err != nil {
		fmt.Printf("Error during tear down: %v\n", err)
		exit(1, logFiles)
	}
	fmt.Println("Tear down completed successfully")

	exit(0, logFiles)
}

func createLogFiles() []LogFile {
	logFiles := []LogFile{}

	raw := os.Getenv("LOG_META")
	if raw == "" {
		fmt.Println("LOG_META not set")
		return logFiles
	}

	logToService := make(map[string]string)

	for entry := range strings.SplitSeq(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid entry: %s\n", entry)
			continue
		}
		logPath := strings.TrimSpace(parts[0])
		serviceName := strings.TrimSpace(parts[1])
		logToService[logPath] = serviceName
	}

	for logPath, serviceName := range logToService {
		logFiles = append(logFiles, LogFile{
			ServiceName: serviceName,
			LogFilePath: logPath,
		})
	}

	return logFiles
}

func exit(code int, logFiles []LogFile) {
	for _, logFile := range logFiles {
		fmt.Printf("Log file for service %s: %s\n", logFile.ServiceName, logFile.LogFilePath)
		if err := saveLog(logFile); err != nil {
			fmt.Printf("Error saving log for service %s: %v\n", logFile.ServiceName, err)
		} else {
			fmt.Printf("Log for service %s saved successfully\n", logFile.ServiceName)
		}
	}

	if code != 0 {
		fmt.Printf("Exiting with code %d\n", code)
	}
	os.Exit(code)
}

func runTests() error {
	var cmd *exec.Cmd
	isJson := flag.Bool("json", false, "Output test results in json format")
	flag.Parse()

	if *isJson {
		fmt.Println("Running tests with json format")
		cmd = exec.Command("go", "test", "-json", "-v", "./...")

	} else {
		fmt.Println("Running tests")
		cmd = exec.Command("go", "test", "-v", "./...")
	}

	outputFile, has := os.LookupEnv("TEST_OUT")
	if !has {
		outputFile = "test_output"
		fmt.Printf("Warning: `TEST_OUT` is not set, output to default file: %s\n", outputFile)
	}

	dir := filepath.Dir(outputFile)
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("Error creating directory for output file: %v\n", err)
	}
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("Error creating file: %v\n", err)
	}
	defer file.Close()

	// Write output to both stdout and file
	mw := io.MultiWriter(os.Stdout, file)
	cmd.Stdout = mw
	cmd.Stderr = os.Stderr // Optional: pipe stderr to terminal

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error running go test: %v\n", err)
	}

	return nil
}

func saveLog(logFile LogFile) error {
	serviceName := logFile.ServiceName
	logFilePath := logFile.LogFilePath

	dir := filepath.Dir(logFilePath)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command("docker", "logs", serviceName)
	cmd.Stdout = file
	cmd.Stderr = file
	fmt.Printf("Running command: %s\n", cmd.String())
	return cmd.Run()
}

func runTearDown(client *sql.DB) error {
	fmt.Println("Tearing down...")
	err := cleanDb(client)
	if err != nil {
		return err
	}

	return nil
}

func cleanDb(client *sql.DB) error {
	tx, err := client.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	queries := []string{
		"DELETE FROM chat_user.chat_user;",
		"DELETE FROM media.asset;",
		"DELETE FROM conversation.conversation;",
	}

	for _, query := range queries {
		_, err = tx.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
