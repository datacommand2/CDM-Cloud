package database

import (
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/test"
	mlogger "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/etcd"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var (
	defaultServiceName     = "cdm-cloud-cockroach"
	defaultServiceHost     = "cockroach"
	defaultServicePort     = 26257
	defaultServiceMetadata = map[string]string{"dialect": "postgres"}

	defaultDBName       = "cdm"
	defaultAuthUsername = "cdm"
	defaultAuthPassword = "password"
)

func TestMain(m *testing.M) {
	// default libraries
	_ = logger.Init(mlogger.WithLevel(mlogger.TraceLevel))
	mlogger.DefaultLogger = mlogger.NewHelper(logger.NewLogger(mlogger.WithLevel(mlogger.TraceLevel)))

	// discover service
	s, err := test.DiscoverService(defaultServiceName)
	if err != nil {
		panic(err)
	}

	// lookup service address from host
	if len(s) == 0 {
		addr, err := test.LookupServiceAddress(defaultServiceHost, defaultServicePort)
		if err != nil {
			panic(err)
		}

		for _, a := range addr {
			s = append(s, test.Service{
				Address:  a,
				Metadata: defaultServiceMetadata,
			})
		}
	}

	// normal services
	var normal []test.Service
	for _, v := range s {
		normal = append(normal, test.Service{
			// normal case
			Name:     "database_test_normal",
			Address:  v.Address,
			Metadata: v.Metadata,
		})
	}

	// abnormal services
	var abnormal = []test.Service{
		{
			// abnormal case: invalid address
			Name:     "database_test_abnormal",
			Address:  "127.0.0.1:1111",
			Metadata: defaultServiceMetadata,
		},
		{
			// abnormal case: no dialect
			Name:     "database_test_abnormal",
			Address:  "127.0.0.1:1111",
			Metadata: map[string]string{},
		},
		{
			// abnormal case: inactive dialect
			Name:     "database_test_abnormal",
			Address:  "127.0.0.1:1111",
			Metadata: map[string]string{"dialect": "mysql"},
		},
		{
			// abnormal case unsupported dialect
			Name:     "database_test_abnormal",
			Address:  "127.0.0.1:1111",
			Metadata: map[string]string{"dialect": "unsupported"},
		},
	}

	// register services for test
	for _, s := range append(normal, abnormal...) {
		s.Register()
	}

	// wait for register services
	time.Sleep(5 * time.Second)

	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func TestInit(t *testing.T) {
	Init(
		"database_test_normal",
		defaultDBName,
		defaultAuthUsername,
		defaultAuthPassword,
		SSLEnable(false),
		HeartbeatInterval(5*time.Second),
		ReconnectInterval(10*time.Second),
	)
}

func TestOpen(t *testing.T) {
	for _, tc := range []struct {
		Registry    registry.Registry
		ServiceName string
		Username    string
		Password    string
		Error       bool
	}{
		{
			ServiceName: "database_test_normal",
			Username:    defaultAuthUsername,
			Password:    defaultAuthPassword,
			Error:       false,
		},
		{
			Registry:    etcd.NewRegistry(registry.Addrs("inactive:1111")),
			ServiceName: "database_test_normal",
			Username:    defaultAuthUsername,
			Password:    defaultAuthPassword,
			Error:       true,
		},
		{
			ServiceName: "database_test_normal",
			Username:    "unknown",
			Password:    "unknown",
			Error:       true,
		},
		{
			ServiceName: "database_test_abnormal",
			Username:    "unknown",
			Password:    "unknown",
			Error:       true,
		},
		{
			ServiceName: "database_test_unknown",
			Username:    "unknown",
			Password:    "unknown",
			Error:       true,
		},
	} {
		db, err := Open(
			tc.ServiceName,
			defaultDBName,
			tc.Username,
			tc.Password,
			Registry(tc.Registry),
		)
		if tc.Error {
			assert.Nil(t, db)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			assert.Nil(t, err)
		}

		if db != nil {
			_ = db.Close()
			_ = db.Close()
		}
	}
}

func TestOpenDefault(t *testing.T) {
	Init(
		"database_test_normal",
		defaultDBName,
		defaultAuthUsername,
		defaultAuthPassword,
		SSLEnable(false),
		HeartbeatInterval(5*time.Second),
		ReconnectInterval(10*time.Second),
	)

	for _, tc := range []struct {
		Registry registry.Registry
		Success  bool
	}{
		{
			Registry: etcd.NewRegistry(registry.Addrs("inactive:1111")),
			Success:  false,
		}, {
			Success: true,
		},
	} {
		db, err := OpenDefault(Registry(tc.Registry))
		if tc.Success {
			assert.NoError(t, err)
			assert.NotNil(t, db)
		} else {
			assert.Error(t, err)
		}

		if db != nil {
			_ = db.Close()
		}
	}
}
