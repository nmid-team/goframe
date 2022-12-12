package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HughNian/nmid/pkg/logger"
	"goframe/pkg/confer"
	"reflect"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
)

type DaoRedisEx struct {
	KeyName          string
	Persistent       bool // 持久化key
	ExpireSecond     int  // 默认过期时间，单实例有效
	tempExpireSecond int  // 临时默认过期时间，单条命令有效
}

type OpOptionEx func(*DaoRedisEx)

// WithExpire 设置超时时间
func WithExpire(expire int) OpOptionEx {
	return func(p *DaoRedisEx) { p.tempExpireSecond = expire }
}

// applyOpts 应用扩展属性
func (p *DaoRedisEx) applyOpts(opts []OpOptionEx) {
	for _, opt := range opts {
		opt(p)
	}
}

// resetTempExpireSecond 重置临时过期时间
func (p *DaoRedisEx) resetTempExpireSecond() {
	p.tempExpireSecond = 0
}

// getExpire 获取过期时间
func (p *DaoRedisEx) getExpire(expire int) int {
	var expireSecond int
	switch {
	case expire != 0:
		expireSecond = expire
	case p.tempExpireSecond != 0:
		expireSecond = p.tempExpireSecond
	case p.ExpireSecond != 0:
		expireSecond = p.ExpireSecond
	}
	if expireSecond < 0 {
		expireSecond = -1
	}
	return expireSecond
}

// 获取redis连接
func (p *DaoRedisEx) getRedisConn() (redis.Conn, error) {
	return getRedisPool().Get(), nil
}

func (p *DaoRedisEx) getKey(key string) string {
	// TODO 每次都需要重复获取配置，性能优化
	prefixRedis := confer.GetGlobalConfig().Redis.Prefix
	if strings.Trim(key, " ") == "" {
		return fmt.Sprintf("%s:%s", prefixRedis, p.KeyName)
	}
	return fmt.Sprintf("%s:%s:%s", prefixRedis, p.KeyName, key)
}

func (p *DaoRedisEx) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	redisClient, err := p.getRedisConn()
	if err != nil {
		return nil, err
	}
	if redisClient.Err() != nil {
		return nil, redisClient.Err()
	}
	defer redisClient.Close()
	defer p.resetTempExpireSecond()
	return redisClient.Do(commandName, args...)
}

func (p *DaoRedisEx) doSet(cmd string, key string, value interface{}, expire int, fields ...string) (interface{}, error) {
	var data, reply interface{}
	var err error
	switch reflect.TypeOf(value).Kind() {
	case reflect.String:
		data = value
	default:
		data, err = json.Marshal(value)
		if err != nil {
			logger.Errorf("redis %s marshal data to json:%s", cmd, err.Error())
			return nil, err
		}
	}
	key = p.getKey(key)
	expire = p.getExpire(expire)
	if len(fields) == 0 {
		if expire > 0 && strings.ToUpper(cmd) == "SET" {
			reply, err = p.do(cmd, key, data, "ex", expire)
		} else {
			reply, err = p.do(cmd, key, data)
		}
	} else {
		field := fields[0]
		reply, err = p.do(cmd, key, field, data)
	}
	if err != nil {
		logger.Errorf("run redis command %s failed:error:%s,key:%s,fields:%v,data:%v", cmd, err.Error(), key, fields, value)
		return nil, err
	}
	return reply, err
}

func (p *DaoRedisEx) doSetNX(cmd string, key string, value interface{}, expire int, field ...string) (num int64, err error) {
	var (
		reply interface{}
		ok    bool
	)
	reply, err = p.doSet(cmd, key, value, expire, field...)
	if err != nil {
		return
	}
	num, ok = reply.(int64)
	if !ok {
		msg := fmt.Sprintf("HSetNX reply to int failed,key:%v,field:%v", key, field)
		logger.Errorf(msg)
		err = errors.New(msg)
		return
	}
	return
}

func (p *DaoRedisEx) doMSet(cmd string, key string, value map[string]interface{}) (interface{}, error) {
	var args []interface{}
	if key != "" {
		key = p.getKey(key)
		args = append(args, key)
	}
	for k, v := range value {
		var data interface{}
		var errJSON error
		switch reflect.TypeOf(v).Kind() {
		case reflect.String:
			data = v
		default:
			data, errJSON = json.Marshal(v)
			if errJSON != nil {
				logger.Errorf("redis %s marshal data: %v to json:%s", cmd, v, errJSON.Error())
				return nil, errJSON
			}
		}
		if key == "" {
			args = append(args, p.getKey(k), data)
		} else {
			args = append(args, k, data)
		}
	}
	var reply interface{}
	var errDo error
	reply, errDo = p.do(cmd, args...)
	if errDo != nil {
		logger.Errorf("run redis command %s failed:error:%s,key:%s,value:%v", cmd, errDo.Error(), key, value)
		return nil, errDo
	}
	return reply, errDo
}

func (p *DaoRedisEx) doGet(cmd string, key string, value interface{}, fields ...string) (bool, error) {
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	for _, f := range fields {
		args = append(args, f)
	}
	result, errDo = p.do(cmd, args...)
	if errDo != nil {
		logger.Errorf("run redis %s command failed: error:%s,key:%s,fields:%v", cmd, errDo.Error(), key, fields)
		return false, errDo
	}
	if result == nil {
		value = nil
		return false, nil
	}
	if reflect.TypeOf(result).Kind() == reflect.Slice {
		byteResult := result.([]byte)
		strResult := string(byteResult)
		if strResult == "[]" {
			return true, nil
		}
	}
	errorJSON := json.Unmarshal(result.([]byte), value)
	if errorJSON != nil {
		if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
			strValue := string(result.([]byte))
			v := value.(*string)
			*v = strValue
			value = v
			return true, nil
		}
		logger.Errorf("get %s command result failed:%s", cmd, errorJSON.Error())
		return false, errorJSON
	}
	return true, nil
}

func (p *DaoRedisEx) doMGet(cmd string, args []interface{}, value interface{}) error {
	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		return fmt.Errorf(fmt.Sprintf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind()))
	}
	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()
	result, errDo := redis.ByteSlices(p.do(cmd, args...))
	if errDo != nil {
		logger.Errorf("run redis %s command failed: error:%s,args:%v", cmd, errDo.Error(), args)
		return errDo
	}
	if result == nil {
		return nil
	}
	if len(result) > 0 {
		for i := 0; i < len(result); i++ {
			r := result[i]
			if r != nil {
				item := reflect.New(refItem)
				errorJSON := json.Unmarshal(r, item.Interface())
				if errorJSON != nil {
					logger.Errorf("%s command result failed:%s", cmd, errorJSON.Error())
					return errorJSON
				}
				refSlice.Set(reflect.Append(refSlice, item.Elem()))
			} else {
				refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
			}
		}
	}
	return nil
}

func (p *DaoRedisEx) doMGetGo(keys []string, value interface{}) error {
	var (
		args     []interface{}
		keysMap  sync.Map
		keysLen  int
		rDo      interface{}
		errDo    error
		resultDo bool
		wg       sync.WaitGroup
	)
	keysLen = len(keys)
	if keysLen == 0 {
		return nil
	}
	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		return fmt.Errorf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind())
	}
	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()
	resultDo = true
	for _, v := range keys {
		args = append(args, p.getKey(v))
	}
	wg.Add(keysLen)
	for _, v := range args {
		go func(getK interface{}) {
			rDo, errDo = p.do("GET", getK)
			if errDo != nil {
				logger.Errorf("run redis GET command failed: error:%s,args:%v", errDo.Error(), getK)
				resultDo = false
			} else {
				keysMap.Store(getK, rDo)
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
	if !resultDo {
		return errors.New("doMGetGo one get error")
	}
	//整合结果
	for _, v := range args {
		r, ok := keysMap.Load(v)
		if ok && r != nil {
			item := reflect.New(refItem)
			errorJson := json.Unmarshal(r.([]byte), item.Interface())
			if errorJson != nil {
				logger.Errorf("GET command result failed:%s", errorJson.Error())
				return errorJson
			}
			refSlice.Set(reflect.Append(refSlice, item.Elem()))
		} else {
			refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
		}
	}
	return nil
}

func (p *DaoRedisEx) doMGetStringMap(cmd string, args ...interface{}) (err error, data map[string]string) {
	data, err = redis.StringMap(p.do(cmd, args...))
	if err != nil {
		logger.Errorf("run redis %s command failed: error:%v, args:%v", cmd, err, args)
		return err, nil
	}
	return
}

func (p *DaoRedisEx) doMGetIntMap(cmd string, args ...interface{}) (err error, data map[string]int) {
	data, err = redis.IntMap(p.do(cmd, args...))
	if err != nil {
		logger.Errorf("run redis %s command failed: error:%v, args:%v", cmd, err, args)
		return err, nil
	}
	return
}

func (p *DaoRedisEx) doIncr(cmd string, key string, value int, expire int, fields ...string) (num int64, err error) {
	var (
		data interface{}
		ok   bool
	)
	expire = p.getExpire(expire)
	key = p.getKey(key)
	if len(fields) == 0 {
		data, err = p.do(cmd, key, value)
	} else {
		field := fields[0]
		data, err = p.do(cmd, key, field, value)
	}
	if err != nil {
		logger.Errorf("run redis %s command failed: error:%s,key:%s,fields:%v,value:%d", cmd, err.Error(), key, fields, value)
		return
	}
	num, ok = data.(int64)
	if !ok {
		msg := fmt.Sprintf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))
		logger.Errorf(msg)
		err = errors.New(msg)
		return
	}
	if expire > 0 {
		_, errExpire := p.do("EXPIRE", key, expire)
		if errExpire != nil {
			logger.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
		}
	}
	return
}

func (p *DaoRedisEx) doIncrNX(cmd string, key string, value int, expire int) (num int64, err error) {
	var (
		data interface{}
		ok   bool
	)
	expire = p.getExpire(expire)
	key = p.getKey(key)
	redisClient, err := p.getRedisConn()
	if err != nil {
		return
	}
	defer redisClient.Close()
	defer p.resetTempExpireSecond()
	luaCmd := "local ck=redis.call('EXISTS', KEYS[1]); if (ck == 1) then return redis.call('INCRBY', KEYS[1], ARGV[1]) else return 'null' end"
	data, err = redisClient.Do("EVAL", luaCmd, 1, key, value)
	if err != nil {
		logger.Errorf("run redis %s command failed: error:%s,key:%s,value:%d", cmd, err.Error(), key, value)
		return
	}
	var luaRet string
	if luaRet, ok = data.(string); ok { // key 不存在
		if luaRet == "null" {
			logger.Errorf("INCRBY key not exists")
			return
		}
	}
	num, ok = data.(int64)
	if !ok {
		msg := fmt.Sprintf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))
		logger.Errorf(msg)
		err = errors.New(msg)
		return
	}
	if expire > 0 {
		_, errExpire := p.do("EXPIRE", key, expire)
		if errExpire != nil {
			logger.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
		}
	}
	return
}

func (p *DaoRedisEx) doDel(cmd string, data ...interface{}) error {
	_, errDo := p.do(cmd, data...)
	if errDo != nil {
		logger.Errorf("run redis %s command failed: error:%s,data:%v", cmd, errDo.Error(), data)
	}
	return errDo
}

/*基础结束*/
func (p *DaoRedisEx) Set(key string, value interface{}, ops ...OpOptionEx) (err error) {
	p.applyOpts(ops)
	_, err = p.doSet("SET", key, value, 0)
	return
}

// MSet mset
func (p *DaoRedisEx) MSet(datas map[string]interface{}) error {
	_, err := p.doMSet("MSET", "", datas)
	return err
}

// SetEx setex
func (p *DaoRedisEx) SetEx(key string, value interface{}, expire int) error {
	_, err := p.doSet("SET", key, value, expire)
	return err
}

// Expire expire
func (p *DaoRedisEx) Expire(key string, expire int) error {
	key = p.getKey(key)
	_, err := p.do("EXPIRE", key, expire)
	if err != nil {
		logger.Errorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", err.Error(), key, expire)
		return err
	}
	return nil
}

func (p *DaoRedisEx) Get(key string, data interface{}) error {
	_, err := p.doGet("GET", key, data)
	return err
}

// 返回 1. key是否存在 2. error
func (p *DaoRedisEx) GetRaw(key string, data interface{}) (bool, error) {
	return p.doGet("GET", key, data)
}

func (p *DaoRedisEx) MGet(keys []string, data interface{}) error {
	var args []interface{}
	for _, v := range keys {
		args = append(args, p.getKey(v))
	}
	err := p.doMGet("MGET", args, data)
	return err
}

// 封装mget通过go并发get
func (p *DaoRedisEx) MGetGo(keys []string, data interface{}) error {
	err := p.doMGetGo(keys, data)
	return err
}

func (p *DaoRedisEx) Incr(key string, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("INCRBY", key, 1, 0)
}

func (p *DaoRedisEx) IncrBy(key string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("INCRBY", key, value, 0)
}

// 存在key 才会自增
func (p *DaoRedisEx) IncrNX(key string, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncrNX("INCRBY", key, 1, 0)
}

// 存在key 才会更新数值
func (p *DaoRedisEx) IncrByNX(key string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncrNX("INCRBY", key, value, 0)
}

func (p *DaoRedisEx) SetEXNX(key string, value interface{}) (string, error) {
	redisClient, err := p.getRedisConn()
	if err != nil {
		return "", err
	}
	defer redisClient.Close()
	key = p.getKey(key)
	reply, err := redis.String(redisClient.Do("SET", key, value, "EX", p.ExpireSecond, "NX"))
	if err == redis.ErrNil {
		err = nil
	}
	return reply, err
}

func (p *DaoRedisEx) SetNX(key string, value interface{}, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doSetNX("SETNX", key, value, 0)
}

func (p *DaoRedisEx) SetNXNoExpire(key string, value interface{}) (int64, error) {
	return p.doSetNX("SETNX", key, value, -1)
}

func (p *DaoRedisEx) Del(key string) error {
	key = p.getKey(key)
	err := p.doDel("DEL", key)
	return err
}

func (p *DaoRedisEx) MDel(key ...string) error {
	var keys []interface{}
	for _, v := range key {
		keys = append(keys, p.getKey(v))
	}
	err := p.doDel("DEL", keys...)
	return err
}

func (p *DaoRedisEx) Exists(key string) (bool, error) {
	key = p.getKey(key)
	data, err := p.do("EXISTS", key)
	if err != nil {
		logger.Errorf("run redis EXISTS command failed: error:%s,key:%s", err.Error(), key)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get EXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		logger.Errorf(err.Error())
		return false, err
	}
	if count == 1 {
		return true, nil
	}

	return false, nil
}

// hash start
func (p *DaoRedisEx) HIncrby(key string, field string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("HINCRBY", key, value, 0, field)
}

func (p *DaoRedisEx) HGet(key string, field string, value interface{}) error {
	_, err := p.doGet("HGET", key, value, field)
	return err
}

// HGetRaw 返回 1. key是否存在 2. error
func (p *DaoRedisEx) HGetRaw(key string, field string, value interface{}) (bool, error) {
	return p.doGet("HGET", key, value, field)
}

func (p *DaoRedisEx) HMGet(key string, fields []interface{}, data interface{}) error {
	var args []interface{}
	args = append(args, p.getKey(key))
	for _, v := range fields {
		args = append(args, v)
	}
	err := p.doMGet("HMGET", args, data)
	return err
}

func (p *DaoRedisEx) HSet(key string, field string, value interface{}, ops ...OpOptionEx) error {
	p.applyOpts(ops)
	_, err := p.doSet("HSET", key, value, 0, field)
	return err
}

func (p *DaoRedisEx) HSetNX(key string, field string, value interface{}, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doSetNX("HSETNX", key, value, 0, field)
}

// HMSet value是filed:data
func (p *DaoRedisEx) HMSet(key string, value map[string]interface{}) error {
	_, err := p.doMSet("HMSet", key, value)
	return err
}

func (p *DaoRedisEx) HLen(key string, data *int) error {
	key = p.getKey(key)
	resultData, err := p.do("HLEN", key)
	if err != nil {
		logger.Errorf("run redis HLEN command failed: error:%s,key:%s", err.Error(), key)
		return err
	}
	length, b := resultData.(int64)
	if !b {
		msg := fmt.Sprintf("redis data convert to int64 failed:%v", resultData)
		logger.Errorf(msg)
		err = errors.New(msg)
		return err
	}
	*data = int(length)
	return nil
}

func (p *DaoRedisEx) HDel(key string, data ...interface{}) error {
	var args []interface{}
	key = p.getKey(key)
	args = append(args, key)
	for _, item := range data {
		args = append(args, item)
	}
	err := p.doDel("HDEL", args...)
	if err != nil {
		logger.Errorf("run redis HDEL command failed: error:%s,key:%s,data:%v", err.Error(), key, data)
	}
	return err
}

func (p *DaoRedisEx) HExists(key string, field string) (bool, error) {
	key = p.getKey(key)
	data, err := p.do("HEXISTS", key, field)
	if err != nil {
		logger.Errorf("run redis HEXISTS command failed: error:%s,key:%s", err.Error(), key)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get HEXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		logger.Errorf(err.Error())
		return false, err
	}
	if count == 1 {
		return true, nil
	}
	return false, nil
}

// hash end

// sorted set start
func (p *DaoRedisEx) ZAdd(key string, score interface{}, data interface{}) error {
	key = p.getKey(key)
	_, errDo := p.do("ZADD", key, score, data)
	if errDo != nil {
		logger.Errorf("run redis ZADD command failed: error:%s,key:%s,score:%d,data:%v", errDo.Error(), key, score, data)
	}
	return errDo
}

func (p *DaoRedisEx) ZCard(key string) (data int, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("ZCARD", key)
	if err != nil {
		logger.Errorf("run redis ZCARD command failed: error:%v,key:%s", err, key)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCard get replay is not int64:%v", reply))
		return
	}
}

func (p *DaoRedisEx) ZCount(key string, min, max int) (data int, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("ZCOUNT", key, min, max)
	if err != nil {
		logger.Errorf("run redis ZCOUNT command failed: error:%v,key:%s,min:%d,max:%d", err, key, min, max)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCount get replay is not int64:%v", reply))
		return
	}
}

func (p *DaoRedisEx) ZIncrBy(key string, increment int, member interface{}) error {
	key = p.getKey(key)
	_, errDo := p.do("ZINCRBY", key, increment, member)
	if errDo != nil {
		logger.Errorf("run redis ZINCRBY command failed: error:%s,key:%s,increment:%d,data:%v", errDo.Error(), key, increment, member)
	}
	return errDo
}

// sorted set start
func (p *DaoRedisEx) ZAddM(key string, value map[string]interface{}) error {
	_, err := p.doMSet("ZADD", key, value)
	return err
}

func (p *DaoRedisEx) ZGetByScore(key string, sort bool, start int, end int, value interface{}) error {
	var cmd string
	if sort {
		cmd = "ZRANGEBYSCORE"
	} else {
		cmd = "ZREVRANGEBYSCORE"
	}
	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)
	err := p.doMGet(cmd, args, value)
	return err
}

func (p *DaoRedisEx) ZGet(key string, sort bool, start int, end int, value interface{}) error {
	var cmd string
	if sort {
		cmd = "ZRANGE"
	} else {
		cmd = "ZREVRANGE"
	}
	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)
	err := p.doMGet(cmd, args, value)
	return err
}

func (p *DaoRedisEx) ZGetWithScores(key string, sort bool, start int, end int) (err error, data map[string]string) {
	var cmd string
	if sort {
		cmd = "ZRANGE"
	} else {
		cmd = "ZREVRANGE"
	}
	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)
	args = append(args, "WITHSCORES")
	err, data = p.doMGetStringMap(cmd, args...)
	return
}

func (p *DaoRedisEx) ZRank(key string, member string, sort bool) (error, int) {
	var cmd string
	if sort {
		cmd = "ZRANK"
	} else {
		cmd = "ZREVRANK"
	}
	key = p.getKey(key)
	result, errDo := p.do(cmd, key, member)
	if errDo != nil {
		logger.Errorf("run redis %s command failed: error:%s,key:%s,increment:%d,data:%v", cmd, errDo.Error(), key, member)
		return errDo, 0
	}
	if v, ok := result.(int64); ok {
		return nil, int(v)
	} else {
		msg := fmt.Sprintf("run redis %s command result failed: key:%v,result:%v", cmd, key, result)
		logger.Errorf(msg)
		err := errors.New(msg)
		return err, 0
	}
}

func (p *DaoRedisEx) ZScore(key string, member string, value interface{}) error {
	cmd := "ZSCORE"
	_, err := p.doGet(cmd, key, value, member)
	return err
}

func (p *DaoRedisEx) ZRevRange(key string, start int, end int, value interface{}) error {
	return p.ZGet(key, false, start, end, value)
}

func (p *DaoRedisEx) ZRem(key string, data ...interface{}) error {
	var args []interface{}
	key = p.getKey(key)
	args = append(args, key)
	for _, item := range data {
		args = append(args, item)
	}
	err := p.doDel("ZREM", args...)
	return err
}

//list start

func (p *DaoRedisEx) LRange(start int, end int, value interface{}) (err error) {
	key := ""
	key = p.getKey(key)
	var args []interface{}
	args = append(args, key)
	args = append(args, start)
	args = append(args, end)
	err = p.doMGet("LRANGE", args, value)
	return
}

func (p *DaoRedisEx) LLen() (int64, error) {
	cmd := "LLEN"
	key := ""
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	result, errDo = p.do(cmd, key)
	if errDo != nil {
		logger.Errorf("run redis %s command failed: error:%s,key:%s", cmd, errDo.Error(), key)
		return 0, errDo
	}
	if result == nil {
		return 0, nil
	}
	num, ok := result.(int64)
	if !ok {
		return 0, errors.New("result to int64 failed")
	}
	return num, nil
}

func (p *DaoRedisEx) LREM(count int, data interface{}) (error, int) {
	key := ""
	key = p.getKey(key)
	result, errDo := p.do("LREM", key, count, data)
	if errDo != nil {
		logger.Errorf("run redis command LREM failed: error:%s,key:%s,count:%d,data:%v", errDo.Error(), key, count, data)
		return errDo, 0
	}
	countRem, ok := result.(int)
	if !ok {
		msg := fmt.Sprintf("redis data convert to int failed:%v", result)
		logger.Errorf(msg)
		err := errors.New(msg)
		return err, 0
	}
	return nil, countRem
}

func (p *DaoRedisEx) LTRIM(start int, end int) (err error) {
	key := ""
	key = p.getKey(key)
	_, err = p.do("LTRIM", key, start, end)
	if err != nil {
		logger.Errorf("run redis command LTRIM failed: error:%v,key:%s,start:%d,end:%d", err, key, start, end)
		return
	}
	return
}

func (p *DaoRedisEx) Scan(pattern string) ([]string, error) {
	iter := 0
	var keys []string
	for {
		arr, err := redis.Values(p.do("SCAN", iter, "MATCH", pattern))
		if err != nil {
			return keys, fmt.Errorf("error retrieving '%s' keys err: %s", pattern, err)
		}
		if len(arr) != 2 {
			return keys, fmt.Errorf("invalid response from SCAN for pattern: %s", pattern)
		}
		k, _ := redis.Strings(arr[1], nil)
		keys = append(keys, k...)
		if iter, _ = redis.Int(arr[0], nil); iter == 0 {
			break
		}
	}
	return keys, nil
}

func (p *DaoRedisEx) RPush(value interface{}) error {
	return p.Push(value, false)
}

func (p *DaoRedisEx) LPush(value interface{}) error {
	return p.Push(value, true)
}

func (p *DaoRedisEx) Push(value interface{}, isLeft bool) error {
	var cmd string
	if isLeft {
		cmd = "LPUSH"
	} else {
		cmd = "RPUSH"
	}
	key := ""
	_, err := p.doSet(cmd, key, value, -1)
	return err
}

func (p *DaoRedisEx) RPop(value interface{}) error {
	return p.Pop(value, false)
}

func (p *DaoRedisEx) LPop(value interface{}) error {
	return p.Pop(value, true)
}

func (p *DaoRedisEx) BLpop(value interface{}, timeout int) error {
	key := p.getKey("")
	var result interface{}
	var errDo error
	result, errDo = p.do("BLPOP", key, timeout)
	if errDo != nil {
		//logger.LogErrorf(logger.LogNameRedis,"run redis BLPOP command failed: error:%s,key:%s", errDo.Error(), key)
		return errDo
	}
	if result == nil {
		value = nil
		return errDo
	}
	results, err := redis.ByteSlices(result, errDo)
	if err != nil {
		return err
		//logger.LogErrorf(logger.LogNameRedis,"get BLPOP command redis.ByteSlices failed:%s", err.Error())
	}
	if len(results) == 2 {
		errorJSON := json.Unmarshal(results[1], value)
		if errorJSON != nil {
			if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
				strValue := string(result.([]byte))
				v := value.(*string)
				*v = strValue
				value = v
				return nil
			}
			return errorJSON
		}
	} else {
		value = nil
		return errDo
	}
	return errDo
}

func (p *DaoRedisEx) BRpop(value interface{}, timeout int) error {
	key := p.getKey("")
	var result interface{}
	var errDo error
	result, errDo = p.do("BRPOP", key, timeout)
	if errDo != nil {
		//logger.LogErrorf(logger.LogNameRedis,"run redis BLPOP command failed: error:%s,key:%s", errDo.Error(), key)
		return errDo
	}
	if result == nil {
		value = nil
		return errDo
	}
	results, err := redis.ByteSlices(result, errDo)
	if err != nil {
		return err
		//logger.LogErrorf(logger.LogNameRedis,"get BLPOP command redis.ByteSlices failed:%s", err.Error())
	}
	if len(results) == 2 {
		errorJSON := json.Unmarshal(results[1], value)
		if errorJSON != nil {
			if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
				strValue := string(result.([]byte))
				v := value.(*string)
				*v = strValue
				value = v
				return nil
			}
			return errorJSON
		}
	} else {
		value = nil
		return errDo
	}
	return errDo
}

func (p *DaoRedisEx) Pop(value interface{}, isLeft bool) error {
	var cmd string
	if isLeft {
		cmd = "LPOP"
	} else {
		cmd = "RPOP"
	}
	key := ""
	_, err := p.doGet(cmd, key, value)
	return err
}

//list end

// Set集合Start
func (p *DaoRedisEx) SAdd(key string, argPs []interface{}) error {
	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)
	_, errDo := p.do("SADD", args...)
	if errDo != nil {
		logger.Errorf("run redis SADD command failed: error:%s,key:%s,args:%v", errDo.Error(), key, args)
	}
	return errDo
}

func (p *DaoRedisEx) SIsMember(key string, arg interface{}) (b bool, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("SISMEMBER", key, arg)
	if err != nil {
		logger.Errorf("run redis SISMEMBER command failed: error:%v,key:%s,member:%s", err, key, arg)
		return
	}
	if code, ok := reply.(int64); ok && code == int64(1) {
		b = true
	}
	return
}

func (p *DaoRedisEx) SCard(key string) int64 {
	redisClient, err := p.getRedisConn()
	if err != nil {
		return 0
	}
	defer redisClient.Close()
	key = p.getKey(key)
	reply, errDo := redisClient.Do("SCARD", key)
	if errDo != nil {
		logger.Errorf("SCARD run redis SCARD command error", errDo)
		return 0
	}
	return reply.(int64)
}

func (p *DaoRedisEx) SRem(key string, argPs []interface{}) error {
	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)
	_, errDo := p.do("SREM", args...)
	if errDo != nil {
		logger.Errorf("run redis SREM command failed: error:%s,key:%s,member:%s", errDo.Error(), key, args)
	}
	return errDo
}

func (p *DaoRedisEx) SPop(key string, value interface{}) error {
	_, err := p.doGet("SPOP", key, value)
	return err
}

func (p *DaoRedisEx) SMembers(key string, value interface{}) (err error) {
	var args []interface{}
	args = append(args, p.getKey(key))
	err = p.doMGet("SMEMBERS", args, value)
	return
}

func (p *DaoRedisEx) HGetAll(key string, data interface{}) error {
	var args []interface{}

	args = append(args, p.getKey(key))

	err := p.doMGet("HGETALL", args, data)

	return err
}

func (p *DaoRedisEx) HGetAllStringMap(key string) (err error, data map[string]string) {
	args := p.getKey(key)
	return p.doMGetStringMap("HGETALL", args)
}

func (p *DaoRedisEx) HGetAllIntMap(key string) (err error, data map[string]int) {
	args := p.getKey(key)
	return p.doMGetIntMap("HGETALL", args)
}

// GetPTtl ：获取key的过期时间，单位为毫秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedisEx) GetPTtl(key string) (ttl int64, err error) {
	return p.doGetTTL("PTTL", key)
}

// GetTTL ：获取key的过期时间，单位为秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedisEx) GetTTL(key string) (ttl int64, err error) {
	return p.doGetTTL("TTL", key)
}

func (p *DaoRedisEx) XAdd(uniqueID string) (err error) {
	key := fmt.Sprintf("watch_config:%s", p.getKey(""))
	_, err = p.do("xadd", key, "MAXLEN", "~", 3000, "*", "configID", fmt.Sprintf("%s:%s", p.getKey(""), uniqueID))
	//fmt.Println("xadd", key, "MAXLEN", "~", 3000, "*", "configID", fmt.Sprintf("%s:%s", p.getKey(""), uniqueID))
	return
}

func (p *DaoRedisEx) doGetTTL(cmd string, key string) (ttl int64, err error) {
	args := p.getKey(key)
	ttl, err = redis.Int64(p.do(cmd, args))
	if err != nil {
		logger.Errorf("doGetTtl run redis command error", err)
		return 0, err
	}
	return
}
