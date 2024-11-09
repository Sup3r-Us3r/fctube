package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/Sup3r-Us3r/fctube/internal/converter"
	"github.com/Sup3r-Us3r/fctube/internal/rabbitmq"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

func connectPostgres() (*sql.DB, error) {
	user := getEnvOrDefault("POSTGRES_USER", "root")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "root")
	dbName := getEnvOrDefault("POSTGRES_DB", "converter")
	host := getEnvOrDefault("POSTGRES_HOST", "postgres")
	sslMode := getEnvOrDefault("POSTGRES_SSLMODE", "disable")

	connectionString := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s sslmode=%s",
		user, password, dbName, host, sslMode,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		slog.Error("error connecting to database", slog.String("connectionString", connectionString))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		slog.Error("error pinging database", slog.String("connectionString", connectionString))
		return nil, err
	}

	slog.Info("connected to postgres successfully")

	return db, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func main() {
	db, err := connectPostgres()
	if err != nil {
		panic(err)
	}

	rabbitmqUrl := getEnvOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	rabbitClient, err := rabbitmq.NewRabbitClient(rabbitmqUrl)
	if err != nil {
		panic(err)
	}
	defer rabbitClient.Close()

	conversionExchange := getEnvOrDefault("CONVERSION_EXCHANGE", "conversion_exchange")
	conversionKey := getEnvOrDefault("CONVERSION_KEY", "conversion")
	queueName := getEnvOrDefault("CONVERSION_QUEUE", "video_conversion_queue")
	confirmationKey := getEnvOrDefault("CONFIRMATION_KEY", "finish-conversion")
	confirmationQueue := getEnvOrDefault("CONFIRMATION_QUEUE", "video_confirmation_queue")

	videoConverter := converter.NewVideoConverter(db, rabbitClient)

	messages, err := rabbitClient.ConsumeMessage(conversionExchange, conversionKey, queueName)
	if err != nil {
		slog.Error("failed to consume messages", slog.String("error", err.Error()))
	}

	slog.Info("starting message consumption in rabbitmq")

	for message := range messages {
		go func(delivery amqp.Delivery) {
			videoConverter.Handle(delivery, conversionExchange, confirmationKey, confirmationQueue)
		}(message)
	}
}
