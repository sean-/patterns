package fsm

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// FromToTuple is used to construct a transition lookup key
type FromToTuple struct {
	From State
	To   State
}

type _Transition struct {
	guards         []Guard
	onEnterActions []EnterHandler
	onExitActions  []ExitHandler
}

type _TransitionMap map[FromToTuple]_Transition

// FSM is a Finite State Machine implementation.  An FSM is created via a
// Builder.
type FSM struct {
	currentState  State
	onExitActions []ExitHandler
	lock          sync.Mutex
	globalGuards  []Guard
	log           zerolog.Logger
	states        []State
	transitions   _TransitionMap
	stater        Stater
}

// State interface specifies the required interface for a State in the FSM.
type State interface {
	// ID is a unique numeric ID representing a State
	ID() int
	MarshalZerologObject(e *zerolog.Event)
	// Name is the human-friendly name of the State
	Name() string
}

// Stater defines an object that persists the state machine's resulting state.
type Stater interface {
	SetState(State) error
	GetState() State
}

// Transition is a user-specified Transition.  From is the old state.  To is the
// new State.  Guards are a collection of zero or more callbacks that are
// executed to determine if a transition may occur.  If a Guard returns an
// error, the transition will fail.  Once all GlobalGuards and
// Transition-specific Guards complete successfully, any OnEnterToActions will
// be executed when the To state is being entered (error handling must be
// handled by the caller and any error conditions should be detected in a
// Guard).  Any OnExitToActions will be called when the FSM transitions to a
// different state (again, error handling must be handled in a Guard or by the
// handler itself).
type Transition struct {
	From             State
	To               State
	Guards           []Guard
	OnEnterToActions []EnterHandler
	OnExitToActions  []ExitHandler
}

// EnterHandler is the On-Enter transition handler
type EnterHandler func(stater Stater)

// ExitHandler is the On-Exit transition handler
type ExitHandler func(stater Stater)

// Guard is the transition guard handler signature
type Guard func(currState, newState State, stater Stater) error

// States returns a list of known states
func (m *FSM) States() []State {
	return m.states
}

// Transitions returns a list of transitions in the form of a slice of
// FromToTuples.
func (m *FSM) Transitions() []FromToTuple {
	ret := make([]FromToTuple, len(m.transitions))
	var i int
	for x := range m.transitions {
		ret[i] = x.Copy()
		i++
	}

	return ret
}

// CurrentState returns the current state of the FSM.  This method will deadlock
// if called within a Transition Handler.
func (m *FSM) CurrentState() State {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.currentState
}

// SetStater sets the Stater object that this FSM can manipulate when
// transitioning between states.
func (m *FSM) SetStater(stater Stater) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.stater = stater
}

// Transition transitions the state of the FSM from one state to another.
// Before transitioning, the FSM looks up to ensure that a given transition
// exists.  If the transition exists, all registered guards are executed
// sequentially and must succeed.  If a guard fails, the error from the guard is
// returned from Transition.  Once all guards have run successfully, any exit
// handlers are executed.  Once the exit handlers have finished, the state is
// changed.  If there are any on-enter hooks, the entry hooks are executed
// before the Transition returns.
func (m *FSM) Transition(s State) error {
	if m.stater != nil {
		if ss := m.stater.GetState(); ss != m.currentState {
			return errors.Errorf("unable to transition stater without matching %v to %v",
				ss, m.currentState)
		}
	}

	transKey := FromToTuple{From: m.currentState, To: s}

	// For now, wrap the entire function in a single mutex.  Reentrant transition
	// changes are not supported at this time.  I'm not sure how that would be
	// modeled.
	m.lock.Lock()
	defer m.lock.Unlock()

	// 1. Ensure the transition exists
	trans, found := m.transitions[transKey]
	if !found {
		return errors.Errorf("unable to transition from %v to %v", m.currentState, s)
	}

	m.log.Debug().Object("from", m.currentState).Object("to", s).Msg("transitioning")

	// 2. Run global guards
	for i, guard := range m.globalGuards {
		err := guard(m.currentState, s, m.stater)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %s to %s in global guard %d", m.currentState, s, i)
		}
	}

	// 3. Run per-transition guards
	for i, guard := range trans.guards {
		err := guard(m.currentState, s, m.stater)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %s to %s in transition guard %d", m.currentState, s, i)
		}
	}

	// 4. Run the exit handlers, if any
	for _, exitFunc := range m.onExitActions {
		exitFunc(m.stater)
	}

	// 5. Change the state
	m.currentState = s

	// 6. Update state of Stater
	if m.stater != nil {
		m.stater.SetState(s)
	}

	// 7. Swap onExit handlers to the new state
	m.onExitActions = trans.onExitActions

	// 8. Run the enter actions, if any
	for _, enterFunc := range trans.onEnterActions {
		enterFunc(m.stater)
	}

	return nil
}

// Copy creates a duplicate FromToTuple
func (ftt FromToTuple) Copy() FromToTuple {
	return FromToTuple{
		From: ftt.From,
		To:   ftt.To,
	}
}
