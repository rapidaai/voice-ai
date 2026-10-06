//go:build cgo

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForPaginated(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return db
}

type TestModel struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func TestNewPaginated(t *testing.T) {
	db := setupTestDBForPaginated(t)
	var count int64
	p := NewPaginated(1, 10, &count, db)

	assert.Equal(t, uint32(1), p.Page)
	assert.Equal(t, uint32(10), p.PageSize)
	assert.Equal(t, &count, p.Count)
	assert.Equal(t, db, p.DB)
}

func TestPaginate(t *testing.T) {
	db := setupTestDBForPaginated(t)
	err := db.AutoMigrate(&TestModel{})
	assert.NoError(t, err)

	// Insert test data
	for i := 1; i <= 25; i++ {
		db.Create(&TestModel{Name: fmt.Sprintf("item%d", i)})
	}

	tests := []struct {
		name        string
		page        uint32
		pageSize    uint32
		expectedLen int
	}{
		{
			name:        "page 1, size 10",
			page:        1,
			pageSize:    10,
			expectedLen: 10,
		},
		{
			name:        "page 2, size 10",
			page:        2,
			pageSize:    10,
			expectedLen: 10,
		},
		{
			name:        "page 3, size 10",
			page:        3,
			pageSize:    10,
			expectedLen: 5,
		},
		{
			name:        "page 0, size 10",
			page:        0,
			pageSize:    10,
			expectedLen: 10,
		},
		{
			name:        "page 1, size 0",
			page:        1,
			pageSize:    0,
			expectedLen: 25,
		},
		{
			name:        "page 1, size 150",
			page:        1,
			pageSize:    150,
			expectedLen: 25,
		},
		{
			name:        "maximum page does not overflow",
			page:        math.MaxUint32,
			pageSize:    10,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int64
			p := NewPaginated(tt.page, tt.pageSize, &count, db)

			var results []TestModel
			err := db.Model(&TestModel{}).Scopes(Paginate(p)).Find(&results).Error
			assert.NoError(t, err)
			assert.Len(t, results, tt.expectedLen)
			// Get total count
			db.Model(&TestModel{}).Count(&count)
			assert.Equal(t, int64(25), count) // Total count should be 25
		})
	}
}

func TestPaginated_EdgeCases(t *testing.T) {
	db := setupTestDBForPaginated(t)

	t.Run("nil count pointer", func(t *testing.T) {
		p := NewPaginated(1, 10, nil, db)
		assert.NotNil(t, p)
		// Should not panic when Paginate is called
		scope := Paginate(p)
		assert.NotNil(t, scope)
	})

	t.Run("empty database", func(t *testing.T) {
		var count int64
		p := NewPaginated(1, 10, &count, db)
		err := db.AutoMigrate(&TestModel{})
		assert.NoError(t, err)

		var results []TestModel
		err = db.Model(&TestModel{}).Scopes(Paginate(p)).Find(&results).Error
		assert.NoError(t, err)
		assert.Len(t, results, 0)
		db.Model(&TestModel{}).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
