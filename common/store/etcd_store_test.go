package store

import (
	"errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/test"
	"github.com/google/uuid"
	mlogger "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/etcd"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
	"time"
)

var (
	abnormalStore Store
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
			Name:    "store_test_normal",
			Address: v.Address,
		})
	}

	// abnormal services
	var abnormal = []test.Service{
		{
			Name:    "store_test_abnormal",
			Address: "127.0.0.1:1111",
		},
	}

	// register services for test
	for _, s := range append(normal, abnormal...) {
		s.Register()
	}

	// wait for register services
	time.Sleep(5 * time.Second)

	// normal test store
	DefaultStore = NewStore(
		"store_test_normal",
		DialTimeout(3*time.Second),
		Auth(defaultAuthUsername, defaultAuthPassword),
	)
	if err = DefaultStore.Connect(); err != nil {
		panic(err)
	}

	// abnormal test store
	abnormalStore = NewStore("store_test_abnormal")

	if err = abnormalStore.Connect(); err != nil {
		panic(err)
	}
}

func teardown() {
	_ = DefaultStore.Close()
}

func TestNewStore(t *testing.T) {
	for _, tc := range []struct {
		Registry    registry.Registry
		ServiceName string
		Success     bool
	}{
		{
			ServiceName: "store_test_normal",
			Success:     true,
		},
		{
			Registry:    etcd.NewRegistry(registry.Addrs("inactive:1111")),
			ServiceName: "store_test_normal",
			Success:     false,
		},
		{
			ServiceName: "store_test_unknown",
			Success:     false,
		},
	} {
		s := NewStore(tc.ServiceName, Registry(tc.Registry))
		err := s.Connect()
		if tc.Success {
			assert.NoError(t, err)
			_ = s.Close()
		} else {
			assert.Error(t, err)
		}
	}
}

func TestOptions(t *testing.T) {
	s := NewStore("store_test_normal",
		DialTimeout(3*time.Second),
		Auth(defaultAuthUsername, defaultAuthPassword),
	)
	err := s.Connect()
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, "store_test_normal", s.Options().ServiceName)

	if v := s.Options().Context.Value(username("")); v != nil {
		assert.Equal(t, defaultAuthUsername, v.(string))
	} else {
		t.Error("username is empty")
	}

	if v := s.Options().Context.Value(password("")); v != nil {
		assert.Equal(t, defaultAuthPassword, v.(string))
	} else {
		t.Error("password is empty")
	}

	if v := s.Options().Context.Value(dialTimeout("")); v != nil {
		assert.Equal(t, v.(time.Duration), 3*time.Second)
	} else {
		t.Error("dialTimeout is empty")
	}

	_ = s.Close()
}

func TestPut(t *testing.T) {
	var (
		err   error
		key   = uuid.New().String()
		value = uuid.New().String()
	)

	err = abnormalStore.Put(key, value,
		PutTTL(5*time.Second),
		PutTimeout(5*time.Second),
	)
	assert.Error(t, err)

	err = Put(key, value,
		PutTTL(5*time.Second),
		PutTimeout(5*time.Second),
	)
	assert.NoError(t, err)

	v, err := Get(key)
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	time.Sleep(6 * time.Second)

	_, err = Get(key)
	assert.Error(t, err)
}

func TestGet(t *testing.T) {
	var (
		err   error
		key   = uuid.New().String()
		value = uuid.New().String()
	)

	_, err = abnormalStore.Get(key)
	assert.Error(t, err)

	_, err = Get(key)
	assert.Error(t, err)

	err = Put(key, value,
		PutTTL(5*time.Second),
		PutTimeout(5*time.Second),
	)
	assert.NoError(t, err)

	v, err := Get(key, GetTimeout(5*time.Second))
	assert.NoError(t, err)
	assert.Equal(t, value, v)
}

func TestList(t *testing.T) {
	var err error
	var keys []string

	prefix := uuid.New().String()

	for i := 0; i < 3; i++ {
		key := path.Join(prefix, uuid.New().String())
		keys = append(keys, key)

		err := Put(key, "value",
			PutTTL(10*time.Second),
			PutTimeout(5*time.Second),
		)
		assert.NoError(t, err)
	}

	_, err = abnormalStore.List(prefix)
	assert.Error(t, err)

	actual, err := List(prefix, ListTimeout(5*time.Second))
	assert.NoError(t, err)
	assert.ElementsMatch(t, keys, actual)
}

func TestDelete(t *testing.T) {
	var err error
	var keys []string

	prefix := uuid.New().String()

	for i := 0; i < 3; i++ {
		key := path.Join(prefix, uuid.New().String())
		keys = append(keys, key)

		err := Put(key, "value",
			PutTTL(10*time.Second),
			PutTimeout(5*time.Second),
		)
		assert.NoError(t, err)
	}

	err = abnormalStore.Delete(keys[0])
	assert.Error(t, err)

	_, err = Get(keys[0])
	assert.NoError(t, err)

	err = Delete(keys[0])
	assert.NoError(t, err)

	_, err = Get(keys[0])
	assert.Error(t, err)

	_, err = List(prefix)
	assert.NoError(t, err)

	err = Delete(prefix,
		DeleteTimeout(5*time.Second),
		DeletePrefix(),
	)
	assert.NoError(t, err)

	_, err = List(prefix)
	assert.Error(t, err)
}

func TestTransaction(t *testing.T) {
	prefix := uuid.New().String()

	key := path.Join(prefix, uuid.New().String())
	val := uuid.New().String()

	// rollback put
	err := Transaction(func(txn Txn) error {
		txn.Put(key, val)
		return errors.New("do not commit")
	})

	assert.Error(t, err)

	// commit put
	err = Transaction(func(txn Txn) error {
		txn.Put(key, val)
		return nil
	})

	v, err := Get(key)
	assert.NoError(t, err)
	assert.Equal(t, val, v)

	// rollback delete
	err = Transaction(func(txn Txn) error {
		txn.Delete(prefix, DeletePrefix())
		return errors.New("do not commit")
	})

	assert.Error(t, err)

	v, err = Get(key)
	assert.NoError(t, err)
	assert.Equal(t, val, v)

	// commit delete
	err = Transaction(func(txn Txn) error {
		txn.Delete(prefix, DeletePrefix())
		return nil
	})

	assert.NoError(t, err)

	v, err = Get(key)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFoundKey, err)
}
