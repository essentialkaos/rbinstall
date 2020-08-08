package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2020 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"pkg.re/essentialkaos/ek.v12/fmtc"
	"pkg.re/essentialkaos/ek.v12/options"
	"pkg.re/essentialkaos/ek.v12/timeutil"
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

var spinnerFrames = []string{"⠸", "⠴", "⠤", "⠦", "⠇", "⠋", "⠉", "⠙"}
var framesDelay = []time.Duration{75, 55, 35, 55, 75, 75, 75, 75}

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

	time.Sleep(25 * time.Millisecond)

	t.hideSpinner(err == nil)

	return result, err
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (t *Task) showSpinner() {
SPINNERLOOP:
	for {
		for i, frame := range spinnerFrames {
			fmtc.Printf("{y}%s  {!}%s… ", frame, t.Desc)
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
		fmtc.Printf("{g}✔  {!}%s {s-}(%s){!}\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	} else {
		fmtc.Printf("{r}✖  {!}%s {s-}(%s){!}\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	}
}
