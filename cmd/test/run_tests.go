package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"

	_ "github.com/lib/pq" // postgresql driver support
)

func runTearDown(client *sql.DB) error {
	fmt.Println("Tearing down...")
	err := cleanDb(client)
	if err != nil {
		return err
	}

	return nil
}

func runTests() error {
	isJson := flag.Bool("json", false, "Output test results in json format")
	flag.Parse()

	if *isJson {
		fmt.Println("Running tests with json format")
		cmd := exec.Command("go", "test", "-json", "-v", "./...")
		outputFile, has := os.LookupEnv("TEST_OUT")
		if !has {
			outputFile = "test_output.json"
			fmt.Printf("Warning: `TEST_OUT` is not set, output json to default file: %s\n", outputFile)
		}

		var err error
		teeCmd := exec.Command("tee", outputFile)
		teeCmd.Stdin, err = cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("Error creating pipe: %v\n", err)
		}
		teeCmd.Stdout = os.Stdout // Output to terminal as well

		err = teeCmd.Start()
		if err != nil {
			return fmt.Errorf("Error starting tee: %v\n", err)
		}

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("Error running go test: %v\n", err)
		}

		err = teeCmd.Wait()
		if err != nil {
			return fmt.Errorf("Error waiting for tee command: %v\n", err)
		}
	} else {
		fmt.Println("Running tests")
		cmd := exec.Command("go", "test", "-v", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Error running go test: %v\n", err)
		}
	}

	return nil
}

func main() {
	adminDbUrl, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		fmt.Println("Missing ADMIN_DATABASE_URL")
		os.Exit(1)
	}

	client, err := sql.Open("postgres", adminDbUrl)
	if err != nil {
		fmt.Println("Error opening database connection:", err)
		os.Exit(1)
	}

	dbLog, has := os.LookupEnv("DB_LOG")
	if has {
		defer func() {
			err = saveLog("chat-app-db-service", dbLog)
			if err != nil {
				fmt.Printf("saveLog(): %v\n", err)
			}
		}()
	}

	snowflakeTLSLog, has := os.LookupEnv("SNOWFLAKE_TLS_SERVICE_LOG")
	if has {
		defer func() {
			err = saveLog("chat-app-snowflake-tls-service", snowflakeTLSLog)
			if err != nil {
				fmt.Printf("saveLog(): %v\n", err)
			}
		}()
	}

	snowflakeNonTLSLog, has := os.LookupEnv("SNOWFLAKE_NON_TLS_SERVICE_LOG")
	if has {
		defer func() {
			err = saveLog("chat-app-snowflake-non-tls-service", snowflakeNonTLSLog)
			if err != nil {
				fmt.Printf("saveLog(): %v\n", err)
			}
		}()
	}

	err = runTests()
	if err != nil {
		fmt.Println("Error running tests:", err)
		os.Exit(1)
	}

	err = runTearDown(client)
	if err != nil {
		fmt.Println("Error running teardown:", err)
		os.Exit(1)
	}
}

func cleanDb(client *sql.DB) error {
	tx, err := client.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`DELETE FROM chat_user.chat_user;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM conversation.conversation;`)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func saveLog(serviceName string, logFilePath string) error {
	file, err := os.Create(logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command("docker", "logs", serviceName)
	cmd.Stdout = file
	cmd.Stderr = file
	return cmd.Run()
}
