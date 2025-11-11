package connectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorm/caches/v4"
	mapstructure "github.com/mitchellh/mapstructure"
	"github.com/rapidaai/pkg/ciphers"
	commons "github.com/rapidaai/pkg/commons"
	configs "github.com/rapidaai/pkg/configs"
	"github.com/redis/go-redis/v9"
)

type RedisConnector interface {
	Connector
	// single command with fixed arg size
	Cmd(context.Context, string, []string) *RedisResponse
	// multi key commands
	Cmds(ctx context.Context, cmd string, args *[]string) *RedisResponse
}

type RedisPostgresCacheConnector interface {
	Connector
	caches.Cacher
}

type redisConnector struct {
	cfg        *configs.RedisConfig
	Connection *redis.Client
	logger     commons.Logger
}

type redisPostgresCacheConnector struct {
	redisConnector
	DefualtCacheExpire uint32
	Prefix             string
}

// return connector behaviour with redis
func NewRedisConnector(config *configs.RedisConfig, logger commons.Logger) RedisConnector {
	return &redisConnector{cfg: config, logger: logger}
}

func NewRedisPostgresCacheConnector(config *configs.RedisConfig, logger commons.Logger) RedisPostgresCacheConnector {
	return &redisPostgresCacheConnector{
		redisConnector:     redisConnector{cfg: config, logger: logger},
		DefualtCacheExpire: 500,
		Prefix:             "PSQL::GORM::",
	}
}

// generate connection string from configuration
func (redisC *redisConnector) connectionString() string {
	return fmt.Sprintf("%s:%d", redisC.cfg.Host, redisC.cfg.Port)
}

// provide a debug name for connector
func (redisC *redisConnector) Name() string {
	return fmt.Sprintf("REDIS %s:%d", redisC.cfg.Host, redisC.cfg.Port)
}

// only connect the call usually made by main.go to create a connection with given configuration
// anyway can be called anywhere as config is will always be in socpe of connect
func (redisC *redisConnector) Connect(ctx context.Context) error {
	opt := &redis.Options{
		Addr:     redisC.connectionString(),
		PoolSize: redisC.cfg.MaxConnection,
		Username: redisC.cfg.Auth.User,
		Password: redisC.cfg.Auth.Password,
		DB:       redisC.cfg.Db,
	}
	if redisC.cfg.InsecureSkipVerify {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: redisC.cfg.InsecureSkipVerify,
		}
	}
	client := redis.NewClient(opt)

	redisC.Connection = client
	redisC.logger.Debugf("Created new client for redis with name: %s", redisC.Name())

	if ok := redisC.IsConnected(ctx); !ok {
		redisC.logger.Errorf("could not connect to redis client")
	}
	return nil
}

// getting connection to use if anyone wants to use the connection
func (redisC *redisConnector) GetConnection() *redis.Client {
	return redisC.Connection
}

// Return boolean status if connected or not
func (redisC *redisConnector) IsConnected(ctx context.Context) bool {

	redisC.logger.Debug("Pinging redis server.")
	pingResponse, err := redisC.Connection.Ping(ctx).Result()
	if err != nil {
		redisC.logger.Errorf("Error while pinging redis server. %v", err)
		return false
	}
	redisC.logger.Debugf("Return from ping command %v", pingResponse)
	return true
}

// Command executor works only for single command with given ptr arguments string
// May return the empty response if the key is not found in redis
func (redisC *redisConnector) Cmd(ctx context.Context, cmd string, args []string) *RedisResponse {
	start := time.Now()
	new := make([]interface{}, len(args)+1)
	new[0] = cmd
	for i, arg := range args {
		new[i+1] = arg
	}
	exCmd := redisC.Connection.Do(ctx, new...)
	val, err := exCmd.Result()
	if err != nil {
		redisC.logger.Errorf("error while executing cmd from redis cmd %v err %v", cmd, err)
		return &RedisResponse{Err: err}
	}
	redisC.logger.Benchmark("redisConnector.Cmd", time.Since(start))
	return &RedisResponse{Result: val, Err: err}
}

// For executing single command with multiple different arguments.
// Pipelined command without TNX as currenty used for only read, multiple write and read may create inconsistent results as its not in TX
func (redisC *redisConnector) Cmds(ctx context.Context, cmd string, args *[]string) *RedisResponse {

	start := time.Now()

	// pipeline start
	redisC.logger.Debugf("started executing redis cmds %s in pipeline no of commands %d", cmd, len(*args))
	pipe := redisC.Connection.Pipeline()
	for _, arg := range *args {
		pipe.Do(ctx, cmd, arg)
	}
	// pipeline end
	exCmds, err := pipe.Exec(ctx)
	if err != nil {
		redisC.logger.Errorf("Error while executing cmds from redis cmd %v args %v err %v", cmd, args, err)
		return &RedisResponse{Err: err}
	}
	redisC.logger.Debugf("ending redis cmds executing in pipeline with result count %d", len(exCmds))

	// result preparation for pipeline result
	result := make([]interface{}, 0, len(*args))
	for _, cmd := range exCmds {
		res, ok := cmd.(*redis.Cmd)
		if ok {
			val, err := res.Result()
			if err != nil {
				redisC.logger.Errorf("Error while executing in pipeline cmd : %v", err)
				return &RedisResponse{Err: err}
			}
			result = append(result, val)
		}

	}
	redisC.logger.Benchmark("redisConnector.Cmds", time.Since(start))
	return &RedisResponse{Result: result, Err: err}
}

// closing connection
func (redisC *redisConnector) Disconnect(ctx context.Context) error {
	redisC.logger.Debug("Disconnecting with redis client.")

	err := redisC.Connection.Close()
	if err != nil {
		redisC.logger.Errorf("Failed to disconnect redis client. %v", err)
		return err
	}
	redisC.logger.Debug("Disconnected successful redis client.")
	// anyway nil the connection reference
	redisC.Connection = nil
	return err

}

type RedisResponse struct {
	Result interface{}
	Err    error
}

func (rs *RedisResponse) Error() error {
	return rs.Err
}

func (rs *RedisResponse) HasError() bool {
	return rs.Err != nil
}

func (rs *RedisResponse) ResultSlice() ([]interface{}, error) {
	if rs.Error() != nil {
		return nil, rs.Err
	}
	switch rs.Result.(type) {
	case []interface{}:
		return rs.Result.([]interface{}), nil
	default:
		return nil, fmt.Errorf("unable to parse the result to []interface{}")
	}

}

func (rs *RedisResponse) ResultStringSlice() ([]string, error) {
	if rs.Error() != nil {
		return nil, rs.Err
	}
	switch rs.Result.(type) {
	case []interface{}:
		cVal := rs.Result.([]interface{})
		var val []string
		for _, itf := range cVal {
			vl, ok := itf.(string)
			if !ok {
				return nil, fmt.Errorf("unable to parse the result to [][]interface{} err %T", itf)
			}
			val = append(val, vl)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("unable to parse the result to []interface{}")
	}

}

func (rs *RedisResponse) ResultStruct(output interface{}) error {
	if rs.Error() != nil {
		return rs.Err
	}
	switch rs.Result.(type) {
	case string:
		fmt.Println("trying to deserialize from string")
		return json.Unmarshal([]byte(rs.Result.(string)), &output)
	case interface{}:
		sgl, ok := rs.Result.([]interface{})
		if !ok {
			return fmt.Errorf("unable to parse the result to []interface{}")
		}
		n := len(sgl)
		if n == 0 {
			return fmt.Errorf("parsing empty results ignore steps. size : %d", n)
		}
		val := make(map[string]interface{}, n/2)
		for i := 0; i < n; i += 2 {
			key, ok := sgl[i].(string)
			if !ok {
				return fmt.Errorf("illegal key defnition unable to cast to string")
			}
			value := sgl[i+1]
			val[key] = value
		}
		err := rs.Cast(val, output)
		return err
	default:
		return fmt.Errorf("unable to parse the result to []interface{}")
	}
}

// Converting/decoding redis result raw interfaces to the target pointer
// multi cmd results to ptr of struct slice
func (rs *RedisResponse) ResultStructs(output interface{}) error {
	if rs.Error() != nil {
		return rs.Err
	}
	switch rs.Result.(type) {
	case []interface{}:
		sgl, ok := rs.Result.([]interface{})
		if !ok {
			return fmt.Errorf("unable to parse the result to []interface{}")
		}
		if len(sgl) == 0 {
			return fmt.Errorf("parsing empty results ignore steps. size : %d", len(sgl))
		}
		outVal := make([]map[string]interface{}, 0, len(sgl))
		for _, itf := range sgl {
			cVal, ok := itf.([]interface{})
			if !ok {
				return fmt.Errorf("unable to parse the result to []interface{}")
			}
			n := len(cVal)
			if n == 0 {
				continue
			}
			val := make(map[string]interface{}, n/2)
			for i := 0; i < n; i += 2 {
				key, ok := cVal[i].(string)
				if !ok {
					return fmt.Errorf("illegal key defnition unable to cast to string")
				}
				value := cVal[i+1]
				val[key] = value
			}
			outVal = append(outVal, val)
		}
		err := rs.Cast(outVal, output)
		return err
	default:
		return fmt.Errorf("unable to parse the result to []interface{}")
	}
}

func (rs *RedisResponse) Cast(input, output interface{}) error {
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &output,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// returning redis result to string slicess used for multi cmds which return more then one console result.
func (rs *RedisResponse) ResultStringSlices() ([][]string, error) {
	if rs.Error() != nil {
		return nil, rs.Err
	}

	// type checking and conversion 2D slice
	switch rs.Result.(type) {
	case []interface{}:
		sgl, ok := rs.Result.([]interface{})
		if !ok {
			return nil, fmt.Errorf("unable to parse the result to []interface{}")
		}
		var val [][]string
		for _, itf := range sgl {
			cVal := []string{}
			vl, ok := itf.([]interface{})
			if !ok {
				return nil, fmt.Errorf("unable to parse the result to [][]interface{} err : %v", ok)
			}
			for _, ct := range vl {
				cVal = append(cVal, ct.(string))
			}
			val = append(val, cVal)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("unable to parse the result to []interface{}")
	}

}

// reference https://pkg.go.dev/github.com/mitchellh/mapstructure@v1.5.0?utm_source=gopls
// setting up mapstruct to decode redis returned result to struct

func (redisC *redisPostgresCacheConnector) Get(ctx context.Context, key string, q *caches.Query[any]) (*caches.Query[any], error) {
	res, err := redisC.Connection.Get(ctx, fmt.Sprintf("%s::%s", redisC.Prefix, ciphers.Hash(key))).Result()
	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	if err := q.Unmarshal([]byte(res)); err != nil {
		return nil, err
	}
	return q, nil
}

func (redisC *redisPostgresCacheConnector) Store(ctx context.Context, key string, val *caches.Query[any]) error {
	res, err := val.Marshal()
	if err != nil {
		return err
	}
	redisC.Connection.Set(ctx, fmt.Sprintf("%s::%s", redisC.Prefix, ciphers.Hash(key)), res, time.Duration(redisC.DefualtCacheExpire)*time.Second)
	return nil
}

func (redisC *redisPostgresCacheConnector) Invalidate(ctx context.Context) error {
	const batchSize = 1000
	var cursor uint64

	for {
		keys, nextCursor, err := redisC.Connection.Scan(ctx, cursor, fmt.Sprintf("%s*", redisC.Prefix), batchSize).Result()
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}

		if len(keys) > 0 {
			// Process keys in smaller batches
			for i := 0; i < len(keys); i += batchSize {
				end := i + batchSize
				if end > len(keys) {
					end = len(keys)
				}
				batch := keys[i:end]

				// Use UNLINK instead of DEL for better performance
				err := redisC.Connection.Unlink(ctx, batch...).Err()
				if err != nil {
					if strings.Contains(err.Error(), "CROSSSLOT") {
						for _, key := range batch {
							if err := redisC.Connection.Unlink(ctx, key).Err(); err != nil {
								return fmt.Errorf("individual unlink error: %w", err)
							}
						}
					} else {
						return fmt.Errorf("batch unlink error: %w", err)
					}
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
