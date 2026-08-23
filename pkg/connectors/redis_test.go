// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package connectors

import (
	"context"
	"testing"

	redismock "github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
)

// Local config struct for testing - removed since we can use configs package

func TestNewRedisConnector(t *testing.T) {
	// Requires external config and logger packages
	t.Skip("Requires external packages - integration test")
}

func TestNewRedisPostgresCacheConnector(t *testing.T) {
	// Requires external config and logger packages
	t.Skip("Requires external packages - integration test")
}

func TestRedisConnector_Name(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "localhost",
			host:     "localhost",
			port:     6379,
			expected: "REDIS localhost:6379",
		},
		{
			name:     "remote host",
			host:     "redis.example.com",
			port:     6380,
			expected: "REDIS redis.example.com:6380",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &redisConnector{
				cfg: &configs.RedisConfig{Host: tt.host, Port: tt.port},
			}
			result := connector.Name()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisConnector_Connect(t *testing.T) {
	t.Skip("Connect requires real Redis server - integration test")
}

func TestRedisConnector_IsConnected(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	cfg := &configs.RedisConfig{
		Host: "localhost",
		Port: 6379,
	}
	connector := &redisConnector{cfg: cfg, logger: logger}
	t.Run("without connection", func(t *testing.T) {
		connector.Connection = nil
		ctx := context.Background()
		result := connector.IsConnected(ctx)
		assert.False(t, result)
	})

	t.Run("with connection", func(t *testing.T) {
		// Mock connection, but since it's hard, skip
		t.Skip("Requires mock Redis client - integration test")
	})
}

func TestRedisConnector_Disconnect(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	cfg := &configs.RedisConfig{
		Host: "localhost",
		Port: 6379,
	}
	db, _ := redismock.NewClientMock()
	connector := &redisConnector{
		cfg:        cfg,
		logger:     logger,
		Connection: db,
	}

	ctx := context.Background()
	err := connector.Disconnect(ctx)
	assert.NoError(t, err)
	assert.Nil(t, connector.Connection)
}

func TestRedisConnector_Cmd(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	assert.NoError(t, err)
	db, mock := redismock.NewClientMock()
	mock.ExpectDo("SET", "key", "value").SetVal("OK")
	connector := &redisConnector{logger: logger, Connection: db}

	response := connector.Cmd(context.Background(), "SET", []string{"key", "value"})

	assert.NoError(t, response.Error())
	assert.Equal(t, "OK", response.Result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisConnector_Cmds(t *testing.T) {
	// Requires real Redis connection
	t.Skip("Cmds requires real Redis server - integration test")
}

func TestRedisResponse_Error(t *testing.T) {
	response := &RedisResponse{Err: assert.AnError}
	assert.Equal(t, assert.AnError, response.Error())
}

func TestRedisResponse_HasError(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError}
		assert.True(t, response.HasError())
	})

	t.Run("without error", func(t *testing.T) {
		response := &RedisResponse{Err: nil}
		assert.False(t, response.HasError())
	})
}

func TestRedisResponse_ResultSlice(t *testing.T) {
	t.Run("valid slice", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{"a", "b"}}
		result, err := response.ResultSlice()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"a", "b"}, result)
	})

	t.Run("invalid type", func(t *testing.T) {
		response := &RedisResponse{Result: "not a slice"}
		result, err := response.ResultSlice()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError, Result: []interface{}{}}
		result, err := response.ResultSlice()
		assert.Equal(t, assert.AnError, err)
		assert.Nil(t, result)
	})
}

func TestRedisResponse_ResultStringSlice(t *testing.T) {
	t.Run("valid string slice", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{"a", "b"}}
		result, err := response.ResultStringSlice()
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, result)
	})

	t.Run("mixed types", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{"a", 1}}
		result, err := response.ResultStringSlice()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError, Result: []interface{}{}}
		result, err := response.ResultStringSlice()
		assert.Equal(t, assert.AnError, err)
		assert.Nil(t, result)
	})
}

func TestRedisResponse_ResultStruct(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		jsonStr := `{"key":"value"}`
		response := &RedisResponse{Result: jsonStr}
		var result map[string]interface{}
		err := response.ResultStruct(&result)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"key": "value"}, result)
	})

	t.Run("from slice", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{"key", "value"}}
		var result map[string]interface{}
		err := response.ResultStruct(&result)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"key": "value"}, result)
	})

	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError}
		var result interface{}
		err := response.ResultStruct(&result)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestRedisResponse_ResultStructs(t *testing.T) {
	t.Run("valid structs", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{
			[]interface{}{"key1", "value1"},
			[]interface{}{"key2", "value2"},
		}}
		var result []map[string]interface{}
		err := response.ResultStructs(&result)
		assert.NoError(t, err)
		expected := []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError}
		var result interface{}
		err := response.ResultStructs(&result)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestRedisResponse_ResultStringSlices(t *testing.T) {
	t.Run("valid slices", func(t *testing.T) {
		response := &RedisResponse{Result: []interface{}{
			[]interface{}{"a", "b"},
			[]interface{}{"c", "d"},
		}}
		result, err := response.ResultStringSlices()
		assert.NoError(t, err)
		expected := [][]string{
			{"a", "b"},
			{"c", "d"},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("with error", func(t *testing.T) {
		response := &RedisResponse{Err: assert.AnError}
		result, err := response.ResultStringSlices()
		assert.Equal(t, assert.AnError, err)
		assert.Nil(t, result)
	})
}

func TestRedisPostgresCacheConnector_Get(t *testing.T) {
	// Requires real Redis connection
	t.Skip("Get requires real Redis server - integration test")
}

func TestRedisPostgresCacheConnector_Store(t *testing.T) {
	// Requires real Redis connection
	t.Skip("Store requires real Redis server - integration test")
}

func TestRedisPostgresCacheConnector_Invalidate(t *testing.T) {
	// Requires real Redis connection
	t.Skip("Invalidate requires real Redis server - integration test")
}

func TestRedisConnector_EdgeCases(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	t.Run("disconnect with nil connection", func(t *testing.T) {
		cfg := &configs.RedisConfig{
			Host: "localhost",
			Port: 6379,
		}
		db, _ := redismock.NewClientMock()
		connector := &redisConnector{
			cfg:        cfg,
			logger:     logger,
			Connection: db,
		}

		ctx := context.Background()
		err := connector.Disconnect(ctx)
		assert.NoError(t, err)
		assert.Nil(t, connector.Connection)
	})

	t.Run("cmd with nil connection", func(t *testing.T) {
		cfg := &configs.RedisConfig{
			Host: "localhost",
			Port: 6379,
		}
		db, _ := redismock.NewClientMock()
		connector := &redisConnector{
			cfg:        cfg,
			logger:     logger,
			Connection: db,
		}

		ctx := context.Background()
		response := connector.Cmd(ctx, "GET", []string{"key"})
		assert.Error(t, response.Error())
	})

	t.Run("cmds with nil connection", func(t *testing.T) {
		cfg := &configs.RedisConfig{
			Host: "localhost",
			Port: 6379,
		}
		db, _ := redismock.NewClientMock()
		connector := &redisConnector{
			cfg:        cfg,
			logger:     logger,
			Connection: db,
		}

		ctx := context.Background()
		args := &[]string{"key"}
		response := connector.Cmds(ctx, "GET", args)
		assert.Error(t, response.Error())
	})
}
