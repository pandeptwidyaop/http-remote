package services

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
)

var ErrExecutionNotFound = errors.New("execution not found")

type ExecutorService struct {
	db         *database.DB
	cfg        *config.Config
	appService *AppService
	streams    map[string][]chan string
	streamsMu  sync.RWMutex
}

func NewExecutorService(db *database.DB, cfg *config.Config, appService *AppService) *ExecutorService {
	return &ExecutorService{
		db:         db,
		cfg:        cfg,
		appService: appService,
		streams:    make(map[string][]chan string),
	}
}

func (s *ExecutorService) CreateExecution(commandID string, userID int64) (*models.Execution, error) {
	id := uuid.New().String()

	_, err := s.db.Exec(
		"INSERT INTO executions (id, command_id, user_id, status) VALUES (?, ?, ?, ?)",
		id, commandID, userID, models.StatusPending,
	)
	if err != nil {
		return nil, err
	}

	return s.GetExecutionByID(id)
}

func (s *ExecutorService) GetExecutionByID(id string) (*models.Execution, error) {
	var exec models.Execution
	var output sql.NullString
	var exitCode sql.NullInt64
	var startedAt, finishedAt sql.NullTime

	err := s.db.QueryRow(
		"SELECT id, command_id, user_id, status, output, exit_code, started_at, finished_at, created_at FROM executions WHERE id = ?",
		id,
	).Scan(&exec.ID, &exec.CommandID, &exec.UserID, &exec.Status, &output, &exitCode, &startedAt, &finishedAt, &exec.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrExecutionNotFound
	}
	if err != nil {
		return nil, err
	}

	if output.Valid {
		exec.Output = output.String
	}
	if exitCode.Valid {
		code := int(exitCode.Int64)
		exec.ExitCode = &code
	}
	if startedAt.Valid {
		exec.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		exec.FinishedAt = &finishedAt.Time
	}

	return &exec, nil
}

func (s *ExecutorService) GetExecutions(limit, offset int) ([]models.ExecutionWithDetails, error) {
	if limit == 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT e.id, e.command_id, e.user_id, e.status, e.output, e.exit_code,
		       e.started_at, e.finished_at, e.created_at,
		       c.name as command_name, a.name as app_name, COALESCE(u.username, 'API') as username
		FROM executions e
		JOIN commands c ON e.command_id = c.id
		JOIN apps a ON c.app_id = a.id
		LEFT JOIN users u ON e.user_id = u.id
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []models.ExecutionWithDetails
	for rows.Next() {
		var exec models.ExecutionWithDetails
		var output sql.NullString
		var exitCode sql.NullInt64
		var startedAt, finishedAt sql.NullTime

		if err := rows.Scan(
			&exec.ID, &exec.CommandID, &exec.UserID, &exec.Status, &output, &exitCode,
			&startedAt, &finishedAt, &exec.CreatedAt,
			&exec.CommandName, &exec.AppName, &exec.Username,
		); err != nil {
			return nil, err
		}

		if output.Valid {
			exec.Output = output.String
		}
		if exitCode.Valid {
			code := int(exitCode.Int64)
			exec.ExitCode = &code
		}
		if startedAt.Valid {
			exec.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			exec.FinishedAt = &finishedAt.Time
		}

		executions = append(executions, exec)
	}
	return executions, nil
}

func (s *ExecutorService) Execute(executionID string) error {
	log.Printf("[Executor] Starting execution %s", executionID)

	execution, err := s.GetExecutionByID(executionID)
	if err != nil {
		log.Printf("[Executor] Error getting execution %s: %v", executionID, err)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	command, err := s.appService.GetCommandByID(execution.CommandID)
	if err != nil {
		log.Printf("[Executor] Error getting command for execution %s: %v", executionID, err)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	app, err := s.appService.GetAppByID(command.AppID)
	if err != nil {
		log.Printf("[Executor] Error getting app for execution %s: %v", executionID, err)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	// Check if working directory exists
	if _, err := os.Stat(app.WorkingDir); os.IsNotExist(err) {
		errMsg := "Working directory does not exist: " + app.WorkingDir
		log.Printf("[Executor] %s (execution %s)", errMsg, executionID)
		s.finishExecution(executionID, models.StatusFailed, errMsg, -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return errors.New(errMsg)
	}

	log.Printf("[Executor] Running command '%s' in %s (execution %s)", command.Name, app.WorkingDir, executionID)

	now := time.Now()
	s.db.Exec(
		"UPDATE executions SET status = ?, started_at = ? WHERE id = ?",
		models.StatusRunning, now, executionID,
	)

	timeout := command.TimeoutSeconds
	if timeout > s.cfg.Execution.MaxTimeout {
		timeout = s.cfg.Execution.MaxTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command.Command)
	cmd.Dir = app.WorkingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[Executor] Error creating stdout pipe: %v (execution %s)", err, executionID)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("[Executor] Error creating stderr pipe: %v (execution %s)", err, executionID)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[Executor] Error starting command: %v (execution %s)", err, executionID)
		s.finishExecution(executionID, models.StatusFailed, err.Error(), -1)
		s.broadcastComplete(executionID, -1, models.StatusFailed)
		return err
	}

	var output string
	var outputMu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.streamOutput(executionID, stdout, &output, &outputMu)
	}()
	go func() {
		defer wg.Done()
		s.streamOutput(executionID, stderr, &output, &outputMu)
	}()

	err = cmd.Wait()
	wg.Wait()

	exitCode := 0
	status := models.StatusSuccess

	if err != nil {
		status = models.StatusFailed
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	outputMu.Lock()
	finalOutput := output
	outputMu.Unlock()

	s.finishExecution(executionID, status, finalOutput, exitCode)
	s.broadcastComplete(executionID, exitCode, status)

	log.Printf("[Executor] Finished execution %s with status=%s, exit_code=%d", executionID, status, exitCode)

	return nil
}

func (s *ExecutorService) streamOutput(executionID string, r io.Reader, output *string, mu *sync.Mutex) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		mu.Lock()
		*output += line + "\n"
		mu.Unlock()

		s.broadcastLine(executionID, line)
	}
}

func (s *ExecutorService) finishExecution(id string, status models.ExecutionStatus, output string, exitCode int) {
	now := time.Now()
	s.db.Exec(
		"UPDATE executions SET status = ?, output = ?, exit_code = ?, finished_at = ? WHERE id = ?",
		status, output, exitCode, now, id,
	)
}

func (s *ExecutorService) Subscribe(executionID string) chan string {
	ch := make(chan string, 100)

	s.streamsMu.Lock()
	s.streams[executionID] = append(s.streams[executionID], ch)
	s.streamsMu.Unlock()

	return ch
}

func (s *ExecutorService) Unsubscribe(executionID string, ch chan string) {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()

	channels := s.streams[executionID]
	for i, c := range channels {
		if c == ch {
			s.streams[executionID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}

	if len(s.streams[executionID]) == 0 {
		delete(s.streams, executionID)
	}
}

func (s *ExecutorService) broadcastLine(executionID string, line string) {
	s.streamsMu.RLock()
	defer s.streamsMu.RUnlock()

	for _, ch := range s.streams[executionID] {
		select {
		case ch <- "output:" + line:
		default:
		}
	}
}

func (s *ExecutorService) broadcastComplete(executionID string, exitCode int, status models.ExecutionStatus) {
	s.streamsMu.RLock()
	defer s.streamsMu.RUnlock()

	for _, ch := range s.streams[executionID] {
		select {
		case ch <- "complete:" + string(status):
		default:
		}
	}
}
