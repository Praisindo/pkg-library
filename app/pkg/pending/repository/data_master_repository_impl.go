package repository

import (
	"context"
	"strings"

	"github.com/Praisindo/pkg-library/app/pkg/constants"
	"github.com/Praisindo/pkg-library/app/pkg/pending/domain"
	"github.com/Praisindo/pkg-library/app/pkg/structType"
	"gorm.io/gorm"
)

type PendingRepositoryImpl struct {
	db *gorm.DB
}

func NewPendingRepository(db *gorm.DB) PendingRepository {
	return &PendingRepositoryImpl{db: db}
}

func (r *PendingRepositoryImpl) CreatePending(ctx context.Context, tx *gorm.DB, model *domain.Pending) error {

	query := tx.WithContext(ctx).Model(&domain.Pending{})
	if err := query.Create(&model).
		Error; err != nil {
		return err
	}

	return nil
}

func (r *PendingRepositoryImpl) UpdatePending(ctx context.Context, tx *gorm.DB, model *domain.Pending) error {
	query := tx.WithContext(ctx)
	if err := query.
		Model(&domain.Pending{ID: model.ID}).
		Select("new_value", "row_status", "return_notes").
		Updates(model).
		Error; err != nil {
		return err
	}
	return nil
}

func (r *PendingRepositoryImpl) DeletePending(ctx context.Context, tx *gorm.DB, id int) error {
	query := tx.WithContext(ctx)

	if err := query.
		Delete(&domain.Pending{ID: id}).Error; err != nil {
		return err
	}
	return nil
}

func (r *PendingRepositoryImpl) GetPending(
	ctx context.Context, tx *gorm.DB, tenantcode, tablename string,
) (*[]domain.Pending, error) {
	var models *[]domain.Pending
	if err := tx.
		WithContext(ctx).
		Model(&domain.Pending{}).Where("tenant_code = ?", tenantcode).Where("table_name = ?", tablename).
		Find(&models).
		Error; err != nil {
		return nil, err
	}
	return models, nil
}

func ListPending(
	tenantcode, tablename string, value interface{}, db *gorm.DB, columnList []string,
) *gorm.DB {
	// Selected
	selectSubQuery1, selectSubQuery2 := structType.GetType(db.Config.Dialector.Name(), value, columnList)
	selectColumn1 := strings.Join(selectSubQuery1, ",")
	selectColumn2 := strings.Join(selectSubQuery2, ",")

	// Subquery for financial_pending_data
	subQuery1 := db.Table((&domain.Pending{}).TableName()).
		Select(selectColumn1).
		Where("tenant_code=? AND table_name=?", tenantcode, tablename)

	// Subquery for financial_financials
	subQuery2 := db.Table(tablename).
		Select(selectColumn2).
		Where("sys_row_status IN ?", constants.FILTER_PENDING).
		Where("tenant_code=?", tenantcode)

	// Combine the subqueries using UNION ALL
	return db.Table("(?) AS combined", db.Raw("? UNION ALL ?", subQuery1, subQuery2))
}
