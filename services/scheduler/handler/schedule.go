package handler

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	types "github.com/datacommand2/cdm-cloud/services/scheduler/constants"
	"github.com/jinzhu/gorm"
	"time"
)

func validateScheduleBrokerMessage(schedule *model.Schedule) error {
	if len(schedule.Topic) == 0 {
		return InvalidTopic()
	}

	if len(schedule.Message) == 0 {
		return InvalidMessage()
	}
	return nil
}

func validateSchedule(schedule *model.Schedule) error {
	if _, err := time.LoadLocation(schedule.Timezone); err != nil || schedule.Timezone == "" {
		return InvalidTimezone(schedule.Timezone)
	}

	if schedule.StartAt == 0 {
		return InvalidStartAt(schedule.StartAt)
	}

	if schedule.EndAt == 0 || schedule.StartAt >= schedule.EndAt {
		return InvalidEndAt(schedule.EndAt)
	}

	switch schedule.Type {
	case types.ScheduleTypeSpecified:
		if time.Now().Unix() > schedule.StartAt {
			return InvalidStartAt(schedule.StartAt)
		}

	case types.ScheduleTypeMinutely:
		if schedule.IntervalMinute == nil {
			return errors.RequiredParameter("interval_minute")
		}

		if !intervalMinuteBoundary.minMax((uint64)(*schedule.IntervalMinute)) {
			return errors.OutOfRangeParameterValue("interval_minute", *schedule.IntervalMinute, intervalMinuteBoundary.Min, intervalMinuteBoundary.Max)
		}

	case types.ScheduleTypeHourly:
		if schedule.IntervalHour == nil {
			return errors.RequiredParameter("interval_hour")
		}

		if !intervalHourBoundary.minMax((uint64)(*schedule.IntervalHour)) {
			return errors.OutOfRangeParameterValue("interval_hour", *schedule.IntervalHour, intervalHourBoundary.Min, intervalHourBoundary.Max)
		}

	case types.ScheduleTypeDaily:
		if schedule.IntervalDay == nil {
			return errors.RequiredParameter("interval_day")
		}

		if !intervalDayBoundary.minMax((uint64)(*schedule.IntervalDay)) {
			return errors.OutOfRangeParameterValue("interval_day", *schedule.IntervalDay, intervalDayBoundary.Min, intervalDayBoundary.Max)
		}

		if schedule.Hour != nil && !hourBoundary.minMax(uint64(*schedule.Hour)) {
			return errors.OutOfRangeParameterValue("hour", *schedule.Hour, hourBoundary.Min, hourBoundary.Max)
		}

		if schedule.Minute != nil && !minuteBoundary.minMax(uint64(*schedule.Minute)) {
			return errors.OutOfRangeParameterValue("minute", *schedule.Minute, minuteBoundary.Min, minuteBoundary.Min)
		}

	case types.ScheduleTypeWeekly:
		if schedule.IntervalWeek == nil {
			return errors.RequiredParameter("interval_week")
		}

		if !intervalWeekBoundary.minMax((uint64)(*schedule.IntervalWeek)) {
			return errors.OutOfRangeParameterValue("interval_week", *schedule.IntervalWeek, intervalWeekBoundary.Min, intervalWeekBoundary.Max)
		}

		if schedule.DayOfWeek == nil {
			return errors.RequiredParameter("day_of_week")
		}

		if !dayOfWeekBoundary.enum(*schedule.DayOfWeek) {
			return errors.OutOfRangeParameterValue("day_of_week", *schedule.DayOfWeek, dayOfMonthBoundary.Min, dayOfMonthBoundary.Max)
		}

		if schedule.Hour != nil && !hourBoundary.minMax(uint64(*schedule.Hour)) {
			return errors.OutOfRangeParameterValue("hour", *schedule.Hour, hourBoundary.Min, hourBoundary.Max)
		}

		if schedule.Minute != nil && !minuteBoundary.minMax(uint64(*schedule.Minute)) {
			return errors.OutOfRangeParameterValue("minute", *schedule.Minute, minuteBoundary.Min, minuteBoundary.Max)
		}

	case types.ScheduleTypeDayOfMonthly:
		if schedule.IntervalMonth == nil {
			return errors.RequiredParameter("interval_month")
		}

		if !intervalMonthBoundary.enum((uint64)(*schedule.IntervalMonth)) {
			return errors.OutOfRangeParameterValue("interval_month", *schedule.IntervalMonth, intervalMonthBoundary.Min, intervalMonthBoundary.Max)
		}

		if schedule.DayOfMonth == nil {
			return errors.RequiredParameter("day_of_month")
		}

		if !dayOfMonthBoundary.enum(*schedule.DayOfMonth) {
			return errors.OutOfRangeParameterValue("day_of_month", *schedule.DayOfMonth, dayOfMonthBoundary.Min, dayOfMonthBoundary.Max)
		}

		if schedule.Hour != nil && !hourBoundary.minMax(uint64(*schedule.Hour)) {
			return errors.OutOfRangeParameterValue("hour", *schedule.Hour, hourBoundary.Min, hourBoundary.Max)
		}

		if schedule.Minute != nil && !minuteBoundary.minMax(uint64(*schedule.Minute)) {
			return errors.OutOfRangeParameterValue("minute", *schedule.Minute, minuteBoundary.Min, minuteBoundary.Max)
		}

	case types.ScheduleTypeWeekOfMonthly:
		if schedule.IntervalMonth == nil {
			return errors.RequiredParameter("interval_month")
		}

		if !intervalMonthBoundary.enum((uint64)(*schedule.IntervalMonth)) {
			return errors.OutOfRangeParameterValue("interval_month", *schedule.IntervalMonth, intervalMonthBoundary.Min, intervalMonthBoundary.Max)
		}

		if schedule.WeekOfMonth == nil {
			return errors.RequiredParameter("week_of_month")
		}

		if !weekOfMonthBoundary.enum(*schedule.WeekOfMonth) {
			return errors.OutOfRangeParameterValue("week_of_month", *schedule.WeekOfMonth, weekOfMonthBoundary.Min, weekOfMonthBoundary.Max)
		}

		if schedule.DayOfWeek == nil {
			return errors.RequiredParameter("day_of_week")
		}

		if !dayOfWeekBoundary.enum(*schedule.DayOfWeek) {
			return errors.OutOfRangeParameterValue("day_of_week", *schedule.DayOfWeek, dayOfWeekBoundary.Min, dayOfWeekBoundary.Max)
		}

		if schedule.Hour != nil && !hourBoundary.minMax(uint64(*schedule.Hour)) {
			return errors.OutOfRangeParameterValue("hour", *schedule.Hour, hourBoundary.Min, hourBoundary.Max)
		}

		if schedule.Minute != nil && !minuteBoundary.minMax(uint64(*schedule.Minute)) {
			return errors.OutOfRangeParameterValue("minute", *schedule.Minute, minuteBoundary.Min, minuteBoundary.Max)
		}

	default:
		return errors.InvalidParameterValue("type", schedule.Type, "invalid schedule type")
	}
	return nil
}

func getSchedule(db *gorm.DB, id uint64) (*model.Schedule, error) {
	if id == 0 {
		return nil, errors.RequiredParameter("id")
	}

	var s model.Schedule
	err := db.First(&s, id).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, NotFoundSchedule(id)

	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	return &s, nil
}

func createSchedule(db *gorm.DB, schedule *model.Schedule) error {
	schedule.ID = 0

	if err := db.Save(schedule).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	b, err := json.Marshal(&schedule.ID)
	if err != nil {
		return errors.Unknown(err)
	}

	err = broker.Publish(constant.TopicNoticeCreateSchedule, &broker.Message{Body: b})
	if err != nil {
		return errors.UnusableBroker(err)
	}
	return nil
}

func updateSchedule(db *gorm.DB, schedule *model.Schedule) error {
	if schedule.ID == 0 {
		return errors.RequiredParameter("id")
	}

	err := db.First(&model.Schedule{}, schedule.ID).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return NotFoundSchedule(schedule.ID)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if err := db.Save(schedule).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	b, err := json.Marshal(&schedule.ID)
	if err != nil {
		return errors.Unknown(err)
	}

	err = broker.Publish(constant.TopicNoticeUpdateSchedule, &broker.Message{Body: b})
	if err != nil {
		return errors.UnusableBroker(err)
	}
	return nil
}

func deleteSchedule(db *gorm.DB, schedule *model.Schedule) error {
	if schedule.ID == 0 {
		return errors.RequiredParameter("id")
	}

	if err := db.Delete(schedule).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	b, err := json.Marshal(&schedule.ID)
	if err != nil {
		return errors.Unknown(err)
	}

	err = broker.Publish(constant.TopicNoticeDeleteSchedule, &broker.Message{Body: b})
	if err != nil {
		return errors.UnusableBroker(err)
	}
	return nil
}
