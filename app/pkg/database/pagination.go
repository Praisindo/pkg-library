package database

import (
	"math"

	"github.com/Praisindo/pkg-library/app/pkg/getfilter"
	"gorm.io/gorm"
)

type Paginate struct {
	Limit      int    `json:"limit,omitempty"`
	Page       int    `json:"page,omitempty"`
	Sort       string `json:"sort"`
	Order      string `json:"order"`
	TotalRows  int    `json:"total_rows,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
}

func NewPaginate(limit, page int, arrSort []getfilter.FilterSort) *Paginate {
	var sort, order string
	for _, filter := range arrSort {
		sort = sort + filter.Field + ","
		order = order + filter.Value + ","
	}
	return &Paginate{
		Limit: limit, Page: page,
		Sort: sort, Order: order,
	}
}

func (p *Paginate) ApplySorting(sort, order string) {
	p.Sort = sort
	p.Order = order
}

func (p *Paginate) PaginatedResult(value interface{}, db *gorm.DB) func(db *gorm.DB) *gorm.DB {
	var totalRows int64
	db.Model(value).Count(&totalRows)
	if p.Limit == 0 {
		p.Limit = int(totalRows)
	}
	if p.Page == 0 {
		p.Page = 1
	}
	offset := (p.Page - 1) * p.Limit

	p.TotalRows = int(totalRows)
	p.TotalPages = int(math.Ceil(float64(totalRows) / float64(p.Limit)))
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(p.Limit).Offset(offset)
	}
}

func (p *Paginate) PaginatedResultJoins(
	value interface{},
	db *gorm.DB,
	joinConditions []string,
) func(db *gorm.DB) *gorm.DB {
	var totalRows int64

	for _, joinCondition := range joinConditions {
		db = db.Joins(joinCondition)
	}

	db.Model(value).Count(&totalRows)
	if p.Limit == 0 {
		p.Limit = int(totalRows)
	}
	if p.Page == 0 {
		p.Page = 1
	}
	offset := (p.Page - 1) * p.Limit

	p.TotalRows = int(totalRows)
	p.TotalPages = int(math.Ceil(float64(totalRows) / float64(p.Limit)))
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(p.Limit).Offset(offset)
	}
}
