package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"time"

	"github.com/gin-gonic/gin"
)

func handleVideoStream(frames *FrameBuffer) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Connection", "close")
		c.Header("Content-Type", "multipart/x-mixed-replace; boundary=frame")
		c.Header("Pragma", "no-cache")

		boundary := "frame"
		flusher, ok := c.Writer.(interface{ Flush() })
		if !ok {
			c.Status(500)
			return
		}

		for {
			if c.Request.Context().Err() != nil {
				return
			}

			frame, err := mockJPEGFrame()
			if err != nil {
				return
			}
			frames.Put(frame)

			part := fmt.Sprintf(
				"--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n",
				boundary,
				len(frame),
			)
			if _, err := c.Writer.WriteString(part); err != nil {
				return
			}
			if _, err := c.Writer.Write(frame); err != nil {
				return
			}
			if _, err := c.Writer.WriteString("\r\n"); err != nil {
				return
			}
			flusher.Flush()

			time.Sleep(200 * time.Millisecond)
		}
	}
}

func mockJPEGFrame() ([]byte, error) {
	const w, h = 640, 360
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	t := time.Now()
	bg := color.RGBA{
		R: uint8((t.Unix() / 2) % 200),
		G: 48,
		B: 64,
		A: 255,
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}
	// Simple bar that moves to show the stream is live.
	barX := int(t.UnixMilli()/16) % (w - 80)
	for y := h/2 - 20; y < h/2+20; y++ {
		for x := barX; x < barX+80; x++ {
			img.Set(x, y, color.RGBA{R: 220, G: 220, B: 220, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
