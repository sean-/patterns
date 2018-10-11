package fsm

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

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

type FSM struct {
	currentState  State
	onExitActions []ExitHandler
	lock          sync.Mutex
	globalGuards  []Guard
	log           zerolog.Logger
	states        []State
	transitions   _TransitionMap
}

type State interface {
	ID() int
	MarshalZerologObject(e *zerolog.Event)
	Name() string
}

type Transition struct {
	From             State
	To               State
	Guards           []Guard
	OnEnterToActions []EnterHandler
	OnExitToActions  []ExitHandler
}

type EnterHandler func()
type ExitHandler func()

type Guard func(currState, newState State) error

func (m *FSM) States() []State {
	return m.states
}

func (m *FSM) Transitions() []FromToTuple {
	ret := make([]FromToTuple, len(m.transitions))
	var i int
	for x := range m.transitions {
		ret[i] = x.Copy()
		i++
	}

	return ret
}

func (m *FSM) CurrentState() State {
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
		err := guard(m.currentState, s)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %s to %s in global guard %d", m.currentState, s, i)
		}
	}

	// 3. Run per-transition guards
	for i, guard := range trans.guards {
		err := guard(m.currentState, s)
		if err != nil {
			return errors.Wrapf(err, "unable to transition from %s to %s in transition guard %d", m.currentState, s, i)
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

func (ftt FromToTuple) Copy() FromToTuple {
	return FromToTuple{
		From: ftt.From,
		To:   ftt.To,
	}
}
