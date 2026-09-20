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

func (p *PiClient) SendMouseDelta(ctx context.Context, dx, dy int) error {
	cmds := mouseMoveCommands(dx, dy)
	if len(cmds) == 0 {
		return nil
	}
	return p.SendCommands(ctx, cmds...)
}

func (p *PiClient) SendMouseClick(ctx context.Context, button string) error {
	return p.SendCommands(ctx, mouseButtonClickCmd(button))
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
