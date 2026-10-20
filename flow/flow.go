package flow

import (
	"errors"
	"fmt"

	"slotmath/spin"
)

var ErrRoundCompleted = errors.New("round is already completed")

// spin 流程管理
type Flow struct {
	engine *spin.Engine
}

type State struct {
	Bet                int64
	Mode               spin.Mode
	FreeSpinsRemaining int
	Completed          bool
}

type StepResult struct {
	State State
	Spin  spin.SpinResult
}

func New(engine *spin.Engine) *Flow { return &Flow{engine: engine} }

func (f *Flow) Next(state State) (StepResult, error) {
	if state.Completed {
		return StepResult{}, ErrRoundCompleted
	}

	mode := state.Mode
	if mode == "" {
		mode = spin.ModeBase
	}

	switch mode {
	case spin.ModeBase:
		if state.FreeSpinsRemaining != 0 {
			return StepResult{}, errors.New("base mode requires zero remaining free spins")
		}
	case spin.ModeFree:
		if state.FreeSpinsRemaining <= 0 {
			return StepResult{}, errors.New("free mode requires remaining free spins")
		}
	default:
		return StepResult{}, fmt.Errorf("unsupported game mode %q", mode)
	}

	spinResult, err := f.engine.Spin(spin.SpinRequest{Bet: state.Bet, Mode: mode})
	if err != nil {
		return StepResult{}, err
	}

	remaining := state.FreeSpinsRemaining
	if mode == spin.ModeFree {
		remaining--
	}
	remaining += spinResult.FreeSpins

	nextMode := spin.ModeBase
	completed := remaining == 0
	if remaining > 0 {
		nextMode = spin.ModeFree
	}

	return StepResult{State: State{
		Bet: state.Bet, Mode: nextMode, FreeSpinsRemaining: remaining, Completed: completed,
	}, Spin: spinResult}, nil
}

func (f *Flow) Engine() *spin.Engine { return f.engine }
