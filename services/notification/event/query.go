package event

import (
	"github.com/datacommand2/cdm-cloud/common/database/model"
)

// EventsQuery events 질의
type EventsQuery struct {
	limit     uint64
	offset    uint64
	event     model.Event
	eventCode model.EventCode
	solution  string
	from, to  int64
	Error     error
}

// NewEventsQuery events 질의 생성
func NewEventsQuery(tenantID uint64) *EventsQuery {
	return &EventsQuery{
		event: model.Event{TenantID: tenantID},
	}
}

// Limit 질의 갯수 제한
func (q *EventsQuery) Limit(limit uint64) *EventsQuery {
	if limit < 10 {
		q.limit = 10
	} else if limit > 100 {
		q.limit = 100
	} else {
		q.limit = limit
	}
	return q
}

// Offset row 시작 인덱스
func (q *EventsQuery) Offset(offset uint64) *EventsQuery {
	q.offset = offset
	return q
}

// From 시작 시간
func (q *EventsQuery) From(from int64) *EventsQuery {
	q.from = from
	return q
}

// To 끝 시간
func (q *EventsQuery) To(to int64) *EventsQuery {
	q.to = to
	return q
}

// Solutions 솔루션들
// - solutions == nil: 모든 솔루션
func (q *EventsQuery) Solutions(solution string) *EventsQuery {
	q.solution = solution
	return q
}

// Level 경고 수준
func (q *EventsQuery) Level(level string) *EventsQuery {
	q.eventCode.Level = level
	return q
}

// Class1 대분류
func (q *EventsQuery) Class1(class1 string) *EventsQuery {
	q.eventCode.Class1 = class1
	return q
}

// Class2 중분류
func (q *EventsQuery) Class2(class2 string) *EventsQuery {
	q.eventCode.Class2 = class2
	return q
}

// Class3 소분류
func (q *EventsQuery) Class3(class3 string) *EventsQuery {
	q.eventCode.Class3 = class3
	return q
}
