package flow

import (
	"errors"

	"slotmath/spin"
)

var ErrRoundCompleted = errors.New("round is already completed")

type State struct {
	Bet                int64
	Mode               spin.Mode
	FreeSpinsRemaining int
	Completed          bool
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
	mode := state.Mode
	if mode == "" {
		mode = spin.ModeBase
	}
	spinResult, err := f.engine.Spin(spin.Request{Bet: state.Bet, Mode: mode})
	if err != nil {
		return Result{}, err
	}
	remaining := state.FreeSpinsRemaining
	if mode == spin.ModeFree {
		if remaining <= 0 {
			return Result{}, errors.New("free spin mode requires remaining free spins")
		}
		remaining--
	}
	remaining += spinResult.FreeSpins
	nextMode := spin.ModeBase
	completed := true
	if remaining > 0 {
		nextMode = spin.ModeFree
		completed = false
	}
	return Result{State: State{
		Bet: state.Bet, Mode: nextMode, FreeSpinsRemaining: remaining, Completed: completed,
	}, Spin: spinResult}, nil
}

func (f *Flow) Engine() *spin.Engine { return f.engine }
