// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"math"

	"github.com/rapidaai/pkg/utils"
	"gorm.io/gorm"
)

type Paginated struct {
	DB       *gorm.DB
	Page     uint32
	PageSize uint32
	Count    *int64
}

func NewPaginated(page uint32, pageSize uint32, count *int64, db *gorm.DB) *Paginated {
	return &Paginated{
		Page: page, PageSize: pageSize, Count: count, DB: db,
	}
}

func Paginate(r *Paginated) func(db *gorm.DB) *gorm.DB {
	if r.Count != nil {
		r.DB.Count(r.Count)
	}
	return func(db *gorm.DB) *gorm.DB {
		if r.PageSize == 0 {
			return db
		}

		page := 1
		if r.Page > 0 {
			convertedPage, err := utils.Uint32ToInt(r.Page)
			if err != nil {
				page = math.MaxInt
			} else {
				page = convertedPage
			}
		}

		pageSize := 100
		if r.PageSize <= 100 {
			pageSize = int(r.PageSize)
		}
		offset := 0
		if page > 1 {
			if page-1 > math.MaxInt/pageSize {
				offset = math.MaxInt
			} else {
				offset = (page - 1) * pageSize
			}
		}
		result := db.Offset(offset).Limit(pageSize)
		return result
	}
}
