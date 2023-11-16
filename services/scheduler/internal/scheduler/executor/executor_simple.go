package executor

import (
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	types "github.com/datacommand2/cdm-cloud/services/scheduler/constants"
	"github.com/datacommand2/cdm-cloud/services/scheduler/internal/scheduler/executor/internal/gocron"
	"strconv"
	"strings"
	"sync"
	"time"
)

func convertDayOfWeek(dayOfWeek string) (int, string) {
	switch dayOfWeek {
	case "sun":
		return 0, "Sunday"
	case "mon":
		return 1, "Monday"
	case "tue":
		return 2, "Tuesday"
	case "wed":
		return 3, "Wednesday"
	case "thu":
		return 4, "Thursday"
	case "fri":
		return 5, "Friday"
	case "sat":
		return 6, "Saturday"
	}
	return 0, ""
}

func convertToOrdinalNumber(num string) string {
	switch num {
	case "1":
		return "1st"
	case "2":
		return "2nd"
	case "3":
		return "3rd"
	default:
		return num + "th"
	}
}

func calculateStartTimeFromNow(t time.Time, d time.Duration) time.Time {
	now := time.Now()
	run := t
	for {
		if run.After(now) || run.Equal(now) {
			break
		}
		run = run.Add(d)
	}
	return run
}

// SimpleExecutor cron expression 형식이 아닌 interval 형식의 스케줄 익스 큐터
type SimpleExecutor struct {
	*gocron.Scheduler
	loc  *time.Location
	jobs map[uint64]*SimpleJob
	lock *sync.Mutex
}

// NewSimpleExecutor cron expression 형식이 아닌 interval 형식의 익스 큐터 생성
func NewSimpleExecutor(timezone string) Executor {
	loc, _ := time.LoadLocation(timezone)

	scheduler := gocron.NewScheduler(loc)
	scheduler.StartAsync()

	return &SimpleExecutor{
		Scheduler: scheduler,
		loc:       loc,
		lock:      &sync.Mutex{},
		jobs:      make(map[uint64]*SimpleJob),
	}
}

func (e *SimpleExecutor) addJob(schedule *model.Schedule, scheduler *gocron.Scheduler) error {
	job := &SimpleJob{schedule: schedule, delete: e.DeleteJob}

	var err error
	job.Job, err = scheduler.Do(job.Run)
	if err != nil {
		err = errors.Unknown(err)
		return err
	}

	e.jobs[schedule.ID] = job

	return nil
}

// CalculateJobNextRunTime simple 스케줄 잡의 실행 시간을 구하는 함수
func (e *SimpleExecutor) CalculateJobNextRunTime(schedule *model.Schedule) (*int64, error) {
	var (
		minute, hour uint
		nextRunTime  int64
	)

	if schedule.Minute != nil {
		minute = *schedule.Minute
	}

	if schedule.Hour != nil {
		hour = *schedule.Hour
	}

	startAt := time.Unix(schedule.StartAt, 0)
	switch schedule.Type {
	case types.ScheduleTypeSpecified:
		nextRunTime = schedule.StartAt

	case types.ScheduleTypeMinutely:
		interval := *schedule.IntervalMinute
		nextRunTime = calculateStartTimeFromNow(startAt, time.Minute*time.Duration(interval)).Unix()

	case types.ScheduleTypeHourly:
		interval := *schedule.IntervalHour
		nextRunTime = calculateStartTimeFromNow(startAt, time.Hour*time.Duration(interval)).Unix()

	case types.ScheduleTypeDaily:
		interval := *schedule.IntervalDay
		startAt = time.Date(startAt.Year(), startAt.Month(), startAt.Day(), int(hour), int(minute), 0, 0, e.loc)
		nextRunTime = calculateStartTimeFromNow(startAt, 24*time.Hour*time.Duration(interval)).Unix()

	case types.ScheduleTypeWeekly:
		interval := *schedule.IntervalWeek
		dayOfWeek, _ := convertDayOfWeek(*schedule.DayOfWeek)
		startAt = time.Date(startAt.Year(), startAt.Month(), startAt.Day(), int(hour), int(minute), 0, 0, e.loc)

		diff := dayOfWeek - int(startAt.Weekday())
		if diff < 0 {
			diff = 7 + diff
		}

		startAt = startAt.AddDate(0, 0, diff)
		nextRunTime = calculateStartTimeFromNow(startAt, 7*24*time.Hour*time.Duration(interval)).Unix()

	default:
		return nil, UnsupportedScheduleType(schedule.Type)
	}
	return &nextRunTime, nil
}

// CreateJob simple 익스큐터에 job 생성 및 등록
func (e *SimpleExecutor) CreateJob(schedule *model.Schedule) error {
	if !schedule.ActivationFlag {
		return nil
	}

	var minute, hour uint
	if schedule.Minute != nil {
		minute = *schedule.Minute
	}

	if schedule.Hour != nil {
		hour = *schedule.Hour
	}

	startAt, err := e.CalculateJobNextRunTime(schedule)
	if err != nil {
		return err
	}

	t := time.Unix(*startAt, 0).In(e.loc)

	e.lock.Lock()
	defer e.lock.Unlock()

	var s *gocron.Scheduler
	switch schedule.Type {
	case types.ScheduleTypeSpecified:
		s = e.Every(uint64(0)).StartAt(t)

	case types.ScheduleTypeMinutely:
		s = e.Every(uint64(*schedule.IntervalMinute)).Minutes().StartAt(t)

	case types.ScheduleTypeHourly:
		s = e.Every(uint64(*schedule.IntervalHour)).Hours().StartAt(t)

	case types.ScheduleTypeDaily:
		s = e.Every(uint64(*schedule.IntervalDay)).Days().At(fmt.Sprintf("%02d:%02d", hour, minute)).StartAt(t)

	case types.ScheduleTypeWeekly:
		s = e.Every(uint64(*schedule.IntervalWeek)).Weeks().At(fmt.Sprintf("%02d:%02d", hour, minute)).StartAt(t)
	}

	return e.addJob(schedule, s)
}

// DeleteJob simple 익스 큐터에 job 삭제
func (e *SimpleExecutor) DeleteJob(schedule *model.Schedule) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if job, ok := e.jobs[schedule.ID]; ok {
		e.RemoveByReference(job.Job)
		delete(e.jobs, schedule.ID)
	}
}

// GetJob simple 익스 큐터에 등록된 스케줄 job 조회
func (e *SimpleExecutor) GetJob(schedule *model.Schedule) Job {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.jobs[schedule.ID]
}

// Close simple 익스 큐터에 스케줄 종료
func (e *SimpleExecutor) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Scheduler.Stop()
	for k := range e.jobs {
		delete(e.jobs, k)
	}
}

// SimpleJob simple 익스 큐터에 스케줄 잡 구조체
type SimpleJob struct {
	*gocron.Job
	schedule *model.Schedule
	delete   func(*model.Schedule)
}

// Run simple 익스 큐터 에스케줄 시, 실제 실행 될 함수
func (j *SimpleJob) Run() {
	runtime := time.Now().Unix()

	if runtime >= j.schedule.EndAt {
		j.delete(j.schedule)
		return
	}

	var msg = broker.Message{
		Header: map[string]string{"runtime": strconv.FormatInt(runtime, 10)},
		Body:   []byte(j.schedule.Message),
	}

	if err := broker.Publish(j.schedule.Topic, &msg); err != nil {
		logger.Errorf("[executor_simple-Run] Error occurred in schedule running cause: %v", err)
	} else {
		logger.Infof("[executor_simple-Run] Published schedule( %d:%s - %s ).", j.schedule.ID, j.schedule.Topic, getScheduleMessage(j.schedule))
	}

	if j.schedule.Type == types.ScheduleTypeSpecified {
		j.delete(j.schedule)
	}
}

func getScheduleMessage(s *model.Schedule) string {
	var (
		msg       string
		min, hour uint
	)
	if s.Minute != nil {
		min = *s.Minute
	}
	if s.Hour != nil {
		hour = *s.Hour
	}

	switch s.Type {
	case types.ScheduleTypeSpecified:
		msg = fmt.Sprintf("specified at %s", time.Unix(s.StartAt, 0).Format("2006/01/02 15:04"))

	case types.ScheduleTypeMinutely:
		msg = fmt.Sprintf("every %d minute(s) since %s", *s.IntervalMinute, time.Unix(s.StartAt, 0).Format("2006/01/02 15:04"))

	case types.ScheduleTypeHourly:
		msg = fmt.Sprintf("every %d hour(s) since %s", *s.IntervalHour, time.Unix(s.StartAt, 0).Format("2006/01/02 15:04"))

	case types.ScheduleTypeDaily:
		if s.Hour == nil && s.Minute == nil {
			hour = uint(time.Unix(s.StartAt, 0).Hour())
			min = uint(time.Unix(s.StartAt, 0).Minute())
		}
		msg = fmt.Sprintf("every %d day(s) at %d:%d", *s.IntervalDay, hour, min)

	case types.ScheduleTypeWeekly:
		_, day := convertDayOfWeek(*s.DayOfWeek)
		msg = fmt.Sprintf("every %d week(s) on %s at %d:%d", *s.IntervalWeek, day, hour, min)

	case types.ScheduleTypeWeekOfMonthly:
		week := strings.Replace(*s.WeekOfMonth, "#", "", -1)
		_, day := convertDayOfWeek(*s.DayOfWeek)
		msg = fmt.Sprintf("every %d month(s) on %s week %s at %d:%d", *s.IntervalMonth, convertToOrdinalNumber(week), day, hour, min)

	case types.ScheduleTypeDayOfMonthly:
		msg = fmt.Sprintf("every %d month(s) on %s day at %d:%d", *s.IntervalMonth, convertToOrdinalNumber(*s.DayOfMonth), hour, min)

	default:
		msg = fmt.Sprintf("%s:unknown", s.Type)

	}
	return msg
}
