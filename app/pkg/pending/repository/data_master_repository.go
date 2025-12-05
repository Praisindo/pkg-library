package repository

import (
	"context"

	"github.com/Praisindo/pkg-library/app/pkg/pending/domain"
	"gorm.io/gorm"
)

type PendingRepository interface {
	//Pending
	CreatePending(ctx context.Context, tx *gorm.DB, model *domain.Pending) error
	UpdatePending(ctx context.Context, tx *gorm.DB, model *domain.Pending) error
	DeletePending(ctx context.Context, tx *gorm.DB, id int) error
	GetPending(ctx context.Context, tx *gorm.DB, tenantcode, tablename string) (*[]domain.Pending, error)
}
