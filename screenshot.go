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
	// Held back from the timeout so that a page which never goes idle still
	// produces an image instead of a deadline error.
	captureReserve = 3 * time.Second

	// A page can go idle and navigate afterwards, so being idle is not enough
	// on its own. The window that covers this is derived from the timeout
	// rather than fixed, because a fixed one would have to be retuned in the
	// source for every site that outlasts it.
	graceDivisor = 10
)

type Size struct {
	Width  int
	Height int
}

type Options struct {
	Timeout   time.Duration
	Viewport  Size
	Resize    Size
	NoSandbox bool
}

// Capture reports whether the page settled within the timeout. An unsettled
// page still yields an image, taken at the moment the budget ran out.
func Capture(url string, opts Options) ([]byte, bool, error) {
	execOpts := chromedp.DefaultExecAllocatorOptions[:]
	if opts.NoSandbox {
		execOpts = append(execOpts, chromedp.NoSandbox)
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), execOpts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	watcher := watchLifecycle(ctx)

	settled := false
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(opts.Viewport.Width), int64(opts.Viewport.Height)),
		page.SetLifecycleEventsEnabled(true),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			deadline, _ := ctx.Deadline()
			settled = watcher.wait(time.Until(deadline)-captureReserve, opts.Timeout/graceDivisor)
			return nil
		}),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return nil, false, err
	}

	if opts.Resize == (Size{}) {
		return buf, settled, nil
	}

	out, err := resizeImage(buf, opts.Resize)
	if err != nil {
		return nil, false, err
	}

	return out, settled, nil
}

// lifecycleWatcher follows the page across client side redirects. Navigate
// returns on the load event of the first document, which for a site behind an
// interstitial is the interstitial and not the page worth capturing.
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
// starting another document, and reports whether that happened before budget
// ran out. The bound is a timer rather than a context deadline, because
// expiring a context derived from the chromedp one takes the target down and
// the capture with it.
func (w *lifecycleWatcher) wait(budget, grace time.Duration) bool {
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
				return false
			}
		}

		remaining := grace - time.Since(idleAt)
		if remaining <= 0 {
			return true
		}

		settle := time.NewTimer(remaining)
		select {
		case <-w.changed:
		case <-settle.C:
		case <-expired.C:
			settle.Stop()
			return false
		}
		settle.Stop()
	}
}

func resizeImage(data []byte, size Size) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, size.Width, size.Height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
