package fsm

import (
	"errors"

	"github.com/rs/zerolog"
)

// Builder is used to construct a new FSM instance
type Builder struct {
	globalGuards []Guard
	log          zerolog.Logger
	start        State
	states       []State
	transitions  []Transition
	stater       Stater
}

// NewBuilder creates a new FSM Builder.
func NewBuilder() (*Builder, error) {
	return &Builder{}, nil
}

// SetGlobalGuards sets the list of guards that are run on every transition.
func (b *Builder) SetGlobalGuards(g []Guard) {
	b.globalGuards = g
}

// SetLog configures the builder and FSM to use the supplied logger
func (b *Builder) SetLog(log zerolog.Logger) {
	b.log = log
}

// SetInitialState sets the initial state of the FSM when it is created.
func (b *Builder) SetInitialState(s State) {
	b.start = s
}

// AddTransition adds a transition to the builder
func (b *Builder) AddTransition(trans Transition) error {
	b.transitions = append(b.transitions, trans)
	return nil
}

// SetTransitions replaces all existing transitions with the supplied arguments.
func (b *Builder) SetTransitions(transitions []Transition) {
	b.transitions = transitions
}

// Build constructs a new FSM
func (b *Builder) Build(stater Stater) (*FSM, error) {
	if stater == nil {
		return nil, errors.New("failed to pass stater type into build")
	}

	transMap := make(_TransitionMap, len(b.transitions))

	for _, t := range b.transitions {
		k := FromToTuple{
			From: t.From,
			To:   t.To,
		}
		transMap[k] = _Transition{
			guards:         append([]Guard{}, t.Guards...),
			onEnterActions: append([]EnterHandler{}, t.OnEnterToActions...),
			onExitActions:  append([]ExitHandler{}, t.OnExitToActions...),
		}
	}

	m := &FSM{
		currentState: stater.GetState(),
		globalGuards: append([]Guard{}, b.globalGuards...),
		log:          b.log,
		states:       b.states,
		transitions:  transMap,
		stater:       stater,
	}

	return m, nil
}
