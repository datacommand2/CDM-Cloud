package notifier

import (
	"fmt"
	conf "github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/services/notification/config"
	"github.com/datacommand2/cdm-cloud/services/notification/notifier/email"
	"github.com/jinzhu/gorm"
	"time"
)

type emailNotifier struct{}

var (
	// defaultTimeZone 기본 타임존
	defaultTimeZone = "UTC"
	// defaultLanguageSet 기본 언어셋
	defaultLanguageSet = "eng"
)

// 이메일 제목
func generateSubject(ev *model.Event, timeZone *time.Location, lang string) string {
	// TODO: 향후 template 구현
	return fmt.Sprintf("raise event %v at %v.",
		ev.Code, time.Unix(ev.CreatedAt, 0).In(timeZone))
}

// 이메일 내용
func generateBody(ev *model.Event, timeZone *time.Location, lang string) string {
	// TODO: 향후 template 구현
	return fmt.Sprintf("raise event %v at %v.",
		ev.Code, time.Unix(ev.CreatedAt, 0).In(timeZone))
}

func getOrDefaultTimeZone(db *gorm.DB, key string, value *string) string {
	if value == nil {
		c := conf.GlobalConfig(db, key)
		if c == nil {
			return defaultTimeZone
		}

		return c.Value.String()
	}

	return *value
}

func getOrDefaultLanguageSet(db *gorm.DB, key string, value *string) string {
	if value == nil {
		c := conf.GlobalConfig(db, key)
		if c == nil {
			return defaultLanguageSet
		}

		return c.Value.String()
	}

	return *value
}

func (e *emailNotifier) notify(dlv *delivery) error {
	return database.Transaction(func(db *gorm.DB) error {
		var (
			user model.User
			lang string
			tz   string
		)

		err := db.Find(&user, model.User{ID: dlv.User.ID}).Error
		switch {
		case err != nil && err == gorm.ErrRecordNotFound:
			return NotFoundUser(dlv.User.ID)

		case err != nil:
			return errors.UnusableDatabase(err)
		}

		if user.Email == nil {
			// 사용자가 Email 을 설정하지 않았다는 것은
			// 사용자가 이메일을 받지 않겠다라는 것과 같으므로, 로깅 없음
			return nil
		}

		lang = getOrDefaultLanguageSet(db, conf.GlobalLanguageSet, user.LanguageSet)

		tz = getOrDefaultTimeZone(db, conf.GlobalTimeZone, user.Timezone)

		loc, err := time.LoadLocation(tz)
		if err != nil {
			return errors.Unknown(err)
		}

		emailInfo, err := config.GetConfigEmailNotifier(db, dlv.Event.TenantID) // TODO: caching with QUEUE change
		if err != nil {
			return err
		}

		mail, err := email.NewEmail(emailInfo.Encryption, emailInfo.AuthMechanism)
		if err != nil {
			return err
		}
		err = mail.Connect(fmt.Sprintf("%v:%v", emailInfo.ServerAddress, emailInfo.ServerPort),
			emailInfo.Sender, emailInfo.AuthPassword)
		if err != nil {
			return errors.Unknown(err)
		}
		err = mail.Auth()
		if err != nil {
			return errors.Unknown(err)
		}
		err = mail.Send(*user.Email, generateSubject(dlv.Event, loc, lang), generateBody(dlv.Event, loc, lang))
		if err != nil {
			return errors.Unknown(err)
		}
		err = mail.Close()
		if err != nil {
			return errors.Unknown(err)
		}

		return nil
	})
}
