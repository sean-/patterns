package fsm

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Builder is used to construct a new FSM instance
type Builder struct {
	globalGuards []Guard
	log          zerolog.Logger
	start        State
	transitions  []Transition
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
func (b *Builder) Build() (*FSM, error) {
	transMap := make(_TransitionMap, len(b.transitions))

	statesIDMap := make(map[StateID]State, 2*len(b.transitions))
	statesNameMap := make(map[string]State, 2*len(b.transitions))

	for _, t := range b.transitions {
		statesIDMap[t.From.ID()] = t.From
		statesIDMap[t.To.ID()] = t.To
		statesNameMap[t.From.String()] = t.From
		statesNameMap[t.To.String()] = t.To

		k := _FromIDToIDTuple{
			From: t.From.ID(),
			To:   t.To.ID(),
		}

		// Check to make sure we're not transitioning from one state to itself
		if t.From.ID() == t.To.ID() {
			return nil, errors.Errorf("unable to re-transition to the same state: from %q to %q", k.From, k.To)
		}

		// Check for duplicate IDs
		if _, found := transMap[k]; found {
			return nil, errors.Errorf("duplicate state transition found: from %q to %q", k.From, k.To)
		}

		// Check for state name reuse
		if v, found := statesNameMap[t.From.String()]; found && v.ID() != t.From.ID() {
			return nil, errors.Errorf("state name reused for a different ID: %q/%q", t.From.String(), t.From.ID())
		} else if v, found := statesNameMap[t.To.String()]; found && v.ID() != t.To.ID() {
			return nil, errors.Errorf("state name reused for a different ID: %q/%q", t.To.String(), t.To.ID())
		}

		transMap[k] = _Transition{
			guards:         append([]Guard{}, t.Guards...),
			onEnterActions: append([]EnterHandler{}, t.OnEnterToActions...),
			onExitActions:  append([]ExitHandler{}, t.OnExitToActions...),
			transition:     FromToTuple{From: t.From, To: t.To},
		}
	}

	states := make([]State, 0, len(statesIDMap))
	for _, s := range statesIDMap {
		states = append(states, s)
	}

	m := &FSM{
		currentState: b.start,
		globalGuards: append([]Guard{}, b.globalGuards...),
		log:          b.log,
		states:       states,
		transitions:  transMap,
	}

	return m, nil
}
