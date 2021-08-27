package report

import (
	"strings"

	"ddosify.com/hammer/core/types"
)

type ReportService interface {
	DoneChan() <-chan struct{}
	init()
	Start(input chan *types.Response)
	Report()
}

func NewReportService(s string) (service ReportService, err error) {
	if strings.EqualFold(s, types.OutputTypeStdout) {
		service = &stdout{}
	} else if strings.EqualFold(s, types.OutputTypeTimescale) {
		service = &timescale{}
	}
	service.init()
	return
}
