package service

import (
	"chat/cache"
	"chat/e"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const month = 60 * 60 * 24 * 30 // 按照30天算一个月,进行过期时间的处理

// 发送消息的类型
type SendMsg struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
}

// 回复的消息
type ReplyMsg struct {
	From    string `json:"from"`
	Code    int    `json:"code"`
	Content string `json:"content"`
}

// 用户类
type Client struct {
	ID     string
	SendID string
	Socket *websocket.Conn
	Send   chan []byte
}

// 广播类，包括广播内容和源用户
type Broadcast struct {
	Client  *Client
	Message []byte
	Type    int
}

// 用户管理
type ClientManager struct {
	Clients    map[string]*Client
	Broadcast  chan *Broadcast
	Reply      chan *Client
	Register   chan *Client
	Unregister chan *Client
}

// Message 信息转JSON (包括：发送者、接收者、内容)
type Message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

var Manager = ClientManager{
	Clients:    make(map[string]*Client), // 参与连接的用户，出于性能的考虑，需要设置最大连接数
	Broadcast:  make(chan *Broadcast),
	Register:   make(chan *Client),
	Reply:      make(chan *Client),
	Unregister: make(chan *Client),
}

func WsHandler(c *gin.Context) {
	uid := c.Query("uid")     // 自己的id
	toUid := c.Query("toUid") // 对方的id
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { // CheckOrigin解决跨域问题
			return true
		}}).Upgrade(c.Writer, c.Request, nil) // 升级成ws协议
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	// 创建一个用户实例
	client := &Client{
		ID:     createId(uid, toUid), //1发给2
		SendID: createId(toUid, uid), //2接受到1的消息
		Socket: conn,
		Send:   make(chan []byte),
	}
	// 用户注册到用户管理上
	Manager.Register <- client
	go client.Read()
	go client.Write()

}

func createId(uid, toUid string) string {
	return uid + "->" + toUid //uid的用户与touid用户通信
}

func (c *Client) Read() {
	defer func() {
		Manager.Unregister <- c //关闭前，将所有信息都传入进去
		_ = c.Socket.Close()
	}()
	for {
		c.Socket.PongHandler()
		sendMsg := new(SendMsg)
		err := c.Socket.ReadJSON(&sendMsg) // 读取json格式，如果不是json格式，会报错
		if err != nil {
			log.Println("数据格式不正确", err)
			Manager.Unregister <- c
			_ = c.Socket.Close()
			break
		}
		if sendMsg.Type == 1 { //发送消息
			r1, _ := cache.RedisClient.Get(c.ID).Result() //看看你的id有没有在缓存里
			r2, _ := cache.RedisClient.Get(c.SendID).Result()
			if r1 >= "3" && r2 == "" { // 限制必须有回复
				replyMsg := ReplyMsg{
					Code:    e.WebsocketLimit,
					Content: "达到限制",
				}
				msg, _ := json.Marshal(replyMsg)                      //进行序列化
				_ = c.Socket.WriteMessage(websocket.TextMessage, msg) //告诉对方不能再进行读操作
			}
		}

	}
}
func (c *Client) Write() {

}
