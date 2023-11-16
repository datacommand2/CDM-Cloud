## cdm-cloud-scheduler
- cdm cloud에서 사용하는 스케줄러 서비스

## 사용법
### 패키지
```go
import scheduler "github.com/datacommand2/cdm-cloud/services/scheduler/proto"
```
### proto 파일 
```go
package scheduler;

service Scheduler {
    rpc CreateSchedule(ScheduleRequest) returns (ScheduleResponse) ;
    rpc UpdateSchedule(ScheduleRequest) returns (ScheduleResponse) ;
    rpc DeleteSchedule(ScheduleRequest) returns (Empty) ;
    rpc CalculateNextRunTime(ScheduleNextRunTimeRequest ) returns (ScheduleNextRunTimeResponse) ;
}

message ScheduleRequest {
	uint64 id = 1;
	bool activation_flag = 2;
	string topic = 3;
	string message = 4;
	uint64 start_at = 5;
	uint64 end_at = 6;
	string type = 7;
	uint64 interval_minute = 8;
	uint64 interval_hour = 9;
	uint64 interval_day = 10;
	uint64 interval_week = 11;
	uint64 interval_month = 12;
	string week_of_month = 13;
	string day_of_month = 14;
	string day_of_week = 15;
	uint64 hour = 16;
	uint64 minute = 17;
	string timezone = 18;
}

message ScheduleResponse {
	uint64 id = 1;
	bool activation_flag = 2;
	string topic = 3;
	string message = 4;
	uint64 start_at = 5;
	uint64 end_at = 6;
	string type = 7;
	uint64 interval_minute = 8;
	uint64 interval_hour = 9;
	uint64 interval_day = 10;
	uint64 interval_week = 11;
	uint64 interval_month = 12;
	string week_of_month = 13;
	string day_of_month = 14;
	string day_of_week = 15;
	uint64 hour = 16;
	uint64 minute = 17;
	string timezone = 18;
}

message Empty {
	
}

message ScheduleNextRunTimeRequest {
	uint64 start_at = 1;
	uint64 end_at = 2;
	string type = 3;
	uint64 interval_minute = 4;
	uint64 interval_hour = 5;
	uint64 interval_day = 6;
	uint64 interval_week = 7;
	uint64 interval_month = 8;
	string week_of_month = 9;
	string day_of_month = 10;
	string day_of_week = 11;
	uint64 hour = 12;
	uint64 minute = 13;
	string timezone = 14;
}

message ScheduleNextRunTimeResponse{
    uint64 next_run_time = 1;
}
```

## 스케줄 종류
```go
ScheduleTypeSpecified = "specified"
ScheduleTypeMinutely = "minutely"
ScheduleTypeHourly = "hourly"
ScheduleTypeDaily = "daily"
ScheduleTypeWeekly = "weekly"
ScheduleTypeDayOfMonthly = "day-of-monthly"
ScheduleTypeWeekOfMonthly = "week-of-monthly"
```

> ** 모든 스케줄의 end_at 시간은 start_at 시간보다 이후로 설정 해야 하며, activation_flag 값이 true 값이 아닐 경우 스케줄은 생성 되지만,
해당 스케줄은 스케줄링 되지 않는다 **

### 특정 일시 스케줄
- 특정 일시에 실행되는 스케줄
- 지정된 start_at 시간에 스케줄링 된다.
- start_at은 현재 시간 보다 이후로 설정 해야 한다.

### 분 단위 스케줄
- 시작 일시부터 종료 일시까지 매분(m분마다)에 수행되는 스케줄
- 지정된 start_at 시간에서부터 end_at 시간까지 매분(m분마다) 스케줄링 된다.
- 스케줄 주기(interval_minute)는 최소 1분 ~ 최대 59분으로 설정 해야 한다.

### 시 단위 스케줄
- 시작 일시부터 종료 일시까지 매시간(h시간마다)에 수행되는 스케줄
- 지정된 start_at 시간에서부터 end_at 시간까지 매분(h시간마다) 스케줄링 된다.
- 스케줄 주기(interval_hour)는 최소 1시간 ~ 최대 23시간으로 설정 해야 한다.

### 일 단위 스케줄
- 시작 일시부터 종료 일시까지 매일(n일마다) h시 m분에 수행되는 스케줄
- 지정된 start_at 시간 이후로 end_at 시간까지 매일(n일마다) 스케줄링 된다.
- 스케줄 주기(interval_day)는 최소 1일 ~ 최대 15일로 설정 해야 한다.
- 스케줄링 되는 시, 분은 hour(0 ~ 23시 사이)와 minute(0 ~ 59분 사이)로 설정 해야 한다.

### 주 단위 스케줄
- 시작 일시부터 종료 일시까지 매주(n주마다) d요일 h시 m분에 수행되는 스케줄
- 지정된 start_at 시간 이후로 end_at 시간까지 매주(n주마다) d요일 h시 m분에 스케줄링 된다.
- 스케줄 주기(interval_week)는 최소 1주 ~ 최대 4주로 설정 해야 한다.
- 스케줄 요일은 "mon", "tue", "wed", "thu", "fri", "sat", "sun" 중 하나의 값으로 설정 해야 한다.
- 스케줄링 되는 시, 분은 hour(0 ~ 23시 사이)와 minute(0 ~ 59분 사이)로 설정 해야 한다.

### 월 단위 스케줄(w번째 요일)
- 시작 일시부터 종료 일시까지 매월(n개월마다) w번째(혹은 마지막) d요일 h시 m분에 수행되는 스케줄
- 지정된 start_at 시간 이후로 end_at 시간까지 매월(n개월마다) w번째(혹은 마지막) d요일 h시 m분에 스케줄링 된다.
- 스케줄 주기(interval_month)는 1, 2, 3, 4, 6, 12 개월 중 하나의 값으로 설정 해야 한다.
- w번째는 "#1", "#2", "#3", "#4", "#5", "L" 중 하나로 설정 해야 한다.(cron 표현식)
- 스케줄 요일은 "mon", "tue", "wed", "thu", "fri", "sat", "sun" 중 하나의 값으로 설정 해야 한다.
- 스케줄링 되는 시, 분은 hour(0 ~ 23시 사이)와 minute(0 ~ 59분 사이)로 설정 해야 한다.

### 월 단위 스케줄(일)
- 시작 일시부터 종료 일시까지 매월(n개월마다) d일(혹은 말일) h시 m분에 수행되는 스케줄
- 지정된 start_at 시간 이후로 end_at 시간까지 매월(n개월마다) d일(혹은 말일) h시 m분에에 스케줄링 된다.
- 스케줄 주기(interval_month)는 1, 2, 3, 4, 6, 12 개월 중 하나의 값으로 설정 해야 한다.
- 스케줄 일은 1 ~ 31일 혹은 L(마지막 일)로 설정 해야 한다.(cron 표현식)
- 스케줄링 되는 시, 분은 hour(0 ~ 23시 사이)와 minute(0 ~ 59분 사이)로 설정 해야 한다.