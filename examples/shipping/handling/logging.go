package handling

import (
	"time"

	"github.com/a69/kit.go/log"

	"github.com/a69/kit.go/examples/shipping/cargo"
	"github.com/a69/kit.go/examples/shipping/location"
	"github.com/a69/kit.go/examples/shipping/voyage"
)

type loggingService struct {
	logger log.Logger
	Service
}

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(logger log.Logger, s Service) Service {
	return &loggingService{logger, s}
}

func (s *loggingService) RegisterHandlingEvent(completed time.Time, id cargo.TrackingID, voyageNumber voyage.Number,
	unLocode location.UNLocode, eventType cargo.HandlingEventType) (err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "register_incident",
			"tracking_id", id,
			"location", unLocode,
			"voyage", voyageNumber,
			"event_type", eventType,
			"completion_time", completed,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.RegisterHandlingEvent(completed, id, voyageNumber, unLocode, eventType)
}
