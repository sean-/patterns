package fsm

import "github.com/rs/zerolog"

type Builder struct {
	globalGuards []Guard
	log          zerolog.Logger
	start        State
	states       []State
	transitions  []Transition
}

func NewBuilder() (*Builder, error) {
	return &Builder{}, nil
}

// SetGlobalGuard sets the list of guards that are run on every transition.
func (b *Builder) SetGlobalGuards(g []Guard) {
	b.globalGuards = g
}

func (b *Builder) SetLog(log zerolog.Logger) {
	b.log = log
}

// SetInitialState sets the initial state of the FSM when it is created.
func (b *Builder) SetInitialState(s State) {
	b.start = s
}

func (b *Builder) AddTransition(trans Transition) error {
	b.transitions = append(b.transitions, trans)
	return nil
}

func (b *Builder) SetTransitions(transitions []Transition) {
	b.transitions = transitions
}

func (b *Builder) Build() (*FSM, error) {
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
		currentState: b.start,
		globalGuards: append([]Guard{}, b.globalGuards...),
		log:          b.log,
		states:       b.states,
		transitions:  transMap,
	}

	return m, nil
}
