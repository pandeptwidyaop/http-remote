package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type StreamHandler struct {
	executorService *services.ExecutorService
}

func NewStreamHandler(executorService *services.ExecutorService) *StreamHandler {
	return &StreamHandler{
		executorService: executorService,
	}
}

func (h *StreamHandler) Stream(c *gin.Context) {
	id := c.Param("id")
	h.streamExecution(c, id)
}

func (h *StreamHandler) streamExecution(c *gin.Context, id string) {
	execution, err := h.executorService.GetExecutionByID(id)
	if err != nil {
		if err == services.ErrExecutionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if execution.Status == models.StatusSuccess || execution.Status == models.StatusFailed {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		if execution.Output != "" {
			lines := strings.Split(execution.Output, "\n")
			for _, line := range lines {
				if line != "" {
					_, _ = fmt.Fprintf(c.Writer, "event: output\ndata: %s\n\n", line)
				}
			}
		}

		exitCode := 0
		if execution.ExitCode != nil {
			exitCode = *execution.ExitCode
		}
		_, _ = fmt.Fprintf(c.Writer, "event: complete\ndata: {\"status\": \"%s\", \"exit_code\": %d}\n\n", execution.Status, exitCode)
		c.Writer.Flush()
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.executorService.Subscribe(id)
	defer h.executorService.Unsubscribe(id, ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}

			if strings.HasPrefix(msg, "output:") {
				line := strings.TrimPrefix(msg, "output:")
				_, _ = fmt.Fprintf(w, "event: output\ndata: %s\n\n", line)
			} else if strings.HasPrefix(msg, "complete:") {
				// Fetch final execution to get actual exit code and any remaining output
				finalExec, err := h.executorService.GetExecutionByID(id)
				exitCode := 1
				status := strings.TrimPrefix(msg, "complete:")
				if err == nil && finalExec.ExitCode != nil {
					exitCode = *finalExec.ExitCode
					status = string(finalExec.Status)
				}
				_, _ = fmt.Fprintf(w, "event: complete\ndata: {\"status\": \"%s\", \"exit_code\": %d}\n\n", status, exitCode)
				return false
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
