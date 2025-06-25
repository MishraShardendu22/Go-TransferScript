package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// Config holds the configuration loaded from config.json
type Config struct {
	OriginalUser string   `json:"originalUser"`
	NewUser      string   `json:"newUser"`
	Repositories []string `json:"repositories"`
}

// Initialize global logger
func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel) // Default level, can be configured via env or flag later
}

// loadConfig reads the configuration from config.json
func loadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data from %s: %w", filePath, err)
	}

	if config.OriginalUser == "" || config.NewUser == "" {
		return nil, errors.New("originalUser and newUser must be specified in config")
	}
	if len(config.Repositories) == 0 {
		return nil, errors.New("repositories list cannot be empty in config")
	}
	log.Infof("Configuration loaded successfully from %s. OriginalUser: %s, NewUser: %s, Repositories: %d",
		filePath, config.OriginalUser, config.NewUser, len(config.Repositories))
	return &config, nil
}

// transferRepo attempts to transfer a GitHub repository and returns an error if the transfer fails.
// repoName is used for logging purposes.
func transferRepo(client *resty.Client, transferURL string, newUser string, repoName string, githubToken string) error {
	log.WithFields(log.Fields{
		"repo":    repoName,
		"newUser": newUser,
		"url":     transferURL,
	}).Info("Attempting to transfer repository")

	res, err := client.R().
		SetBody(map[string]string{
			"new_owner": newUser,
			"new_name":  repoName,
		}).
		SetHeader("Accept", "application/vnd.github+json").
		SetHeader("Authorization", "Bearer "+githubToken).
		SetHeader("X-GitHub-Api-Version", "2022-11-28").
		Post(transferURL)

	if err != nil {
		// This error is from the client (e.g., network issue)
		return fmt.Errorf("client request for repo %s failed: %w", repoName, err)
	}

	logFields := log.Fields{
		"repo":       repoName,
		"statusCode": res.StatusCode(),
		"status":     res.Status(),
	}
	if res.StatusCode() != 202 { // Log body for non-successful transfers for debugging
		logFields["responseBody"] = res.String()
	}


	switch res.StatusCode() {
	case 202: // Accepted
		log.WithFields(logFields).Info("Repository transfer successful")
		return nil
	case 401: // Unauthorized
		log.WithFields(logFields).Error("Repository transfer failed: Unauthorized")
		return fmt.Errorf("repo %s: Unauthorized (HTTP %d). Check GitHub token and permissions. Response: %s", repoName, res.StatusCode(), res.String())
	case 403: // Forbidden
		log.WithFields(logFields).Error("Repository transfer failed: Forbidden")
		return fmt.Errorf("repo %s: Forbidden (HTTP %d). API rate limits or insufficient permissions. Response: %s", repoName, res.StatusCode(), res.String())
	case 404: // Not Found
		log.WithFields(logFields).Error("Repository transfer failed: Not Found")
		return fmt.Errorf("repo %s: Repository or user not found (HTTP %d). Response: %s", repoName, res.StatusCode(), res.String())
	case 422: // Unprocessable Entity
		log.WithFields(logFields).Error("Repository transfer failed: Unprocessable Entity")
		return fmt.Errorf("repo %s: Unprocessable Entity (HTTP %d). Semantic errors. Response: %s", repoName, res.StatusCode(), res.String())
	default:
		log.WithFields(logFields).Error("Repository transfer failed: Unexpected status code")
		return fmt.Errorf("repo %s: Unexpected status code (HTTP %d). Response: %s", repoName, res.StatusCode(), res.String())
	}
}

func main() {
	log.Info("Initializing GitHub Repository Transfer Script")

	// Load .env file for GitHub token
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Fatal error: Failed to load .env file: %v", err)
	}
	githubToken := os.Getenv("GITHUB_TOKEN_CLASSIC")
	if githubToken == "" {
		log.Fatal("Fatal error: GITHUB_TOKEN_CLASSIC environment variable is not set.")
	}
	log.Info("GITHUB_TOKEN_CLASSIC loaded successfully.")

	// Load configuration from config.json
	appConfig, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Fatal error: Failed to load configuration from config.json: %v", err)
	}

	originalUser := appConfig.OriginalUser
	newUser := appConfig.NewUser
	allRepos := appConfig.Repositories

	if len(allRepos) == 0 {
		log.Info("No repositories listed in config.json. Exiting application.")
		return
	}
	log.Infof("Processing %d repositories for transfer from %s to %s.", len(allRepos), originalUser, newUser)

	client := resty.New().
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		SetCloseConnection(true)

	var wg sync.WaitGroup
	numWorkers := 5 // This could be made configurable

	jobs := make(chan string, len(allRepos))
	results := make(chan string, len(allRepos))

	log.Infof("Starting %d worker goroutines.", numWorkers)
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int, oUser, nUser string) {
			defer wg.Done()
			workerLog := log.WithFields(log.Fields{"workerID": workerID})
			workerLog.Infof("Worker started. Transferring from %s to %s.", oUser, nUser)

			for repoName := range jobs {
				repoLog := workerLog.WithField("repo", repoName)
				repoLog.Info("Processing repository transfer")

				transferAPIURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/transfer", oUser, repoName)
				err := transferRepo(client, transferAPIURL, nUser, repoName, githubToken)
				if err != nil {
					repoLog.WithError(err).Error("Repository transfer attempt failed")
					results <- fmt.Sprintf("[FAIL] Worker %d, Repo %s: %v", workerID, repoName, err)
				} else {
					repoLog.Info("Repository transfer attempt successful")
					results <- fmt.Sprintf("[SUCCESS] Worker %d, Repo %s: Transfer successful", workerID, repoName)
				}
			}
			workerLog.Info("Worker finished.")
		}(w, originalUser, newUser)
	}

	log.Infof("Distributing %d repository transfer jobs to workers.", len(allRepos))
	for _, repo := range allRepos {
		jobs <- repo
	}
	close(jobs)
	log.Info("All jobs dispatched. Waiting for workers to complete.")

	wg.Wait()
	close(results)
	log.Info("All workers have completed.")

	log.Info("--- Final Transfer Summary ---")
	successCount := 0
	failureCount := 0
	for resMsg := range results {
		// Log the raw message from channel for now, could be more structured
		log.Debugf("Raw result message: %s", resMsg)
		if _, parseErr := fmt.Sscanf(resMsg, "[SUCCESS]%s", new(string)); parseErr == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	log.Infof("Script execution finished. Total Repositories: %d, Successes: %d, Failures: %d",
		len(allRepos), successCount, failureCount)
}
