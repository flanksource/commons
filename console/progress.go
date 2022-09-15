package console

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/timer"
)

type terminalProgress struct {
	bar    *mpb.Bar
	test   *TestResults
	status string
}
type textProgress struct {
	timer timer.Timer
	test  *TestResults
	logger.Logger
}

func (p *textProgress) Start() {
	p.Debugf("starting")
	p.timer = timer.NewTimer()

}
func (p *textProgress) Done() {
	p.Infof("finished in %s with %s", p.timer, p.test)
	p.timer.Stop()
}
func (p *textProgress) Status(status string) {
	p.Infof("%s", status)
}

func (p *terminalProgress) Start() {
	p.bar.SetCurrent(1)
}
func (p *terminalProgress) Done() {
	p.bar.SetCurrent(100)
}
func (p *terminalProgress) Status(status string) {
	p.status = status
}

func NewTextProgress(name string, localTest *TestResults) Progress {
	return &textProgress{
		Logger: logger.WithValues("test", name),
		test:   localTest,
	}
}
func NewTerminalProgress(name string, localTest *TestResults, progress *mpb.Progress) Progress {
	tp := &terminalProgress{test: localTest}
	filler := mpb.NewSpinnerFiller(mpb.DefaultSpinnerStyle, mpb.SpinnerOnLeft)
	completedFn := mpb.BarFillerFunc(func(w io.Writer, width int, st decor.Statistics) {
		if st.Completed {
			io.WriteString(w, strings.ReplaceAll(tp.test.String(), "\n", ""))
		} else {
			io.WriteString(w, tp.status)
			filler.Fill(w, 27, st)
		}
	})

	bar := progress.Add(int64(100), completedFn, mpb.PrependDecorators(
		decor.NewElapsed(decor.ET_STYLE_MMSS, time.Now()),
		decor.Name(fmt.Sprintf(" %-20s", name), decor.WC{W: len(name) + 1}),
	))
	tp.bar = bar
	return tp
}

type Progress interface {
	Done()
	Status(status string)
	Start()
}
