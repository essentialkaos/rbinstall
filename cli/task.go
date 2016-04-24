package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/timeutil"
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

// Start start task
func (t *Task) Start(args ...string) (string, error) {
	t.start = time.Now()

	go t.showSpinner()

	result, err := t.Handler(args...)

	t.hideSpinner(err == nil)

	return result, err
}

func (t *Task) showSpinner() {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	t.spinnerActive = true
	t.spinnerHiden = false

SPINNERLOOP:
	for {
		for _, frame := range spinner {
			fmtc.Printf("{c}%s{!} "+t.Desc, frame)
			time.Sleep(50 * time.Millisecond)
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
		fmtc.Printf("{g}✔{!} %s {s}(%s){!}\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	} else {
		fmtc.Printf("{r}✘{!} %s {s}(%s){!}\n\n", t.Desc, timeutil.PrettyDuration(time.Since(t.start)))
	}
}
