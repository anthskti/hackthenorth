package main

import (
	"context"
	"fmt"
)

const mouseChunk = 127

func mouseMoveCommands(dx, dy int) []string {
	var cmds []string
	for dx != 0 || dy != 0 {
		cx := clampMouse(dx)
		cy := clampMouse(dy)
		if cx == 0 && cy == 0 {
			if dx != 0 {
				cx = sign(dx) * mouseChunk
			} else {
				cy = sign(dy) * mouseChunk
			}
		}
		cmds = append(cmds, fmt.Sprintf("m%d,%d", cx, cy))
		dx -= cx
		dy -= cy
	}
	return cmds
}

func clampMouse(v int) int {
	if v > mouseChunk {
		return mouseChunk
	}
	if v < -mouseChunk {
		return -mouseChunk
	}
	return v
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	return 1
}

func mouseButtonClickCmd(button string) string {
	switch button {
	case "right":
		return "bRc"
	case "middle":
		return "bMc"
	default:
		return "bLc"
	}
}

func (p *PiClient) SendMouseMoveTo(ctx context.Context, state *manualPointerState, x, y int) error {
	if state == nil {
		return nil
	}
	dx := x - state.x
	dy := y - state.y
	if dx == 0 && dy == 0 {
		return nil
	}
	cmds := mouseMoveCommands(dx, dy)
	if len(cmds) == 0 {
		return nil
	}
	if err := p.SendCommands(ctx, cmds...); err != nil {
		return err
	}
	state.x = x
	state.y = y
	return nil
}

func (p *PiClient) SendMouseClick(ctx context.Context, state *manualPointerState, x, y int, button string) error {
	var cmds []string
	if state != nil {
		dx := x - state.x
		dy := y - state.y
		cmds = append(cmds, mouseMoveCommands(dx, dy)...)
		state.x = x
		state.y = y
	}
	cmds = append(cmds, mouseButtonClickCmd(button))
	return p.SendCommands(ctx, cmds...)
}

func (p *PiClient) SendMouseWheel(ctx context.Context, delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 9999 {
		delta = 9999
	}
	if delta < -9999 {
		delta = -9999
	}
	return p.SendCommands(ctx, fmt.Sprintf("w%d", delta))
}

type manualPointerState struct {
	x int
	y int
}
