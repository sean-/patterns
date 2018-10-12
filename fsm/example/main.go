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

type _State int

const (
	stateInitializing _State = iota
	stateRunning
	stateStopped
)

var states map[_State]string

func init() {
	states = map[_State]string{
		stateInitializing: "initializing",
		stateRunning:      "running",
		stateStopped:      "stopped",
	}
}

func (s _State) ID() int {
	return int(s)
}

func (s _State) Name() string {
	if str, found := states[s]; found {
		return str
	}

	panic(fmt.Sprintf("unknown state: %v", s))
	return "unknown"
}

func (s _State) MarshalZerologObject(e *zerolog.Event) {
	e.Str("state", s.Name())
}

func (s _State) String() string {
	return s.Name()
}

type _Stater struct {
	state fsm.State
}

func (s *_Stater) SetState(ss fsm.State) error {
	s.state = ss
	return nil
}

func (s *_Stater) GetState() fsm.State {
	return s.state
}

func (s *_Stater) MarshalZerologObject(e *zerolog.Event) {
	// just delegate down to the state marshaler
	s.GetState().MarshalZerologObject(e)
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
		func(old, new fsm.State, stater fsm.Stater) error {
			st := stater.(*_Stater)
			log.Debug().Object("old", old).Object("new", new).Object("stater", st).Msg("global guard fired")
			return nil
		},
	})
	fb.SetLog(log)

	_ = fb.AddTransition(fsm.Transition{
		From: stateInitializing,
		To:   stateRunning,
		Guards: []fsm.Guard{
			func(old, new fsm.State, stater fsm.Stater) error {
				st := stater.(*_Stater)
				log.Debug().Object("old", old).Object("new", new).Object("stater", st).Msg("transition guard fired")
				return nil
			},
		},
		OnEnterToActions: []fsm.EnterHandler{
			func(stater fsm.Stater) {
				st := stater.(*_Stater)
				log.Debug().Object("stater", st).Msg("entering running state")
			},
		},
		OnExitToActions: []fsm.ExitHandler{
			func(stater fsm.Stater) {
				st := stater.(*_Stater)
				log.Debug().Object("stater", st).Msg("exiting running state")
			},
		},
	})
	_ = fb.AddTransition(fsm.Transition{
		From: stateRunning,
		To:   stateStopped,
		OnEnterToActions: []fsm.EnterHandler{
			func(stater fsm.Stater) {
				st := stater.(*_Stater)
				log.Debug().Object("stater", st).Msg("entering stopped state")
			},
		},
	})

	var stater = &_Stater{}

	if err := stater.SetState(stateInitializing); err != nil {
		log.Error().Err(err).Msg("unable to build FSM")
		return sysexits.Software
	}

	m, err := fb.Build(stater)
	if err != nil {
		log.Error().Err(err).Msg("unable to build FSM")
		return sysexits.Software
	}

	for _, state := range m.States() {
		log.Debug().Object("state", state).Msg("")
	}

	log.Debug().Object("current state", m.CurrentState()).Msg("")

	log.Debug().Object("stater", stater).Msg("")

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

	log.Debug().Object("stater", stater).Msg("")

	log.Debug().Msg("done")

	return sysexits.OK
}
