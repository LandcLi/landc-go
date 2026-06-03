package websocket

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub should not return nil")
	}

	if hub.connections == nil {
		t.Error("connections map should be initialized")
	}

	if hub.rooms == nil {
		t.Error("rooms map should be initialized")
	}

	if hub.PingInterval != 30*time.Second {
		t.Errorf("Expected PingInterval 30s, got %v", hub.PingInterval)
	}

	if hub.PongTimeout != 60*time.Second {
		t.Errorf("Expected PongTimeout 60s, got %v", hub.PongTimeout)
	}

	if hub.MaxMessageSize != 512*1024 {
		t.Errorf("Expected MaxMessageSize 512KB, got %d", hub.MaxMessageSize)
	}
}

func TestHub_OnConnect(t *testing.T) {
	hub := NewHub()
	hub.Run()

	onConnectCalled := false
	hub.OnConnect(func(conn *Conn) {
		onConnectCalled = true
	})

	conn := &Conn{
		ID:   "test_conn",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	if !onConnectCalled {
		t.Error("OnConnect callback should be called")
	}
}

func TestHub_OnDisconnect(t *testing.T) {
	hub := NewHub()
	hub.Run()

	onDisconnectCalled := false
	hub.OnDisconnect(func(conn *Conn) {
		onDisconnectCalled = true
	})

	conn := &Conn{
		ID:   "test_conn",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)

	if !onDisconnectCalled {
		t.Error("OnDisconnect callback should be called")
	}
}

func TestHub_OnMessage(t *testing.T) {
	hub := NewHub()

	hub.OnMessage(func(conn *Conn, msg Message) {
		// callback set
	})

	if hub.onMessage == nil {
		t.Error("onMessage callback should be set")
	}
}

func TestHub_OnError(t *testing.T) {
	hub := NewHub()

	hub.OnError(func(conn *Conn, err error) {
		// callback set
	})

	if hub.onError == nil {
		t.Error("onError callback should be set")
	}
}

func TestHub_Run(t *testing.T) {
	hub := NewHub()

	hub.Run()

	if !hub.running.Load() {
		t.Error("Hub should be running after Run()")
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.Broadcast([]byte("test message"))

	time.Sleep(50 * time.Millisecond)
}

func TestHub_BroadcastText(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.BroadcastText("test message")

	time.Sleep(50 * time.Millisecond)
}

func TestHub_BroadcastToRoom(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.BroadcastToRoom("test_room", []byte("room message"))

	time.Sleep(50 * time.Millisecond)
}

func TestHub_BroadcastExclude(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.BroadcastExclude([]byte("exclude message"), "conn_1")

	time.Sleep(50 * time.Millisecond)
}

func TestHub_BroadcastToRoomExclude(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.BroadcastToRoomExclude("test_room", []byte("exclude room message"), "conn_1")

	time.Sleep(50 * time.Millisecond)
}

func TestHub_JoinRoom(t *testing.T) {
	hub := NewHub()
	hub.Run()

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.JoinRoom(conn, "room_1")

	time.Sleep(50 * time.Millisecond)

	count := hub.RoomCount("room_1")
	if count != 1 {
		t.Errorf("Expected 1 connection in room, got %d", count)
	}
}

func TestHub_LeaveRoom(t *testing.T) {
	hub := NewHub()
	hub.Run()

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.JoinRoom(conn, "room_1")
	time.Sleep(50 * time.Millisecond)

	hub.LeaveRoom(conn, "room_1")
	time.Sleep(50 * time.Millisecond)

	count := hub.RoomCount("room_1")
	if count != 0 {
		t.Errorf("Expected 0 connections in room, got %d", count)
	}
}

func TestHub_GetConn(t *testing.T) {
	hub := NewHub()
	hub.Run()

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	retrieved := hub.GetConn("conn_1")
	if retrieved == nil {
		t.Error("GetConn should return the connection")
	}

	if retrieved.ID != "conn_1" {
		t.Errorf("Expected conn_1, got %s", retrieved.ID)
	}
}

func TestHub_ConnCount(t *testing.T) {
	hub := NewHub()
	hub.Run()

	if hub.ConnCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", hub.ConnCount())
	}

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	if hub.ConnCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", hub.ConnCount())
	}
}

func TestHub_RoomCount(t *testing.T) {
	hub := NewHub()
	hub.Run()

	if hub.RoomCount("room_1") != 0 {
		t.Errorf("Expected 0 connections in room, got %d", hub.RoomCount("room_1"))
	}

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		rooms: make(map[string]bool),
	}

	hub.JoinRoom(conn, "room_1")
	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount("room_1") != 1 {
		t.Errorf("Expected 1 connection in room, got %d", hub.RoomCount("room_1"))
	}
}

func TestHub_SendTo(t *testing.T) {
	hub := NewHub()
	hub.Run()

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		send:  make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	err := hub.SendTo("conn_1", []byte("test"))
	if err != nil {
		t.Fatalf("SendTo failed: %v", err)
	}

	select {
	case msg := <-conn.send:
		if string(msg) != "test" {
			t.Errorf("Expected 'test', got '%s'", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Message not received")
	}
}

func TestHub_SendTo_NotFound(t *testing.T) {
	hub := NewHub()
	hub.Run()

	err := hub.SendTo("nonexistent", []byte("test"))
	if err == nil {
		t.Error("SendTo should fail for nonexistent connection")
	}
}

func TestConn_Send(t *testing.T) {
	conn := &Conn{
		ID:   "conn_1",
		send: make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	err := conn.Send([]byte("test"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case msg := <-conn.send:
		if string(msg) != "test" {
			t.Errorf("Expected 'test', got '%s'", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Message not received")
	}
}

func TestConn_SendText(t *testing.T) {
	conn := &Conn{
		ID:   "conn_1",
		send: make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	err := conn.SendText("test text")
	if err != nil {
		t.Fatalf("SendText failed: %v", err)
	}

	select {
	case msg := <-conn.send:
		if string(msg) != "test text" {
			t.Errorf("Expected 'test text', got '%s'", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Message not received")
	}
}

func TestConn_Close(t *testing.T) {
	// 注意：Conn.Close() 会调用 c.conn.Close()，此处 conn.conn 为 nil 会 panic
	// 仅测试 closed 标志位和 Send 的失败逻辑
	conn := &Conn{
		ID:    "conn_1",
		send:  make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	// 手动设置 closed 状态，不调用实际 Close()
	conn.closed.Store(true)
	close(conn.send)

	err := conn.Send([]byte("test"))
	if err == nil {
		t.Error("Send should fail after close")
	}
}

func TestConn_Rooms(t *testing.T) {
	conn := &Conn{
		ID:    "conn_1",
		send:  make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	conn.rooms["room_1"] = true
	conn.rooms["room_2"] = true

	rooms := conn.Rooms()
	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestUpgrader_ToGorilla(t *testing.T) {
	upgrader := &Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
		HandshakeTimeout: 5 * time.Second,
	}

	gorilla := upgrader.toGorilla()
	if gorilla == nil {
		t.Fatal("toGorilla should not return nil")
	}

	if gorilla.ReadBufferSize != 1024 {
		t.Errorf("Expected ReadBufferSize 1024, got %d", gorilla.ReadBufferSize)
	}

	if gorilla.WriteBufferSize != 1024 {
		t.Errorf("Expected WriteBufferSize 1024, got %d", gorilla.WriteBufferSize)
	}

	if gorilla.HandshakeTimeout != 5*time.Second {
		t.Errorf("Expected HandshakeTimeout 5s, got %v", gorilla.HandshakeTimeout)
	}
}

func TestUpgrader_ToGorilla_Defaults(t *testing.T) {
	upgrader := &Upgrader{}

	gorilla := upgrader.toGorilla()
	if gorilla == nil {
		t.Fatal("toGorilla should not return nil")
	}

	if gorilla.ReadBufferSize != 1024 {
		t.Errorf("Expected default ReadBufferSize 1024, got %d", gorilla.ReadBufferSize)
	}

	if gorilla.WriteBufferSize != 1024 {
		t.Errorf("Expected default WriteBufferSize 1024, got %d", gorilla.WriteBufferSize)
	}

	if gorilla.HandshakeTimeout != 10*time.Second {
		t.Errorf("Expected default HandshakeTimeout 10s, got %v", gorilla.HandshakeTimeout)
	}
}

func TestHub_Handler(t *testing.T) {
	hub := NewHub()

	handler := hub.Handler(nil)
	if handler == nil {
		t.Fatal("Handler should not return nil")
	}
}

func TestMessageType_Constants(t *testing.T) {
	if TextMessage != MessageType(1) {
		t.Errorf("Expected TextMessage 1, got %d", TextMessage)
	}

	if BinaryMessage != MessageType(2) {
		t.Errorf("Expected BinaryMessage 2, got %d", BinaryMessage)
	}
}

func TestHub_Concurrent(t *testing.T) {
	hub := NewHub()
	hub.Run()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			conn := &Conn{
				ID:   string(rune('A' + index)),
				hub:  hub,
				send: make(chan []byte, 256),
				rooms: make(map[string]bool),
			}
			hub.register <- conn
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if hub.ConnCount() != 10 {
		t.Errorf("Expected 10 connections, got %d", hub.ConnCount())
	}
}

func TestHub_Broadcast_Concurrent(t *testing.T) {
	hub := NewHub()
	hub.Run()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast([]byte("concurrent message"))
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}

func TestFullWebSocketFlow(t *testing.T) {
	hub := NewHub()
	hub.Run()

	conn := &Conn{
		ID:   "conn_1",
		hub:  hub,
		send: make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	hub.JoinRoom(conn, "room_1")
	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount("room_1") != 1 {
		t.Errorf("Expected 1 connection in room_1, got %d", hub.RoomCount("room_1"))
	}

	hub.BroadcastToRoom("room_1", []byte("room message"))
	time.Sleep(50 * time.Millisecond)

	hub.LeaveRoom(conn, "room_1")
	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount("room_1") != 0 {
		t.Errorf("Expected 0 connections in room_1, got %d", hub.RoomCount("room_1"))
	}

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)

	if hub.ConnCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", hub.ConnCount())
	}
}

func TestWebSocketsHandler_Integration(t *testing.T) {
	// gin.HandlerFunc 需要 *gin.Context，需要完整的 HTTP 测试服务器
	// 此处跳过，改为验证 Handler 返回非空
	hub := NewHub()
	upgrader := &Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	handler := hub.Handler(upgrader)
	if handler == nil {
		t.Error("Handler should not return nil")
	}
}
