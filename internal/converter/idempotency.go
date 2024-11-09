package converter

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
)

func IsProcessed(db *sql.DB, videoID int) bool {
	var isProcessed bool

	query := "SELECT EXISTS(SELECT 1 FROM processed_videos WHERE video_id = $1 AND status = 'success')"
	err := db.QueryRow(query, videoID).Scan(&isProcessed)
	if err != nil {
		slog.Error("error checking if video is proceed", slog.Int("video_id", videoID), slog.String("error", err.Error()))
		return false
	}

	return isProcessed
}

func MarkProcessed(db *sql.DB, videoID int) error {
	query := "INSERT INTO processed_videos (video_id, status, processed_at) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, videoID, "success", time.Now())
	if err != nil {
		slog.Error("error marking video as processed", slog.Int("video_id", videoID), slog.String("error", err.Error()))
		return err
	}

	return nil
}

func RegisterError(db *sql.DB, errorData map[string]any, err error) {
	serializedError, _ := json.Marshal(errorData)

	query := "INSERT INTO process_errors_log (error_details, created_at) VALUES ($1, $2)"
	_, dbErr := db.Exec(query, serializedError, time.Now())
	if dbErr != nil {
		slog.Error(
			"error registering error",
			slog.String("original_error", err.Error()),
			slog.String("error_details", string(serializedError)),
		)
	}
}
