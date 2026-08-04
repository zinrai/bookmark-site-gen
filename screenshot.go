package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"golang.org/x/image/draw"
)

const (
	viewportWidth  = 1280
	viewportHeight = 800
	outputWidth    = 400
	outputHeight   = 250

	// Held back from the timeout so that a page which never goes idle still
	// produces an image instead of a deadline error.
	captureReserve = 3 * time.Second

	// A page can go idle and navigate afterwards, so being idle is not enough
	// on its own. The window that covers this is derived from the timeout
	// rather than fixed, because a fixed one would have to be retuned in the
	// source for every site that outlasts it.
	graceDivisor = 10
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

	watcher := watchLifecycle(ctx)

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(viewportWidth, viewportHeight),
		page.SetLifecycleEventsEnabled(true),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			deadline, _ := ctx.Deadline()
			watcher.wait(time.Until(deadline)-captureReserve, timeout/graceDivisor)
			return nil
		}),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return nil, err
	}

	return resizeImage(buf)
}

// lifecycleWatcher follows the page across client side redirects. Navigate
// returns on the load event of the first document, which for a site behind an
// interstitial is the interstitial and not the page worth a thumbnail.
type lifecycleWatcher struct {
	mu      sync.Mutex
	idle    bool
	idleAt  time.Time
	changed chan struct{}
}

func watchLifecycle(ctx context.Context) *lifecycleWatcher {
	w := &lifecycleWatcher{changed: make(chan struct{}, 1)}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		e, ok := ev.(*page.EventLifecycleEvent)
		if !ok {
			return
		}

		w.mu.Lock()
		switch e.Name {
		case "init":
			w.idle = false
		case "networkIdle":
			w.idle = true
			w.idleAt = time.Now()
		}
		w.mu.Unlock()

		select {
		case w.changed <- struct{}{}:
		default:
		}
	})

	return w
}

// wait blocks until the page has stayed network idle for grace without
// starting another document, or until budget runs out. The bound is a timer
// rather than a context deadline, because expiring a context derived from the
// chromedp one takes the target down and the capture with it.
func (w *lifecycleWatcher) wait(budget, grace time.Duration) {
	expired := time.NewTimer(budget)
	defer expired.Stop()

	for {
		w.mu.Lock()
		idle, idleAt := w.idle, w.idleAt
		w.mu.Unlock()

		if !idle {
			select {
			case <-w.changed:
				continue
			case <-expired.C:
				return
			}
		}

		remaining := grace - time.Since(idleAt)
		if remaining <= 0 {
			return
		}

		settle := time.NewTimer(remaining)
		select {
		case <-w.changed:
		case <-settle.C:
		case <-expired.C:
			settle.Stop()
			return
		}
		settle.Stop()
	}
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
