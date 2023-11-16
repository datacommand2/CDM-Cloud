package executor

import (
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	types "github.com/datacommand2/cdm-cloud/services/scheduler/constants"
	"github.com/gorhill/cronexpr"
	"github.com/robfig/cron/v3"
	"strconv"
	"sync"
	"time"
)

type cronParser struct{}

func (c *cronParser) Parse(spec string) (cron.Schedule, error) {
	return cronexpr.Parse(spec)
}

func cronMonthExpression(startMonth time.Month, interval uint) string {
	mod := uint(startMonth) % interval
	if mod == 0 {
		mod = interval
	}
	return fmt.Sprintf("%d/%d", mod, interval)
}

// CronExecutor cron expression 형식에 익스 큐터
// 월단위 스케줄 등록 삭제가 가능
type CronExecutor struct {
	*cron.Cron
	loc   *time.Location
	jobs  map[uint64]*CronJob
	lock  *sync.Mutex
	parse cron.ScheduleParser
}

// NewCronExecutor cron expression 형식에 익스 큐터 생성
func NewCronExecutor(timezone string) Executor {
	loc, _ := time.LoadLocation(timezone)
	p := new(cronParser)
	c := cron.New(cron.WithLocation(loc), cron.WithParser(p))
	c.Start()

	return &CronExecutor{
		Cron:  c,
		loc:   loc,
		lock:  &sync.Mutex{},
		jobs:  make(map[uint64]*CronJob),
		parse: p,
	}
}

func (e *CronExecutor) addJob(schedule *model.Schedule, spec string) error {
	job := &CronJob{schedule: schedule, delete: e.DeleteJob}

	id, err := e.AddJob(spec, job)
	if err != nil {
		err = errors.Unknown(err)
		return err
	}

	job.Entry = e.Entry(id)
	e.jobs[schedule.ID] = job
	return nil
}

func (e *CronExecutor) createCronExpression(schedule *model.Schedule) (string, error) {
	var (
		minute, hour uint
		spec         string
	)

	if schedule.Minute != nil {
		minute = *schedule.Minute
	}

	if schedule.Hour != nil {
		hour = *schedule.Hour
	}

	startAt := time.Unix(schedule.StartAt, 0).In(e.loc)
	switch schedule.Type {
	case types.ScheduleTypeDayOfMonthly:
		month := cronMonthExpression(startAt.Month(), *schedule.IntervalMonth)
		spec = fmt.Sprintf("0 %d %d %s %s ? *", minute, hour, *schedule.DayOfMonth, month)

	case types.ScheduleTypeWeekOfMonthly:
		month := cronMonthExpression(startAt.Month(), *schedule.IntervalMonth)
		spec = fmt.Sprintf("0 %d %d ? %s %s%s *", minute, hour, month, *schedule.DayOfWeek, *schedule.WeekOfMonth)

	default:
		return "", UnsupportedScheduleType(schedule.Type)
	}
	return spec, nil
}

// CalculateJobNextRunTime cron 스케줄 잡의 실행 시간을 구하는 함수
func (e *CronExecutor) CalculateJobNextRunTime(schedule *model.Schedule) (*int64, error) {
	var nextRunTime int64

	spec, err := e.createCronExpression(schedule)
	if err != nil {
		return nil, err
	}

	expression, err := e.parse.Parse(spec)
	if err != nil {
		return nil, err
	}

	fromTime := time.Now().In(e.loc)
	for {
		nextRunTime = expression.Next(fromTime).Unix()
		if nextRunTime >= schedule.StartAt {
			break
		}
		fromTime = time.Unix(nextRunTime, 0)
	}
	return &nextRunTime, nil
}

// CreateJob cron job 생성 및 등록
func (e *CronExecutor) CreateJob(schedule *model.Schedule) error {
	if !schedule.ActivationFlag {
		return nil
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	spec, err := e.createCronExpression(schedule)
	if err != nil {
		return err
	}
	return e.addJob(schedule, spec)
}

// DeleteJob cron 익스 큐터 job 삭제
func (e *CronExecutor) DeleteJob(schedule *model.Schedule) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if job, ok := e.jobs[schedule.ID]; ok {
		e.Remove(job.Entry.ID)
		delete(e.jobs, schedule.ID)
	}
}

// GetJob cron 익스 큐터에 등록된 스케줄 job 조회
func (e *CronExecutor) GetJob(schedule *model.Schedule) Job {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.jobs[schedule.ID]
}

// Close cron 익스 큐터 스케줄 종료
func (e *CronExecutor) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()

	ctx := e.Cron.Stop()
	<-ctx.Done()

	for k := range e.jobs {
		delete(e.jobs, k)
	}
}

// CronJob cron 익스 큐터에 스케줄 잡 구조체
type CronJob struct {
	cron.Entry
	schedule *model.Schedule
	delete   func(*model.Schedule)
}

// Run cron 익스 큐터 스케줄 시, 실제 실행 될 함수
func (j *CronJob) Run() {
	runtime := time.Now().Unix()

	if runtime >= j.schedule.EndAt {
		j.delete(j.schedule)
		return
	} else if j.Entry.Next.Unix() < j.schedule.StartAt {
		return
	}

	var msg = broker.Message{
		Header: map[string]string{"runtime": strconv.FormatInt(runtime, 10)},
		Body:   []byte(j.schedule.Message),
	}

	if err := broker.Publish(j.schedule.Topic, &msg); err != nil {
		logger.Errorf("[executor_cron-Run] Error occurred in schedule running cause: %v", err)
	} else {
		logger.Infof("[executor_cron-Run] Published schedule( %d:%s - %s ).", j.schedule.ID, j.schedule.Topic, getScheduleMessage(j.schedule))
	}
}
