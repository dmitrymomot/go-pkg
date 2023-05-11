package cqrs

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

// NewFacade creates a new CQRS Facade.
// Read more about cqrs component: https://watermill.io/docs/cqrs/
func NewFacade(
	redisClient redis.UniversalClient, logger Logger, router *message.Router,
	commandsPublisher, eventsPublisher message.Publisher,
	commandsSubscriber message.Subscriber,
	commandHandlers []CommanfHandlerFactory, eventHandlers []EventHandlerFactory,
) (*cqrs.Facade, error) {
	// no command handlers, no event handlers - no sense to create cqrs facade
	if len(eventHandlers) == 0 && len(commandHandlers) == 0 {
		return nil, fmt.Errorf("no command or event handlers provided")
	}

	// if logger is nil, we will use watermill's default logger
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	// We are using JSON marshaler, but you can use any marshaler you want.
	cqrsMarshaler := cqrs.JSONMarshaler{}

	// CQRS Facade configuration.
	cnf := cqrs.FacadeConfig{
		Router:                router,
		CommandEventMarshaler: cqrsMarshaler,
		Logger:                logger,
	}

	// If we don't have any command handlers, we don't need to create command buses and subscribers.
	if len(commandHandlers) > 0 {
		cnf.GenerateCommandsTopic = func(commandName string) string {
			return commandName
		}
		cnf.CommandHandlers = func(cb *cqrs.CommandBus, eb *cqrs.EventBus) []cqrs.CommandHandler {
			ch := make([]cqrs.CommandHandler, len(commandHandlers))
			for i, factory := range commandHandlers {
				ch[i] = factory(cb, eb)
			}
			return ch
		}
		cnf.CommandsPublisher = commandsPublisher
		cnf.CommandsSubscriberConstructor = func(handlerName string) (message.Subscriber, error) {
			// we can reuse subscriber, because all commands have separated topics
			return commandsSubscriber, nil
		}
	}

	// If we don't have any event handlers, we don't need to create event buses and subscribers.
	if len(eventHandlers) > 0 {
		cnf.GenerateEventsTopic = func(eventName string) string {
			return eventName
		}
		cnf.EventHandlers = func(cb *cqrs.CommandBus, eb *cqrs.EventBus) []cqrs.EventHandler {
			eh := make([]cqrs.EventHandler, len(eventHandlers))
			for i, factory := range eventHandlers {
				eh[i] = factory(cb, eb)
			}
			return eh
		}
		cnf.EventsPublisher = eventsPublisher
		cnf.EventsSubscriberConstructor = func(handlerName string) (message.Subscriber, error) {
			return NewSubscriber(redisClient, handlerName, logger)
		}
	}

	// cqrs.Facade is facade for Command and Event buses and processors.
	// You can use facade, or create buses and processors manually (you can inspire with cqrs.NewFacade)
	cqrsFacade, err := cqrs.NewFacade(cnf)
	if err != nil {
		return nil, fmt.Errorf("could not create cqrs facade: %w", err)
	}

	return cqrsFacade, nil
}

// NewCommandBus creates a new CommandBus.
// Read more about command bus: https://watermill.io/docs/cqrs/#command-bus
func NewCommandBus(publisher message.Publisher) (*cqrs.CommandBus, error) {
	return cqrs.NewCommandBus(
		publisher,
		func(commandName string) string { return commandName },
		cqrs.JSONMarshaler{},
	)
}

// NewEventBus creates a new EventBus.
// Read more about event bus: https://watermill.io/docs/cqrs/#event-bus
func NewEventBus(publisher message.Publisher) (*cqrs.EventBus, error) {
	return cqrs.NewEventBus(
		publisher,
		func(eventName string) string { return eventName },
		cqrs.JSONMarshaler{},
	)
}
