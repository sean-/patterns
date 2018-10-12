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

// FromIDToIDTuple is used to construct a transition lookup key
type _FromIDToIDTuple struct {
	From StateID
	To   StateID
}

type _Transition struct {
	transition     FromToTuple
	guards         []Guard
	onEnterActions []EnterHandler
	onExitActions  []ExitHandler
}

type _TransitionMap map[_FromIDToIDTuple]_Transition

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
}

// StateID is the internal descriptor used to uniquely identify a State.
type StateID int64

// State interface specifies the required interface for a State in the FSM.
type State interface {
	// ID is a unique numeric ID representing a State
	ID() StateID
	MarshalZerologObject(e *zerolog.Event)
	// String is the human-friendly name of the State
	String() string
}

// Transition is a user-specified Transition.  From is the old state.  To is the
// new State.  Guards are a collection of zero or more callbacks that are
// executed to determine if a transition may occur.  If a Guard returns an
// error, the transition will fail.  Once all GlobalGuards and
// Transition-specific Guards complete successfully, any OnExitToActions will be
// executed when the From State is being exited (error handling must be handled
// by the caller and any error conditions should be detected in a Guard).  Any
// OnEnterToActions will be called when the FSM transitions to a different state
// (again, error handling must be handled in a Guard or by the handler itself).
type Transition struct {
	From             State
	To               State
	Guards           []Guard
	OnEnterToActions []EnterHandler
	OnExitToActions  []ExitHandler
}

// EnterHandler is the On-Enter transition handler
type EnterHandler func()

// ExitHandler is the On-Exit transition handler
type ExitHandler func()

// Guard is the transition guard handler signature
type Guard func(currState, newState State) error

// States returns a list of known states
func (m *FSM) States() []State {
	return m.states
}

// Transitions returns a list of transitions in the form of a slice of
// FromToTuples.
func (m *FSM) Transitions() []FromToTuple {
	ret := make([]FromToTuple, len(m.transitions))
	var i int
	for _, v := range m.transitions {
		ret[i] = v.transition.Copy()
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

// Transition transitions the state of the FSM from one state to another.
// Before transitioning, the FSM looks up to ensure that a given transition
// exists.  If the transition exists, all registered guards are executed
// sequentially and must succeed.  If a guard fails, the error from the guard is
// returned from Transition.  Once all guards have run successfully, any exit
// handlers are executed.  Once the exit handlers have finished, the state is
// changed.  If there are any on-enter hooks, the entry hooks are executed
// before the Transition returns.
func (m *FSM) Transition(s State) error {
	transKey := _FromIDToIDTuple{From: m.currentState.ID(), To: s.ID()}

	// For now, wrap the entire function in a single mutex.  Reentrant transition
	// changes are not supported at this time.  I'm not sure how that would be
	// modeled.
	m.lock.Lock()
	defer m.lock.Unlock()

	// 1. Ensure the transition exists
	trans, found := m.transitions[transKey]
	if !found {
		return errors.Errorf("unable to transition from %q to %q", m.currentState, s)
	}

	m.log.Debug().Object("from", m.currentState).Object("to", s).Msg("transitioning")

	// 2. Run global guards
	for i, guard := range m.globalGuards {
		err := guard(m.currentState, s)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %q to %q in global guard %d", m.currentState, s, i)
		}
	}

	// 3. Run per-transition guards
	for i, guard := range trans.guards {
		err := guard(m.currentState, s)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %q to %q in transition guard %d", m.currentState, s, i)
		}
	}

	// 4. Run the exit handlers, if any
	for _, exitFunc := range m.onExitActions {
		exitFunc()
	}

	// 5. Change the state
	m.currentState = s
	m.onExitActions = trans.onExitActions

	// 6. Run the enter actions, if any
	for _, enterFunc := range trans.onEnterActions {
		enterFunc()
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
