package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2019 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"pkg.re/essentialkaos/ek.v10/fmtc"
	"pkg.re/essentialkaos/ek.v10/options"
	"pkg.re/essentialkaos/ek.v10/timeutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Task struct {
	Desc    string
	Handler func(args ...string) (string, error)

	start         time.Time
	spinnerActive bool
	spinnerHiden  bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

var (
	spinnerFrames = []string{"⠸", "⠴", "⠤", "⠦", "⠇", "⠋", "⠉", "⠙"}
	framesDelay   = []time.Duration{75, 55, 35, 55, 75, 75, 75, 75}
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Start start task
func (t *Task) Start(args ...string) (string, error) {
	t.start = time.Now()
	t.spinnerActive = true
	t.spinnerHiden = false

	if options.GetB(OPT_NO_PROGRESS) {
		t.spinnerHiden = true
	} else {
		go t.showSpinner()
	}

	result, err := t.Handler(args...)

	t.hideSpinner(err == nil)

	return result, err
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (t *Task) showSpinner() {
SPINNERLOOP:
	for {
		for i, frame := range spinnerFrames {
			fmtc.Printf("{y}%s {!}%s… ", frame, t.Desc)
			time.Sleep(framesDelay[i] * time.Millisecond)
			fmtc.Printf("\r")

			if !t.spinnerActive {
				t.spinnerHiden = true
				break SPINNERLOOP
			}
		}
	}
}

func (t *Task) hideSpinner(ok bool) {
	t.spinnerActive = false

	for {
		if t.spinnerHiden {
			break
		}
	}

	if ok {
		fmtc.Printf("{g}✔ {!}%s {s-}(%s){!}\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	} else {
		fmtc.Printf("{r}✖ {!}%s {s-}(%s){!}\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	}
}
