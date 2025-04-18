package main

import (
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/model"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sync"

	_ "github.com/lib/pq" // postgresql driver support
)

const (
	USER_DATA_FILE        = "chat_users.json"
	PARTICIPANT_DATA_FILE = "participants.json"
	CONV_DATA_FILE        = "conversations.json"
	MSG_DATA_FILE         = "messages.json"
)

func runSetup(client *sql.DB) error {
	fmt.Println("Setting up...")
	err := cleanDb(client)
	if err != nil {
		return err
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	userDataFilePath := path.Join(projectDir, "tests", "data", USER_DATA_FILE)
	userData, err := os.ReadFile(userDataFilePath)
	if err != nil {
		return err
	}
	var users []model.ChatUser
	err = json.Unmarshal(userData, &users)
	if err != nil {
		fmt.Println(err)
		return err
	}

	convDataFilePath := path.Join(projectDir, "tests", "data", CONV_DATA_FILE)
	convData, err := os.ReadFile(convDataFilePath)
	if err != nil {
		return err
	}
	var conversations []model.Conversation
	err = json.Unmarshal(convData, &conversations)
	if err != nil {
		fmt.Println(err)
		return err
	}

	participantDataFilePath := path.Join(projectDir, "tests", "data", PARTICIPANT_DATA_FILE)
	participantData, err := os.ReadFile(participantDataFilePath)
	if err != nil {
		return err
	}
	var participants []model.Participant
	err = json.Unmarshal(participantData, &participants)
	if err != nil {
		fmt.Println(err)
		return err
	}

	msgDataFilePath := path.Join(projectDir, "tests", "data", MSG_DATA_FILE)
	msgData, err := os.ReadFile(msgDataFilePath)
	if err != nil {
		return err
	}
	var messages []model.Message
	err = json.Unmarshal(msgData, &messages)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tx, err := client.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	fmt.Println("Creating temporary users")
	for _, user := range users {
		// password is not hashed but that is not important for this test
		_, err = tx.Exec(`INSERT INTO chat_user.chat_user (
            id, email, username, password, role, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6
        );`, user.Id, user.Email, user.Username, user.Password, user.Role, user.CreatedAt)
		if err != nil {
			return err
		}
	}

	fmt.Println("Creating temporary conversations")
	for _, conv := range conversations {
		_, err = tx.Exec(`INSERT INTO conversation.conversation (
            id, type, creator_id, created_at
        ) VALUES (
            $1, $2, $3, $4
        );`, conv.Id, conv.Type, conv.CreatorId, conv.CreatedAt)
		if err != nil {
			return err
		}
	}

	fmt.Println("Creating temporary participants")
	for _, p := range participants {
		_, err = tx.Exec(`INSERT INTO conversation.participant (
            id, name, joined_at, user_id, conversation_id
        ) VALUES (
            $1, $2, $3, $4, $5
        );`, p.Id, p.Name, p.JoinedAt, p.UserId, p.ConversationId)
		if err != nil {
			return err
		}
	}

	fmt.Println("Creating temporary messages")
	for _, msg := range messages {
		_, err = tx.Exec(`INSERT INTO message.message (
            id, content, type, sender_id, receiver_id, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6
        );`, msg.Id, msg.Content, msg.Type, msg.SenderId, msg.ReceiverId, msg.CreatedAt)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

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
	adminDbURL, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		fmt.Println("Missing ADMIN_DATABASE_URL")
		os.Exit(1)
	}

	var err error
	dbLog, has := os.LookupEnv("DB_LOG")
	if has {
		defer func() {
			err = saveLog("chat-app-db-service", dbLog)
			if err != nil {
				fmt.Printf("saveLog(): %v\n", err)
			}
		}()
	}

	snowflakeLog, has := os.LookupEnv("SNOWFLAKE_SERVICE_LOG")
	if has {
		defer func() {
			err = saveLog("chat-app-snowflake-service", snowflakeLog)
			if err != nil {
				fmt.Printf("saveLog(): %v\n", err)
			}
		}()
	}

	client, err := sql.Open("postgres", adminDbURL)
	if err != nil {
		fmt.Println("Error opening database connection:", err)
		os.Exit(1)
	}
	defer func() {
		fmt.Println("Closing database connection")
		client.Close()
	}()

	err = runSetup(client)
	if err != nil {
		fmt.Println("Error running setup:", err)
		os.Exit(1)
	}
	// fmt.Println("Press any key to continue")
	// input := bufio.NewScanner(os.Stdin)
	// input.Scan()

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

func cleanDb(client *sql.DB) error {
	_, err := client.Exec(`DELETE FROM chat_user.chat_user;`) // this will also delete conversations, participants, and messages (cascade)
	if err != nil {
		return err
	}

	return nil
}

type LogCapture struct {
	cmd  *exec.Cmd
	file *os.File
	done chan struct{}
	wg   sync.WaitGroup
	err  error
}

func StartLogCapture(serviceName string, logFilePath string) (*LogCapture, error) {
	// Open file for writing logs
	fmt.Printf("Capturing %s and save in %s\n", serviceName, logFilePath)
	file, err := os.Create(logFilePath)
	if err != nil {
		return nil, err
	}

	// Start docker logs command with --follow
	cmd := exec.Command("docker", "logs", serviceName, "--follow")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		file.Close()
		return nil, err
	}

	lc := &LogCapture{
		cmd:  cmd,
		file: file,
		done: make(chan struct{}),
	}

	// Start copying logs to file in a goroutine
	lc.wg.Add(1)
	go func() {
		defer lc.wg.Done()
		_, lc.err = io.Copy(file, stdout)
	}()

	// Start the command
	if err := cmd.Start(); err != nil {
		file.Close()
		return nil, err
	}

	return lc, nil
}

func (lc *LogCapture) Stop() error {
	// Signal the command to stop
	lc.cmd.Process.Signal(os.Interrupt)
	close(lc.done)

	// Wait for the copy goroutine to finish
	lc.wg.Wait()

	// Close the file
	fileErr := lc.file.Close()

	// Wait for the command to exit and get any error
	cmdErr := lc.cmd.Wait()

	if lc.err != nil {
		fmt.Printf("lc")
		return lc.err
	}
	if fileErr != nil {
		fmt.Printf("file")
		return fileErr
	}
	return cmdErr
}
