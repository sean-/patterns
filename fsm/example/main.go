package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/fsm"
	"github.com/sean-/seed"
	"github.com/sean-/sysexits"
)

func main() {
	os.Exit(realMain())
}

type State int

const (
	stateInitializing State = iota
	stateRunning
	stateStopped
)

var states map[State]string

func init() {
	states = map[State]string{
		stateInitializing: "initializing",
		stateRunning:      "running",
		stateStopped:      "stopped",
	}
}

type States []State

func (s State) ID() int {
	return int(s)
}

func (s State) Name() string {
	if str, found := states[s]; found {
		return str
	}

	panic(fmt.Sprintf("unknown state: %v", s))
	return "unknown"
}

func (s State) MarshalZerologObject(e *zerolog.Event) {
	e.Str("state", s.Name())
}

func realMain() int {
	seed.MustInit()

	const logTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

	zerolog.DurationFieldUnit = time.Microsecond
	zerolog.DurationFieldInteger = true
	zerolog.TimeFieldFormat = logTimeFormat

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().Timestamp().Logger()

	fb, err := fsm.NewBuilder()
	if err != nil {
		log.Error().Err(err).Msg("unable to create FSM Builder")
		return sysexits.Software
	}

	fb.SetGlobalGuards([]fsm.Guard{
		func(old, new fsm.State) error {
			log.Debug().Object("old", old).Object("new", new).Msg("global guard fired")
			return nil
		},
	})
	fb.SetLog(log)
	fb.SetStates([]fsm.State{
		stateInitializing,
	})
	fb.SetStart(stateInitializing)
	_ = fb.AddTransition(fsm.Transition{
		From: stateInitializing,
		To:   stateRunning,
		Guards: []fsm.Guard{
			func(old, new fsm.State) error {
				log.Debug().Object("old", old).Object("new", new).Msg("transition guard fired")
				return nil
			},
		},
		OnEnterToActions: []fsm.EnterHandler{
			func() {
				log.Debug().Msg("entering running state")
			},
		},
		OnExitToActions: []fsm.ExitHandler{
			func() {
				log.Debug().Msg("exiting running state")
			},
		},
	})
	_ = fb.AddTransition(fsm.Transition{
		From: stateRunning,
		To:   stateStopped,
		OnEnterToActions: []fsm.EnterHandler{
			func() {
				log.Debug().Msg("entering stopped state")
			},
		},
	})

	m, err := fb.Build()
	if err != nil {
		log.Error().Err(err).Msg("unable to build FSM")
		return sysexits.Software
	}

	for _, state := range m.States() {
		log.Debug().Object("state", state).Msg("")
	}

	log.Debug().Object("current state", m.CurrentState()).Msg("")

	for _, x := range m.Transitions() {
		log.Debug().Object("from", x.From).Object("to", x.To).Msg("")
	}

	if err := m.Transition(stateStopped); err != nil {
		log.Error().Err(err).Msg("failed to transition")
	}

	if err := m.Transition(stateRunning); err != nil {
		log.Error().Err(err).Msg("failed to transition")
	}

	if err := m.Transition(stateStopped); err != nil {
		log.Error().Err(err).Msg("failed to transition")
	}

	log.Debug().Object("current state", m.CurrentState()).Msg("")

	log.Debug().Msg("done")

	return sysexits.OK
}
