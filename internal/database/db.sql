CREATE TABLE IF NOT EXISTS processed_videos (
  video_id INT PRIMARY KEY,
  status VARCHAR(50) NOT NULL,
  processed_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS process_errors_log (
  id SERIAL PRIMARY KEY,
  error_details JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL
);
