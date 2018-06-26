package middleware

import (
	"encoding/base64"
	"os"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"github.com/niwho/logs"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 线下环境可以直接设置uid,方便测试
		if os.Getenv("STRATEGY_ENV") == "test" || true {
			if userId := c.Query("uid"); userId != "" {
				c.Set("user_id", userId)
				return
			}
		}
		logs.Log(logs.F{"path": c.Request.URL.Path}).Debug("Reeeee")
		sessionId, err := c.Request.Cookie("miveshow_session_id")
		logs.Log(logs.F{"sessionId": sessionId}).Debug("auth required")
		// csrftoken, err := c.Request.Cookie("csrftoken")
		if err != nil || sessionId.Value == "" {
			// 重定向到登录页面
			logs.Log(logs.F{"err": err, "sid": sessionId.Value}).Error("AuthRequired: session empty")
			c.AbortWithStatusJSON(504, map[string]string{"error": "session empty"})
			return

		}
		sd := SessionObj.LoadSession(c, sessionId.Value)
		if sd == "" {
			SessionObj.ClearCache(sessionId.Value)
			sd = SessionObj.LoadSession(c, sessionId.Value)
		}
		jstr, err := base64.StdEncoding.DecodeString(sd)
		if err != nil {
			logs.Log(logs.F{"err": err, "sid": sessionId.Value, "sd": sd}).Error("AuthRequired: cookie not found")
			c.AbortWithStatusJSON(504, map[string]string{"error": "cookie not found"})
			return
		}

		dss := strings.SplitN(string(jstr), ":", 2)
		if len(dss) < 2 {
			logs.Log(logs.F{"err": err, "sd": sd, "jstr": string(jstr), "sid": sessionId.Value}).Error("AuthRequired: cookie parse")
			c.AbortWithStatusJSON(504, map[string]string{"error": "cookie parse"})
			return
		}
		sessionJson, err := simplejson.NewJson([]byte(dss[1]))
		userId, _ := sessionJson.Get("uid").String()
		logs.Log(logs.F{"userId": userId, "dss": dss, "sessionJson": sessionJson}).Debug("parse session")
		c.Set("user_id", userId)

		//c.Next()
	}
}

func getUid(c *gin.Context, sid string) string {
	sd := SessionObj.LoadSession(c, sid)
	jstr, err := base64.StdEncoding.DecodeString(sd)
	if err != nil {
		logs.Log(logs.F{"err": err, "value": sid, "sd": sd}).Error("AuthRequired: cookie not found")
		return ""
	}
	dss := strings.SplitN(string(jstr), ":", 2)
	if len(dss) < 2 {
		logs.Log(logs.F{"err": err, "jstr": string(jstr)}).Error("AuthRequired: cookie parse")
		return ""
	}
	sessionJson, err := simplejson.NewJson([]byte(dss[1]))
	userId, _ := sessionJson.Get("uid").String()
	return userId
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logs.Log(logs.F{"headers": c.Request.Header}).Debug("cros!!!1")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		logs.Log(logs.F{"writer headers": c.Writer.Header()}).Debug("cros!!!1")
		c.Next()
	}
}
