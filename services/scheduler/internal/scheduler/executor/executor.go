package executor

import (
	"github.com/datacommand2/cdm-cloud/common/database/model"
)

// Executor 인터페이스
// executor job 생성, 삭제, 실행 함수로 구성
type Executor interface {
	GetJob(*model.Schedule) Job
	CreateJob(*model.Schedule) error
	DeleteJob(*model.Schedule)
	CalculateJobNextRunTime(*model.Schedule) (*int64, error)
	Close()
}

// Job 인터페이스
// schedule 실행 시 수행 될 함수로 구성
type Job interface {
	Run()
}
