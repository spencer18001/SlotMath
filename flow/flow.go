package flow

import (
	"errors"

	"slotmath/spin"
)

// 通知此次 spin 已完成的錯誤
var ErrRoundCompleted = errors.New("round is already completed")

// spin 流程管理
type Flow struct {
	engine *spin.Engine // spin 引擎
}

type State struct {
	Bet                int64     // 押注
	Mode               spin.Mode // 遊戲模式
	FreeSpinsRemaining int       // 剩下幾局 free game
	Completed          bool      // 是否此次 spin 已完成
}

type Result struct {
	State State
	Spin  spin.SpinResult
}

// 生成 Flow
func New(engine *spin.Engine) *Flow { return &Flow{engine: engine} }

// 下一步
func (f *Flow) Next(state State) (Result, error) {
	if state.Completed {
		return Result{}, ErrRoundCompleted
	}

	mode := state.Mode
	if mode == "" {
		mode = spin.ModeBase
	}

	spinResult, err := f.engine.Spin(spin.SpinRequest{Bet: state.Bet, Mode: mode})
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
