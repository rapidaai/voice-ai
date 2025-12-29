// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"gorm.io/gorm"
)

type Paginated struct {
	DB       *gorm.DB
	Page     int
	PageSize int
	Count    *int64
}

func NewPaginated(page int, pageSize int, count *int64, db *gorm.DB) *Paginated {
	return &Paginated{
		Page: page, PageSize: pageSize, Count: count, DB: db,
	}
}

func Paginate(r *Paginated) func(db *gorm.DB) *gorm.DB {
	// i don't know the lifecycle of gorm scope but quick fix
	dx := make(chan bool, 1)
	go r.count(dx)
	<-dx
	return func(db *gorm.DB) *gorm.DB {
		if r.PageSize == 0 {
			return db
		}

		page := r.Page
		if page <= 0 {
			page = 1
		}

		pageSize := r.PageSize
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize < 0:
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		result := db.Offset(offset).Limit(pageSize)
		return result
	}
}

func (r *Paginated) count(dx chan bool) {
	r.DB.Count(r.Count)
	dx <- true
}
