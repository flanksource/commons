package console

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/timer"
)

type textProgress struct {
	timer timer.Timer
	test  *TestResults
	logger.Logger
}

func (p *textProgress) Start() {
	p.timer = timer.NewTimer()

}
func (p *textProgress) Done() {
	p.Infof("finished in %s with %s", p.timer, p.test)
	p.timer.Stop()
}
func (p *textProgress) Status(status string) {
	p.Infof("%s", status)
}

func NewTextProgress(name string, localTest *TestResults) Progress {
	return &textProgress{
		Logger: logger.WithValues("test", name),
		test:   localTest,
	}
}

type Progress interface {
	Done()
	Status(status string)
	Start()
}
