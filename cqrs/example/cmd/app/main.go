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

var simulateTraffic = env.GetBool("SIMULATE_TRAFFIC", false)

func main() {
	logger := logrus.WithFields(logrus.Fields{
		"app":       "cqrs-example-" + getCurrentHostName(),
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

	// Command subscriber is used to consume commands from Redis.
	commandsSubscriber, err := cqrs.NewSubscriber(redisClient, "example-commands",
		cqrs.NewLogrusWrapper(logger.WithField("component", "cqrs-commands-subscriber")),
	)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create commands subscriber")
	}
	defer commandsSubscriber.Close()

	// Events will be published to PubSub configured Redis, because they may be consumed by multiple consumers.
	// (in that case BookingsFinancialReport and OrderBeerOnRoomBooked).
	eventsPublisher, err := cqrs.NewPublisher(redisClient,
		cqrs.NewLogrusWrapper(logger.WithField("component", "cqrs-events-publisher")),
	)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create events publisher")
	}
	defer eventsPublisher.Close()

	// Router is used to route commands to correct command handler.
	router, err := cqrs.NewRouter(cqrs.NewLogrusWrapper(logger.WithField("component", "cqrs-router")), 10)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create router")
	}
	defer router.Close()

	// cqrs.Facade is facade for Command and Event buses, and processors.
	// You can use facade, or create buses and processors manually (you can inspire with cqrs.NewFacade)
	cqrsFacade, err := cqrs.NewFacade(
		redisClient,
		cqrs.NewLogrusWrapper(logger.WithField("component", "cqrs-facade")),
		router,
		commandsPublisher, eventsPublisher, commandsSubscriber,
		[]cqrs.CommanfHandlerFactory{
			booking.NewBookRoomHandler(),
			booking.NewOrderBeerHandler(),
		}, []cqrs.EventHandlerFactory{
			booking.NewBookingsFinancialReport(),
			booking.NewOrderBeerOnRoomBooked(),
		},
	)
	if err != nil {
		logger.WithError(err).Fatal("Cannot create cqrs facade")
	}

	if simulateTraffic {
		// publish BookRoom commands every second to simulate incoming traffic
		go func() {
			publishCommands(
				cqrsFacade.CommandBus(),
				logger.WithField("component", "publishCommands"),
			)
		}()
	}

	// processors are based on router, so they will work when router will start
	if err := router.Run(context.Background()); err != nil {
		logger.WithError(err).Fatal("Cannot run router")
	}
}

// publish BookRoom commands every 1/100 second to simulate incoming traffic
func publishCommands(commandBus cqrs.CommandBus, logger *logrus.Entry) {
	logger.Info("Publishing commands started")
	startTime := time.Now()
	defer func() {
		logger.WithField("duration", time.Since(startTime)).Info("Publishing commands finished")
	}()

	for i := 1; i < 10000; i++ {
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

		time.Sleep(time.Second / 100) // simulate incoming traffic
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
