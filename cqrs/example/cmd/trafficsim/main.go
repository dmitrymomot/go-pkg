package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dmitrymomot/go-env"
	"github.com/dmitrymomot/go-pkg/cqrs"
	"github.com/dmitrymomot/go-pkg/cqrs/example/booking"
	"github.com/dmitrymomot/go-utils"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var simulateRPS = env.GetInt("SIMULATE_RPS", int64(100))

func main() {
	logger := logrus.WithFields(logrus.Fields{
		"app":       "cqrs-traffic-simulator-" + getCurrentHostName(),
		"component": "main",
	})
	defer func() { logger.Info("Server successfully shutdown") }()

	// Redis is used as a message broker.
	redisOptions, err := redis.ParseURL("redis://redis:6379/0")
	if err != nil {
		logger.WithError(err).Fatal("Cannot parse redis URL")
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()

	// Command publisher is used to publish commands to Redis.
	commandsPublisher, err := cqrs.NewPublisher(redisClient,
		cqrs.NewLogrusWrapper(logger.WithField("component", "cqrs-commands-publisher")),
	)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create commands publisher")
	}
	defer commandsPublisher.Close()

	// init command bus
	cmdBus, err := cqrs.NewCommandBus(commandsPublisher)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create command bus")
	}

	// run command publisher to simulate incoming traffic
	publishCommands(
		cmdBus,
		simulateRPS,
		logger.WithField("component", "publishCommands"),
	)
}

// publish BookRoom commands every 1/100 second to simulate incoming traffic
func publishCommands(commandBus cqrs.CommandBus, simulateRPS int64, logger *logrus.Entry) {
	logger.Info("Publishing commands started")
	startTime := time.Now()
	defer func() {
		logger.WithField("duration", time.Since(startTime)).Info("Publishing commands finished")
	}()

	i := 0
	for {
		go func(n int) {
			startDate := time.Now().Add(time.Hour * 24 * 2)
			endDate := startDate.Add(time.Hour * 24 * 3)

			bookRoomCmd := &booking.BookRoom{
				RoomId:    fmt.Sprintf("%d", n),
				GuestName: "John",
				StartDate: utils.Pointer(startDate),
				EndDate:   utils.Pointer(endDate),
				UnixTime:  time.Now().UnixNano(),
			}
			if err := commandBus.Send(context.Background(), bookRoomCmd); err != nil {
				logger.WithError(err).Error("Cannot send BookRoom command")
			}
		}(i) // simulate some work

		time.Sleep(time.Second / time.Duration(simulateRPS)) // simulate incoming traffic
		i++
	}
}

// getCurrentLocalHostName returns current local host name.
// It is used to generate unique instance ID for each instance of cqrs-example.
// Instance ID is used to prevent multiple instances of cqrs-example to handle same commands.
// In production, you probably will use some kind of distributed lock to prevent multiple instances to handle same commands.
func getCurrentHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}
