package report

import (
	"strings"
	"sync"

	"ddosify.com/hammer/core/types"
)

type ReportService interface {
	DoneChan() <-chan struct{}
	init()
	Start(input chan *types.Response)
	Report()
}

var once sync.Once
var service ReportService

func CreateReportService(s string) (ReportService, error) {
	if service == nil {
		once.Do(
			func() {
				if strings.EqualFold(s, types.OutputTypeStdout) {
					service = &stdout{}
				} else if strings.EqualFold(s, types.OutputTypeTimescale) {
					service = &timescale{}
				}
				service.init()
			},
		)
	}
	return service, nil
}
