package flow

import (
	"errors"

	"slotmath/spin"
)

var ErrRoundCompleted = errors.New("round is already completed")

type State struct {
	Bet       int64
	Completed bool
}

type Result struct {
	State State
	Spin  spin.Result
}

type Flow struct{ engine *spin.Engine }

func New(engine *spin.Engine) *Flow { return &Flow{engine: engine} }

func (f *Flow) Next(state State) (Result, error) {
	if state.Completed {
		return Result{}, ErrRoundCompleted
	}
	spinResult, err := f.engine.Spin(spin.Request{Bet: state.Bet})
	if err != nil {
		return Result{}, err
	}
	return Result{State: State{Bet: state.Bet, Completed: true}, Spin: spinResult}, nil
}

func (f *Flow) Engine() *spin.Engine { return f.engine }
