package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/niwho/logs"
	"pusic/push/message/common"
)

var SessionObj *Session

type Session struct {
	cache *common.CacheManager

	localCache      *common.LocalCache
	SessionDbClient *common.DBClient
	RedisClient     *common.RedisClient

	ttl int
}

func InitSession(db *common.DBClient, redis *common.RedisClient) {
	SessionObj = &Session{
		cache:      common.NewCacheManager(),
		localCache: common.NewLocalCache(1e10, 24*time.Hour),
		ttl:        60,

		SessionDbClient: db,
		RedisClient:     redis,
	}
	// 注意添加的顺序
	SessionObj.cache.AddChains(SessionObj.getFromLocal, SessionObj.getFromRedis, SessionObj.getFromDataBase)
}

func (sess *Session) ClearCache(key string) {
	if key != "" {
		sess.localCache.Remove(key)
		sess.RedisClient.Remove(fmt.Sprintf("golang_session_%s", key))
	}
}

func (sess *Session) getFromLocal(c *common.CacheRequest) {
	val, found := sess.localCache.Get(c.GetKey())
	if found {
		c.Abort()
		c.SetData(val)
		return
	}
	c.Next()
	// 处理下一级返回值
	if dat, ok := c.GetData(); ok {
		if dat != "" {
			sess.localCache.Set(c.GetKey(), dat)
		} else {
			// 保护逻辑
			logs.Log(logs.F{"err": "nodata", "key": c.GetKey()}).Error("tsession: local")
			sess.localCache.SetWithTtl(c.GetKey(), dat, 1)
		}
	}
}

func (sess *Session) getFromRedis(c *common.CacheRequest) {
	key, _ := c.GetKey().(string)
	rdval, err := sess.RedisClient.GetString(fmt.Sprintf("golang_session_%s", key))
	if err == nil {
		c.Abort()
		c.SetData(rdval)
		return
	} else {
		logs.Log(logs.F{"err": err, "key": key}).Error("tsession: getFromRedis")
	}
	c.Next()
	// 处理下一级返回值
	if dat, ok := c.GetData(); ok {
		tmpval, _ := dat.(string)
		ttl := sess.ttl
		if tmpval == "" {
			ttl = 1
		}
		sess.RedisClient.SetString(fmt.Sprintf("golang_session_%s", key), string(tmpval), ttl)
	}
}

type DjangoSession struct {
	SessionKey  string `gorm:"column:session_key"`
	SessionData string `gorm:"column:session_data"`
}

func (*DjangoSession) TableName() string {
	return "django_session"
}

func (sess *Session) getFromDataBase(c *common.CacheRequest) {
	key, ok := c.GetKey().(string)
	if !ok {
		c.Abort()
		return
	}
	var se DjangoSession
	errs := sess.SessionDbClient.RealDb.Where("session_key=?", key).First(&se).GetErrors()
	if len(errs) > 0 {
		logs.Log(logs.F{"errs": errs, "key": key}).Error("tsession: getFromDb")
	}
	c.SetData(se.SessionData)
	c.Abort()
	return
}

func (sess *Session) LoadSession(c context.Context, sid string) string {
	//sid 潜在一致性问题
	// 先从cache获取
	data := sess.cache.GetData(c, sid)
	logs.Log(logs.F{"data": data}).Debug("loadsession")
	str, _ := data.(string)
	return str
}
