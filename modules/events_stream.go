package modules

import (
	"fmt"
	"strconv"

	"github.com/bettercap/bettercap/core"
	"github.com/bettercap/bettercap/log"
	"github.com/bettercap/bettercap/session"
)

type EventsStream struct {
	session.SessionModule
	ignoreList *IgnoreList
	quit       chan bool
}

func NewEventsStream(s *session.Session) *EventsStream {
	stream := &EventsStream{
		SessionModule: session.NewSessionModule("events.stream", s),
		quit:          make(chan bool),
		ignoreList:    NewIgnoreList(),
	}

	stream.AddHandler(session.NewModuleHandler("events.stream on", "",
		"Start events stream.",
		func(args []string) error {
			return stream.Start()
		}))

	stream.AddHandler(session.NewModuleHandler("events.stream off", "",
		"Stop events stream.",
		func(args []string) error {
			return stream.Stop()
		}))

	stream.AddHandler(session.NewModuleHandler("events.show LIMIT?", "events.show(\\s\\d+)?",
		"Show events stream.",
		func(args []string) error {
			limit := -1
			if len(args) == 1 {
				arg := core.Trim(args[0])
				limit, _ = strconv.Atoi(arg)
			}
			return stream.Show(limit)
		}))

	stream.AddHandler(session.NewModuleHandler("events.ignore FILTER", "events.ignore ([^\\s]+)",
		"Events with an identifier matching this filter will not be shown (use multiple times to add more filters).",
		func(args []string) error {
			return stream.ignoreList.Add(args[0])
		}))

	stream.AddHandler(session.NewModuleHandler("events.include FILTER", "events.include ([^\\s]+)",
		"Used to remove filters passed with the events.ignore command.",
		func(args []string) error {
			return stream.ignoreList.Remove(args[0])
		}))

	stream.AddHandler(session.NewModuleHandler("events.filters", "",
		"Print the list of filters used to ignore events.",
		func(args []string) error {
			if stream.ignoreList.Empty() {
				fmt.Printf("Ignore filters list is empty.\n")
			} else {
				stream.ignoreList.RLock()
				defer stream.ignoreList.RUnlock()

				for _, filter := range stream.ignoreList.Filters() {
					fmt.Printf("  '%s'\n", string(filter))
				}
			}
			return nil
		}))

	stream.AddHandler(session.NewModuleHandler("events.clear", "",
		"Clear events stream.",
		func(args []string) error {
			stream.Session.Events.Clear()
			return nil
		}))

	return stream
}

func (s EventsStream) Name() string {
	return "events.stream"
}

func (s EventsStream) Description() string {
	return "Print events as a continuous stream."
}

func (s EventsStream) Author() string {
	return "Simone Margaritelli <evilsocket@protonmail.com>"
}

func (s *EventsStream) Configure() error {
	return nil
}

func (s *EventsStream) Start() error {
	return s.SetRunning(true, func() {
		for {
			var e session.Event
			select {
			case e = <-s.Session.Events.NewEvents:
				if s.ignoreList.Ignored(e) == false {
					s.View(e, true)
				} else {
					log.Debug("Skipping ignored event %v", e)
				}
				break

			case <-s.quit:
				return
			}
		}
	})
}

func (s *EventsStream) Show(limit int) error {
	events := s.Session.Events.Sorted()
	num := len(events)
	from := 0

	if limit > 0 && num > limit {
		from = num - limit
	}

	for _, e := range events[from:num] {
		s.View(e, false)
	}

	s.Session.Refresh()

	return nil
}

func (s *EventsStream) Stop() error {
	return s.SetRunning(false, func() {
		s.quit <- true
	})
}
