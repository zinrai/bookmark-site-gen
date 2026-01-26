package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/image/draw"
)

const (
	viewportWidth  = 1280
	viewportHeight = 800
	outputWidth    = 400
	outputHeight   = 250
)

func CaptureScreenshot(url string, timeout time.Duration, noSandbox bool) ([]byte, error) {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	if noSandbox {
		opts = append(opts, chromedp.NoSandbox)
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(viewportWidth, viewportHeight),
		chromedp.Navigate(url),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return nil, err
	}

	return resizeImage(buf)
}

func resizeImage(data []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
