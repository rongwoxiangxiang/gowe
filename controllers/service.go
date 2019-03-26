package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"gowe/common"
	"strings"

	//wechatApi "github.com/silenceper/wechat"
	"github.com/astaxie/beego/context"
	"github.com/silenceper/wechat/cache"
	"github.com/silenceper/wechat/message"
	"gowe/models"
	"time"
)

var redis *cache.Redis

func init()  {
	opts := &cache.RedisOpts{
		Host: beego.AppConfig.String("redis_host"),
	}
	redis = cache.NewRedis(opts)
}

func Service(ctx *context.Context) {
	wechatConfig := config(ctx)
	flag := ctx.Input.Query(":flag")
	msg := message.MixMessage{Content:flag}
	msg.FromUserName = "dsadsadasda"
	msg.SetMsgType(message.MsgTypeText)
	res := responseEventText(msg,wechatConfig)
	fmt.Println(res)

	//wechatConfig := config(ctx)
	//server := wechatApi.NewWechat(&wechatApi.Config{
	//	AppID:          wechatConfig["Appid"].(string),
	//	AppSecret:      wechatConfig["Appsecret"].(string),
	//	Token:          wechatConfig["Token"].(string),
	//	EncodingAESKey: wechatConfig["EncodingAesKey"].(string),
	//	Cache:			redis,
	//}).GetServer(ctx.Request, ctx.ResponseWriter)
	//server.SetMessageHandler(func(msg message.MixMessage) *message.Reply {
	//	return responseEventText(msg,wechatConfig)
	//
	//})
	//
	////处理消息接收以及回复
	//err := server.Serve()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	////发送回复的消息
	//server.Send()
}

func responseEventText(msg message.MixMessage ,conf map[string]interface{}) *message.Reply {
	var reply models.Reply
	switch msg.MsgType {
	case message.MsgTypeText:
		reply = models.Reply{Wid:int64(conf["Id"].(float64)), Alias:msg.Content}.FindOne()
	case message.MsgTypeEvent:
		if msg.Event != "" {
			reply = models.Reply{Wid:int64(conf["Id"].(float64)), ClickKey:msg.EventKey}.FindOne()
		}
	default:
		reply = models.Reply{Wid:int64(conf["Id"].(float64)), Alias:msg.EventKey}.FindOne()
	}
	return replyActivity(reply, msg.FromUserName)
}


func replyActivity(reply models.Reply, userOpenId string)(msgReply *message.Reply)  {
	if reply.Id > 0 {
		switch reply.Type {
		case models.REPLY_TYPE_TEXT:
			msgReply = &message.Reply{
				MsgType: message.MsgTypeText,
				MsgData: message.NewText(reply.Success),
			}
		case models.REPLY_TYPE_CODE:
			msgReply = &message.Reply{
				MsgType: message.MsgTypeText,
				MsgData: message.NewText(doReplyCode(reply, userOpenId)),
			}
		case models.REPLY_TYPE_LUCKY:
			msgReply = &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText(reply.Success)}
		case models.REPLY_TYPE_CHECKIN:
			msgReply = &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText(reply.Success)}
		default:
			msgReply = &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText(reply.Success)}
		}
	}
	return
}

func doReplyCode(reply models.Reply, userOpenId string) string {
	wechatUser := getWechatUser(userOpenId)
	history := models.PrizeHistory{ActivityId:reply.ActivityId,Wuid:wechatUser.Id}.GetByActivityWuId()
	if len(history) > 0 {
		return strings.Replace(reply.Success, "%code%", history[0].Prize, 1)
	}
	prize := models.Prize{ActivityId:reply.ActivityId, Level:int8(models.PRIZE_LEVEL_DEFAULT), Used:common.NO_VALUE}.FindOneUsedCode()
	if prize.Code != "" {
		return strings.Replace(reply.Success, "%code%", prize.Code, 1)
	}
	return reply.Fail
}

func getWechatUser(userOpenId string) (wu models.WechatUser) {
	wu.Openid = userOpenId
	return wu.GetByOpenid()
}



func config(ctx *context.Context) map[string]interface{} {
	var mp map[string]interface{}
	flag := ctx.Input.Query(":flag")
	wechatConfig := redis.Get(flag)
	if wechatConfig != nil {
		return wechatConfig.(map[string]interface{})
	}
	wechatStruct  := models.Wechat{Flag:flag}.Find()
	wechatJson, _ := json.Marshal(wechatStruct[0])
	json.Unmarshal([]byte(wechatJson), &mp)
	if err := redis.Set(flag, mp, 10 * time.Hour); err != nil {
		fmt.Println("cache: set wechat config error", err)
	}
	return mp
}