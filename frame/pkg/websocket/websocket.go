package websocket

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Upgrader WebSocket 升级器配置
type Upgrader struct {
	// ReadBufferSize 读缓冲区大小
	ReadBufferSize int
	// WriteBufferSize 写缓冲区大小
	WriteBufferSize int
	// CheckOrigin 跨域检查（默认允许所有）
	CheckOrigin func(r *http.Request) bool
	// HandshakeTimeout 握手超时
	HandshakeTimeout time.Duration
}

func (u *Upgrader) toGorilla() *websocket.Upgrader {
	readBuf := u.ReadBufferSize
	if readBuf <= 0 {
		readBuf = 1024
	}
	writeBuf := u.WriteBufferSize
	if writeBuf <= 0 {
		writeBuf = 1024
	}
	checkOrigin := u.CheckOrigin
	if checkOrigin == nil {
		// 默认同源校验：仅允许与请求 Host 一致的 Origin，防止 CSWSH（跨站 WebSocket 劫持）
		checkOrigin = defaultCheckOrigin
	}
	timeout := u.HandshakeTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &websocket.Upgrader{
		ReadBufferSize:   readBuf,
		WriteBufferSize:  writeBuf,
		CheckOrigin:      checkOrigin,
		HandshakeTimeout: timeout,
	}
}

// defaultCheckOrigin 默认跨域校验：要求 Origin 与请求 Host 同源。
// 非浏览器客户端（无 Origin 头）默认放行。
func defaultCheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// 无 Origin 头的客户端（如 curl、服务端 SDK）直接放行
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// 校验协议与 Host（防止跨协议攻击，如 https 站点到 http 的 WS）
	var expectedScheme string
	if r.TLS != nil {
		expectedScheme = "https"
	} else {
		expectedScheme = "http"
	}
	if originURL.Scheme != expectedScheme {
		return false
	}

	return strings.EqualFold(originURL.Host, r.Host)
}

// MessageType 消息类型
type MessageType int

const (
	TextMessage   MessageType = websocket.TextMessage
	BinaryMessage MessageType = websocket.BinaryMessage
)

// Message WebSocket 消息
type Message struct {
	Type MessageType
	Data []byte
}

// Conn WebSocket 连接封装
type Conn struct {
	ID       string
	conn     *websocket.Conn
	hub      *Hub
	send     chan []byte
	rooms    map[string]bool
	mu       sync.RWMutex
	closed   atomic.Bool
	Metadata sync.Map // 用户自定义数据
}

// Send 发送文本消息
func (c *Conn) Send(data []byte) error {
	if c.closed.Load() {
		return fmt.Errorf("connection %s is closed", c.ID)
	}
	select {
	case c.send <- data:
		return nil
	default:
		return fmt.Errorf("connection %s send buffer full", c.ID)
	}
}

// SendText 发送文本消息
func (c *Conn) SendText(text string) error {
	return c.Send([]byte(text))
}

// Close 关闭连接
func (c *Conn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.send)
		c.conn.Close()
		if c.hub != nil {
			c.hub.unregister <- c
		}
	}
}

// Rooms 获取连接加入的所有房间
func (c *Conn) Rooms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rooms := make([]string, 0, len(c.rooms))
	for room := range c.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// Hub WebSocket 连接中心（管理所有连接和房间）
type Hub struct {
	// 所有连接
	connections map[string]*Conn
	// 房间 -> 连接ID集合
	rooms map[string]map[string]*Conn
	// 注册/注销通道
	register   chan *Conn
	unregister chan *Conn
	// 广播
	broadcast chan *broadcastMsg
	// 房间操作
	joinRoom  chan *roomOp
	leaveRoom chan *roomOp
	// 事件回调
	onConnect    func(conn *Conn)
	onDisconnect func(conn *Conn)
	onMessage    func(conn *Conn, msg Message)
	onError      func(conn *Conn, err error)

	mu      sync.RWMutex
	running atomic.Bool

	// 配置
	PingInterval   time.Duration
	PongTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxMessageSize int64
	SendBufferSize int
}

type broadcastMsg struct {
	data    []byte
	room    string // 为空表示全局广播
	exclude string // 排除的连接ID
}

type roomOp struct {
	conn *Conn
	room string
}

// NewHub 创建连接中心
func NewHub() *Hub {
	return &Hub{
		connections:    make(map[string]*Conn),
		rooms:          make(map[string]map[string]*Conn),
		register:       make(chan *Conn, 64),
		unregister:     make(chan *Conn, 64),
		broadcast:      make(chan *broadcastMsg, 256),
		joinRoom:       make(chan *roomOp, 64),
		leaveRoom:      make(chan *roomOp, 64),
		PingInterval:   30 * time.Second,
		PongTimeout:    60 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxMessageSize: 512 * 1024, // 512KB
		SendBufferSize: 256,
	}
}

// OnConnect 设置连接建立回调
func (h *Hub) OnConnect(fn func(conn *Conn)) {
	h.onConnect = fn
}

// OnDisconnect 设置连接断开回调
func (h *Hub) OnDisconnect(fn func(conn *Conn)) {
	h.onDisconnect = fn
}

// OnMessage 设置消息接收回调
func (h *Hub) OnMessage(fn func(conn *Conn, msg Message)) {
	h.onMessage = fn
}

// OnError 设置错误回调
func (h *Hub) OnError(fn func(conn *Conn, err error)) {
	h.onError = fn
}

// Run 启动 Hub（需在 goroutine 中运行）
func (h *Hub) Run() {
	if h.running.CompareAndSwap(false, true) {
		go h.run()
	}
}

//nolint:gocyclo // Hub 主循环（注册/注销/广播/心跳/清理）分支多，拆分收益低
func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn.ID] = conn
			h.mu.Unlock()
			if h.onConnect != nil {
				h.onConnect(conn)
			}

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn.ID]; ok {
				delete(h.connections, conn.ID)
				// 从所有房间移除
				for room := range conn.rooms {
					if members, ok := h.rooms[room]; ok {
						delete(members, conn.ID)
						if len(members) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			h.mu.Unlock()
			if h.onDisconnect != nil {
				h.onDisconnect(conn)
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			if msg.room == "" {
				// 全局广播
				for id, conn := range h.connections {
					if id == msg.exclude {
						continue
					}
					select {
					case conn.send <- msg.data:
					default:
					}
				}
			} else {
				// 房间广播
				if members, ok := h.rooms[msg.room]; ok {
					for id, conn := range members {
						if id == msg.exclude {
							continue
						}
						select {
						case conn.send <- msg.data:
						default:
						}
					}
				}
			}
			h.mu.RUnlock()

		case op := <-h.joinRoom:
			h.mu.Lock()
			if h.rooms[op.room] == nil {
				h.rooms[op.room] = make(map[string]*Conn)
			}
			h.rooms[op.room][op.conn.ID] = op.conn
			op.conn.mu.Lock()
			op.conn.rooms[op.room] = true
			op.conn.mu.Unlock()
			h.mu.Unlock()

		case op := <-h.leaveRoom:
			h.mu.Lock()
			if members, ok := h.rooms[op.room]; ok {
				delete(members, op.conn.ID)
				if len(members) == 0 {
					delete(h.rooms, op.room)
				}
			}
			op.conn.mu.Lock()
			delete(op.conn.rooms, op.room)
			op.conn.mu.Unlock()
			h.mu.Unlock()
		}
	}
}

// Broadcast 广播消息给所有连接
func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- &broadcastMsg{data: data}
}

// BroadcastText 广播文本消息
func (h *Hub) BroadcastText(text string) {
	h.Broadcast([]byte(text))
}

// BroadcastToRoom 向指定房间广播
func (h *Hub) BroadcastToRoom(room string, data []byte) {
	h.broadcast <- &broadcastMsg{data: data, room: room}
}

// BroadcastExclude 广播（排除某连接）
func (h *Hub) BroadcastExclude(data []byte, excludeID string) {
	h.broadcast <- &broadcastMsg{data: data, exclude: excludeID}
}

// BroadcastToRoomExclude 房间广播（排除某连接）
func (h *Hub) BroadcastToRoomExclude(room string, data []byte, excludeID string) {
	h.broadcast <- &broadcastMsg{data: data, room: room, exclude: excludeID}
}

// JoinRoom 将连接加入房间
func (h *Hub) JoinRoom(conn *Conn, room string) {
	h.joinRoom <- &roomOp{conn: conn, room: room}
}

// LeaveRoom 将连接移出房间
func (h *Hub) LeaveRoom(conn *Conn, room string) {
	h.leaveRoom <- &roomOp{conn: conn, room: room}
}

// GetConn 根据 ID 获取连接
func (h *Hub) GetConn(id string) *Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[id]
}

// ConnCount 获取当前连接数
func (h *Hub) ConnCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// RoomCount 获取房间中的连接数
func (h *Hub) RoomCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if members, ok := h.rooms[room]; ok {
		return len(members)
	}
	return 0
}

// SendTo 向指定连接发送消息
func (h *Hub) SendTo(connID string, data []byte) error {
	h.mu.RLock()
	conn, ok := h.connections[connID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", connID)
	}
	return conn.Send(data)
}

// Handler 创建 Gin WebSocket handler
func (h *Hub) Handler(upgrader *Upgrader) gin.HandlerFunc {
	if upgrader == nil {
		upgrader = &Upgrader{}
	}
	gorillaUpgrader := upgrader.toGorilla()

	return func(c *gin.Context) {
		ws, err := gorillaUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		conn := &Conn{
			ID:    generateConnID(),
			conn:  ws,
			hub:   h,
			send:  make(chan []byte, h.SendBufferSize),
			rooms: make(map[string]bool),
		}

		h.register <- conn

		go h.writePump(conn)
		go h.readPump(conn)
	}
}

func (h *Hub) readPump(conn *Conn) {
	defer conn.Close()

	conn.conn.SetReadLimit(h.MaxMessageSize)
	_ = conn.conn.SetReadDeadline(time.Now().Add(h.PongTimeout))
	conn.conn.SetPongHandler(func(string) error {
		_ = conn.conn.SetReadDeadline(time.Now().Add(h.PongTimeout))
		return nil
	})

	for {
		msgType, data, err := conn.conn.ReadMessage()
		if err != nil {
			if h.onError != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				h.onError(conn, err)
			}
			return
		}

		if h.onMessage != nil {
			h.onMessage(conn, Message{
				Type: MessageType(msgType),
				Data: data,
			})
		}
	}
}

func (h *Hub) writePump(conn *Conn) {
	ticker := time.NewTicker(h.PingInterval)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-conn.send:
			if !ok {
				_ = conn.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			_ = conn.conn.SetWriteDeadline(time.Now().Add(h.WriteTimeout))
			if err := conn.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				if h.onError != nil {
					h.onError(conn, err)
				}
				return
			}

		case <-ticker.C:
			_ = conn.conn.SetWriteDeadline(time.Now().Add(h.WriteTimeout))
			if err := conn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var connIDCounter atomic.Int64

func generateConnID() string {
	id := connIDCounter.Add(1)
	return fmt.Sprintf("conn_%d_%d", time.Now().UnixMilli(), id)
}
