//go:build with_database_connection
// +build with_database_connection

package database

import (
	"fmt"
	"testing"
	"time"
)

// ConnectionWrapper 의 데이터베이스
func TestReconnect(t *testing.T) {
	db, err := Open(
		ServiceName("database_test_normal"),
		DBName("cdm"),
		Auth("root", "password"),
		HeartbeatInterval(time.Second),
		ReconnectInterval(time.Second),
	)
	if err != nil {
		return
	}
	defer func() {
		_ = db.Close()
	}()

	fmt.Print(`
==========================================
Way to database reconnection manually test
(wait 30 seconds for manually test)
==========================================

1. Stop connected database server
2. Wait about 2 seconds
3. Start database server (If has other available database server, skip this step)
4. See log and check if tried to reconnect

`)
	time.Sleep(30 * time.Second)
}
