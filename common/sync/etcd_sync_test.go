package sync

import (
	"context"
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
	abnormalSync Sync
)

var (
	defaultServiceName  = "cdm-cloud-etcd"
	defaultServiceHost  = "etcd"
	defaultServicePort  = 2379
	defaultAuthUsername = "cdm"
	defaultAuthPassword = "password"
)

func TestMain(m *testing.M) {
	setup()
	defer teardown()

	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func setup() {
	_ = logger.Init(mlogger.WithLevel(mlogger.TraceLevel))
	mlogger.DefaultLogger = mlogger.NewHelper(logger.NewLogger(mlogger.WithLevel(mlogger.TraceLevel)))

	// discover service
	s, err := test.DiscoverService(defaultServiceName)
	if err != nil {
		panic(err)
	}
	logger.Errorf("test :%v", s)
	// lookup service address from host
	if len(s) == 0 {
		addr, err := test.LookupServiceAddress(defaultServiceHost, defaultServicePort)
		if err != nil {
			panic(err)
		}

		for _, a := range addr {
			s = append(s, test.Service{Address: a})
		}
	}

	// normal services
	var normal []test.Service
	for _, v := range s {
		normal = append(normal, test.Service{
			// normal case
			Name:    "sync_test_normal",
			Address: v.Address,
		})
	}
	// abnormal services
	var abnormal = []test.Service{
		{
			Name:    "sync_test_abnormal",
			Address: "127.0.0.1:1111",
		},
	}

	for _, s := range append(normal, abnormal...) {
		s.Register()
	}

	// wait for register services
	time.Sleep(5 * time.Second)

	DefaultSync, err = NewEtcdSync(
		"sync_test_normal",
		Auth(defaultAuthUsername, defaultAuthPassword),
		HeartbeatInterval(10*time.Second),
		TTL(10),
	)
	if err != nil {
		panic(err)
	}

	abnormalSync, err = NewEtcdSync("sync_test_normal")
	if err != nil {
		panic(err)
	}
}

func teardown() {
	_ = DefaultSync.Close()
}

func TestNewEtcdSync(t *testing.T) {
	for _, tc := range []struct {
		Registry    registry.Registry
		ServiceName string
		Success     bool
	}{
		{
			ServiceName: "sync_test_normal",
			Success:     true,
		},
		{
			Registry:    etcd.NewRegistry(registry.Addrs("inactive:1111")),
			ServiceName: "sync_test_normal",
			Success:     false,
		},
		{
			ServiceName: "sync_test_unknown",
			Success:     false,
		},
	} {
		s, err := NewEtcdSync(tc.ServiceName, Registry(tc.Registry))
		if tc.Success {
			assert.NoError(t, err)
			_ = s.Close()
		} else {
			assert.Error(t, err)
		}
	}
}

func TestCampaignLeaderAndResign(t *testing.T) {
	_, err := abnormalSync.CampaignLeader(context.TODO(), "path")
	assert.Error(t, err)

	var leader1 Leader
	leader1, err = DefaultSync.CampaignLeader(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var leader2 Leader
		leader2, err = DefaultSync.CampaignLeader(context.TODO(), "path")
		assert.NoError(t, leader2.Resign(context.TODO()))
		assert.NoError(t, leader2.Close())
		ch <- struct{}{}
	}()
	assert.NoError(t, leader1.Resign(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

func TestStatus(t *testing.T) {
	leader1, err := DefaultSync.CampaignLeader(context.TODO(), "path")
	assert.NoError(t, err)

	boolCh := leader1.Status()

	select {
	case <-time.After(2 * time.Second):
	case <-boolCh:
		assert.Fail(t, "status error")

	}

	go func() {
		time.Sleep(2 * time.Second)
		var leader2 Leader
		leader2, err = DefaultSync.CampaignLeader(context.TODO(), "path")
		assert.NoError(t, leader2.Resign(context.TODO()))
	}()

	assert.NoError(t, leader1.Resign(context.TODO()))

	select {
	case <-boolCh:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

func TestLockAndUnlock(t *testing.T) {
	_, err := abnormalSync.Lock(context.TODO(), "test")
	assert.Error(t, err)

	var locker1 Mutex
	locker1, err = DefaultSync.Lock(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var locker2 Mutex
		locker2, err = DefaultSync.Lock(context.TODO(), "path")
		assert.NoError(t, locker2.Unlock(context.TODO()))
		assert.NoError(t, locker2.Close())
		ch <- struct{}{}
	}()
	assert.NoError(t, locker1.Unlock(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

/* data race
func TestCampaignLeaderWithContextTimeout(t *testing.T) {
	leader1, err := DefaultSync.CampaignLeader(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var leader2 Leader
		leader2, err = DefaultSync.CampaignLeader(context.TODO(), "path")
		assert.NoError(t, leader2.Resign(context.TODO()))

		ch <- struct{}{}
	}()

	go func() {
		time.Sleep(1 * time.Second)

		ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
		defer cancel()

		_, err = DefaultSync.CampaignLeader(ctx, "path")
		assert.Error(t, err)

		ch <- struct{}{}
	}()

	time.Sleep(3 * time.Second)
	assert.NoError(t, leader1.Resign(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

func TestCampaignLeaderWithContextCancel(t *testing.T) {
	leader1, err := DefaultSync.CampaignLeader(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var leader2 Leader
		leader2, err = DefaultSync.CampaignLeader(context.TODO(), "path")
		assert.NoError(t, leader2.Resign(context.TODO()))

		ch <- struct{}{}
	}()

	go func() {
		time.Sleep(1 * time.Second)

		ctx, cancel := context.WithCancel(context.TODO())

		time.AfterFunc(2*time.Second, func() {
			cancel()
		})

		_, err = DefaultSync.CampaignLeader(ctx, "path")
		assert.Error(t, err)

		ch <- struct{}{}
	}()

	time.Sleep(3 * time.Second)
	assert.NoError(t, leader1.Resign(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

func TestLockContextTimeout(t *testing.T) {
	locker1, err := DefaultSync.Lock(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var locker2 Mutex
		locker2, err = DefaultSync.Lock(context.TODO(), "path")
		assert.NoError(t, locker2.Unlock(context.TODO()))

		ch <- struct{}{}
	}()

	go func() {
		time.Sleep(1 * time.Second)

		ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
		defer cancel()

		_, err = DefaultSync.Lock(ctx, "path")
		assert.Error(t, err)

		ch <- struct{}{}
	}()

	time.Sleep(3 * time.Second)
	assert.NoError(t, locker1.Unlock(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}

func TestLockContextCancel(t *testing.T) {
	locker1, err := DefaultSync.Lock(context.TODO(), "path")
	assert.NoError(t, err)

	ch := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)

		var locker2 Mutex
		locker2, err = DefaultSync.Lock(context.TODO(), "path")
		assert.NoError(t, locker2.Unlock(context.TODO()))

		ch <- struct{}{}
	}()

	go func() {
		time.Sleep(1 * time.Second)

		ctx, cancel := context.WithCancel(context.TODO())

		time.AfterFunc(2*time.Second, func() {
			cancel()
		})

		_, err = DefaultSync.Lock(ctx, "path")
		assert.Error(t, err)

		ch <- struct{}{}
	}()

	time.Sleep(3 * time.Second)
	assert.NoError(t, locker1.Unlock(context.TODO()))

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "timeout occurred")
	}
}
*/
