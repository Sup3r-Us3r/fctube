package converter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/Sup3r-Us3r/fctube/internal/rabbitmq"
	"github.com/streadway/amqp"
)

type VideoConverter struct {
	db             *sql.DB
	rabbitmqClient *rabbitmq.RabbitClient
}

// {"video_id": 1, "path": "/media/uploads/1"}
type VideoTask struct {
	VideoID int    `json:"video_id"`
	Path    string `json:"path"`
}

func NewVideoConverter(db *sql.DB, rabbitmqClient *rabbitmq.RabbitClient) *VideoConverter {
	return &VideoConverter{
		db:             db,
		rabbitmqClient: rabbitmqClient,
	}
}

func (vc *VideoConverter) Handle(delivery amqp.Delivery, conversionExchange, confirmationKey, confirmationQueue string) {
	var task VideoTask

	err := json.Unmarshal(delivery.Body, &task)
	if err != nil {
		vc.logError(task, "failed to unmarshal task", err)
	}

	if IsProcessed(vc.db, task.VideoID) {
		slog.Warn("video already processed", slog.Int("video_id", task.VideoID))
		delivery.Ack(false)
		return
	}

	err = vc.processVideo(&task)
	if err != nil {
		vc.logError(task, "failed to process video", err)
		return
	}

	err = MarkProcessed(vc.db, task.VideoID)
	if err != nil {
		vc.logError(task, "failed to mark video as processed", err)
		return
	}
	slog.Info("video marked as processed", slog.Int("video_id", task.VideoID))

	confirmationMessage := []byte(fmt.Sprintf(
		`{"video_id": %d, "path": "%s"}`, task.VideoID, task.Path,
	))
	err = vc.rabbitmqClient.PublishMessage(conversionExchange, confirmationKey, confirmationQueue, confirmationMessage)
	if err != nil {
		vc.logError(task, "failed to publish confirmation message", err)
	}

	delivery.Ack(false)
	slog.Info("video finished and published", slog.Int("video_id", task.VideoID))
}

func (vc *VideoConverter) processVideo(task *VideoTask) error {
	mergedFile := filepath.Join(task.Path, "merged.mp4")
	mpegDashPath := filepath.Join(task.Path, "mpeg-dash")

	slog.Info("merging chunks", slog.String("path", task.Path))
	err := vc.mergeChunks(task.Path, mergedFile)
	if err != nil {
		vc.logError(*task, "failed to merge chunks", err)
		return err
	}

	slog.Info("creating mpeg-dash directory", slog.String("path", task.Path))
	err = os.MkdirAll(mpegDashPath, os.ModePerm)
	if err != nil {
		vc.logError(*task, "failed to create mpeg-dash directory", err)
		return err
	}

	slog.Info("converting video to mpeg-dash", slog.String("path", task.Path))
	numberOfThreads := runtime.NumCPU()
	ffmpegCmd := exec.Command(
		"ffmpeg", "-i", mergedFile,
		"-c:v", "copy", "-c:a", "copy",
		"-movflags", "+faststart",
		"-threads", fmt.Sprint(numberOfThreads),
		"-f", "dash",
		filepath.Join(mpegDashPath, "output.mpd"),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		vc.logError(*task, "failed to convert video mpeg-dash, output: "+string(output), err)
		return err
	}

	slog.Info("removing merged file", slog.String("path", mergedFile))
	err = os.Remove(mergedFile)
	if err != nil {
		vc.logError(*task, "failed to remove merged file", err)
		return err
	}

	return nil
}

func (vc *VideoConverter) extractNumber(fileName string) int {
	regex := regexp.MustCompile(`\d+`)
	numberToString := regex.FindString(filepath.Base(fileName))
	number, err := strconv.Atoi(numberToString)
	if err != nil {
		return -1
	}

	return number
}

func (vc *VideoConverter) mergeChunks(inputDir string, outputFile string) error {
	chunks, err := filepath.Glob(filepath.Join(inputDir, "*.chunk"))
	if err != nil {
		return fmt.Errorf("failed to find chunks: %v", err)
	}

	sort.Slice(chunks, func(i, j int) bool {
		return vc.extractNumber(chunks[i]) < vc.extractNumber(chunks[j])
	})

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer output.Close()

	for _, chunk := range chunks {
		input, err := os.Open(chunk)
		if err != nil {
			return fmt.Errorf("failed to open chunk: %v", err)
		}

		_, err = output.ReadFrom(input)
		if err != nil {
			return fmt.Errorf("failed to write chunk %s to merged file: %v", chunk, err)
		}

		input.Close()
	}

	return nil
}

func (vc *VideoConverter) logError(task VideoTask, message string, err error) {
	errorData := map[string]any{
		"video_id": task.VideoID,
		"error":    message,
		"details":  err.Error(),
		"time":     time.Now(),
	}
	serializedError, _ := json.Marshal(errorData)
	slog.Error("processing error", slog.String("error_details", string(serializedError)))

	RegisterError(vc.db, errorData, err)
}
