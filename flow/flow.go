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

type Flow struct{ game *spin.Game }

func New(game *spin.Game) *Flow { return &Flow{game: game} }

func (f *Flow) Next(state State) (Result, error) {
	if state.Completed {
		return Result{}, ErrRoundCompleted
	}
	spinResult, err := f.game.Spin(spin.Request{Bet: state.Bet})
	if err != nil {
		return Result{}, err
	}
	return Result{State: State{Bet: state.Bet, Completed: true}, Spin: spinResult}, nil
}

func (f *Flow) Game() *spin.Game { return f.game }
