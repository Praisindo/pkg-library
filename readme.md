This library will be used as main library for helpers


````
go get github.com/Praisindo/pkg-library
````

to update:
````
go get -u github.com/Praisindo/pkg-library@v.x.x
````

How to Use CQRS

1. In the main package you are using, make sure you import "github.com/Praisindo/pkg-library/app/pkg/database".

2. During the declaration of using the database, you would run a function from the database folder called NewDatabase(driver string, cfg *Config)

   2.1 Config of db consists of:
    ````
    type Config struct {
    DbHost   string
    DbUser   string
    DbPass   string
    DbName   string
    DbPort   string
    DbPrefix string
    }
    `````
   2.2 Driver you can use: "postgres" / "pgsql", "mysql", "sqlserver", "oracle".

This declaration of the Database will return a struct consisting of:
```
type Database struct {
    db     *gorm.DB
    isCqrs bool
}
```

3. If you want to enable CQRS, you may use CqrsDB(driver string, cfg *Config) to insert the replica value.

4. For migrating purposes, you would use methods from the Database struct:
````
    MigrateDB(dst ...interface{})
    DownMigrate(all bool, dst ...interface{})
    DropColumnDB(dst interface{}, columnTarget string)
    RenameColumnDB(dst interface{}, oldname, columnTarget string)
    DownIndexDB(dst interface{}, columnTarget string)
    WipeTable(dst interface{})
    DeleteTable(dst ...interface{})
````    

5. Depending on the isCqrs value in the Database struct, the migration would run in master only or master-replica.

6. Dual Approval
# Dual Approval System - pkg-library

## Overview
This project implements a Dual Approval system using GORM, designed to manage data entities with an approval workflow. The system supports multiple states for data entities (pending insert, pending update, pending delete, active, etc.) and integrates with an audit service for tracking changes.

- **Pending Repository**: [Pending Repository](./app/pkg/pending)
- **Audit Service**: [Audit Service Repository](https://github.com/Praisindo/cpm-audit-service)

---

## System Design

The complete system design and documentation are available here:
- [System Design Documentation](https://praisindo.getoutline.com/s/1d3acb03-4e66-4e7f-a401-9e0444c32bb5)

---

## Struct Definitions

### Example Struct

This struct holds the necessary fields, including pending data and system metadata, to manage the approval process:

```go
type Example struct {
  ID                   string                 `bson:"_id" json:"id" validate:"required,uuid" gorm:"primaryKey;type:uuid"`
  Title                string                 `bson:"title" json:"title" validate:"required"`
  TenantCode           string                 `gorm:"size:100" json:"tenant_code"`
  PendingId            *int                   `json:"pending_id"`
  Pending              *pendingDomain.Pending `gorm:"foreignKey:PendingId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"pending,omitempty"`
  SysRowStatus         int                    `gorm:"column:sys_row_status" json:"sys_row_status"`
  SysCreatedBy         string                 `gorm:"type:varchar(100);column:sys_created_by" json:"sys_created_by"`
  SysCreatedTime       *time.Time             `gorm:"autoCreateTime;column:sys_created_time" json:"sys_created_time"`
  SysCreatedHost       string                 `gorm:"type:varchar(256);column:sys_created_host" json:"sys_created_host"`
  SysLastApprovalNotes string                 `gorm:"type:varchar(512);column:sys_last_approval_notes" json:"sys_last_approval_notes"`
  SysLastPendingBy     string                 `gorm:"type:varchar(100);column:sys_last_pending_by" json:"sys_last_pending_by"`
  SysLastPendingTime   *time.Time             `gorm:"default:null;column:sys_last_pending_time" json:"sys_last_pending_time"`
  SysLastPendingHost   string                 `gorm:"type:varchar(256);column:sys_last_pending_host" json:"sys_last_pending_host"`
  SysLastApproveBy     string                 `gorm:"type:varchar(100);column:sys_last_approve_by" json:"sys_last_approve_by"`
  SysLastApproveTime   *time.Time             `gorm:"default:null;column:sys_last_approve_time" json:"sys_last_approve_time"`
  SysLastApproveHost   string                 `gorm:"column:sys_last_approve_host" json:"sys_last_approve_host"`
}
```
## How to Get TenantCode
- **RequestHeader**: [](./app/pkg/loggingdata)
```go
func (h ExampleHandler) CreateExample(ctx *gin.Context) {
	    var RequestHeader *loggingdata.RequestHeader
	    RequestHeader = loggingdata.GetRequestHeader(ctx)
        tenantCode := RequestHeader.TenantCode
	//example Handler using gin
}
```


---

## Service Implementation

The service interacts with GORM, the repository, and the audit service to manage the approval workflow.

### ExampleServiceImpl Struct

```go
type ExampleServiceImpl struct {
  db                    *gorm.DB
  exampleRepo           repository.ExampleRepository
  PendingDataRepository repository2.PendingRepository //from pending repo provided pkg-library
  AuditSvcExternal      auditExternal.AuditSvcExternal
}
```

---

## CRUD Operations

### Insert (Pending Insert)

This function inserts data into the database with `sys_row_status = 0` (pending insert).

#### Flow:

1. Check for duplicates in the main table and pending table.
2. Declare Pending Insert using the structType function.
3. Initiate pending insert.

```go
func (s *ExampleServiceImpl) CreateExample(
	ctx context.Context, model *entity.UpsertExample, requestHeader *loggingdata.RequestHeader,
) *exception.Exception {
	tx := s.db.Begin()
	defer tx.Rollback()

	txRead := s.db
	body := &entity.Example{ID: uuid.NewString(), TenantCode: requestHeader.TenantCode}

	auditRequest := auditExternal.NewRequestAuditCreate(
		requestHeader.TenantCode,
		(&entity.Example{}).TableName(),
		requestHeader.CreatedHost,
		requestHeader.CreatedBy,
		"title",
		constants.ACTION_TYPE_INSERT,
		constants.APPROVAL_STATUS_PENDING,
		constants.AUDIT_STATUS_SUCCESS,
	)
	defer func() {
		auditRequest.DeclareAuditEndTime()
		if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
			slog.Error("failed to create audit", slog.Any("error", err))
		}
	}()

	duplicateCheck, err := s.exampleRepo.FindOneExampleByTitle(ctx, txRead, requestHeader.TenantCode, model.Title)
	if err != nil {
		auditRequest.DeclareAuditError("Failed to check duplicate", err)
		return exception.Internal("error finding example by title", err)
	}
	if duplicateCheck != nil {
		auditRequest.DeclareAuditError("example with title :"+model.Title+" already exists", nil)
		return exception.AlreadyExists("example with title :" + model.Title + " already exists")
	}

	checkUniquePending, err := s.PendingDataRepository.GetPending(ctx, txRead, requestHeader.TenantCode, (&entity.Example{}).TableName())
	if err != nil {
		auditRequest.DeclareAuditError("Failed to check duplicate", err)
		return exception.Internal("Failed to check duplicate", err)
	}
	if len(*checkUniquePending) > 0 {
		uniquePendingData := entity.Example{}
		for _, pending := range *checkUniquePending {
			if pending.NewValue != nil {
				if err := json.Unmarshal(pending.NewValue, &uniquePendingData); err != nil {
					auditRequest.DeclareAuditError("failed to unmarshal json", err)
					return exception.Internal("failed to unmarshal json", err)
				}
				if uniquePendingData.Title == model.Title {
					auditRequest.DeclareAuditError("Title already exists", nil)
					return exception.AlreadyExists("Title already exists")
				}
			}
		}
	}

	if err := structType.DeclarePendingInsert(body, requestHeader); err != nil {
		auditRequest.DeclareAuditError("Failed declare", err)
		return exception.Internal("declare error: ", err)
	}

	if err := s.exampleRepo.CreateExample(ctx, tx, body); err != nil {
		return exception.Internal("error creating example", err)
	}

	if err := tx.Commit().Error; err != nil {
		return exception.Internal("commit transaction", err)
	}

	auditRequest.DeclareAuditTraceID(body.ID)
	auditRequest.DeclareAuditNewValue(body)
	return nil
}

```

---

### Update (Retry Insert, Retry Update, Pending Update)

This function handles retrying insert/update or marking an update as pending, based on the current `sys_row_status`.

#### Flow:

1. Check for duplicates.
2. Check if current data's `sys_row_status = 1` (active), `4` (return insert), or `5` (return update).
3. Declare Insert Retry/Update Retry/Pending Update using structType function.
4. Initiate update.

```go
func (s *ExampleServiceImpl) UpdateExample(
	ctx context.Context, id string, model *entity.UpsertExample, requestHeader *loggingdata.RequestHeader,
) *exception.Exception {
	tx := s.db.Begin()
	defer tx.Rollback()

	txRead := s.db
	body := &entity.Example{ID: id, TenantCode: requestHeader.TenantCode}

	auditRequest := auditExternal.NewRequestAuditCreate(
		requestHeader.TenantCode,
		(&entity.Example{}).TableName(),
		requestHeader.CreatedHost,
		requestHeader.CreatedBy,
		"title",
		constants.ACTION_TYPE_UPDATE,
		constants.APPROVAL_STATUS_PENDING,
		constants.AUDIT_STATUS_SUCCESS,
	)
	defer func() {
		auditRequest.DeclareAuditEndTime()
		if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
			slog.Error("failed to create audit", slog.Any("error", err))
		}
	}()

	detailExample, err := s.exampleRepo.FindOneExample(ctx, txRead, id)
	if err != nil {
		return exception.Internal("error finding example", err)
	}
	if detailExample == nil {
		auditRequest.DeclareAuditError("example with id of "+id+" not found", nil)
		return exception.NotFound("example with id of " + id + " not found")
	}

	if detailExample.SysRowStatus != constants.SYSROW_STATUS_ACTIVE && detailExample.SysRowStatus != constants.SYSROW_STATUS_RETURN_INSERT && detailExample.SysRowStatus != constants.SYSROW_STATUS_RETURN_UPDATE {
		auditRequest.DeclareAuditError("example with id of "+id+" still has pending status", nil)
		return exception.PermissionDenied("example with id of " + id + " still has pending status")
	}

	duplicateCheck, err := s.exampleRepo.FindOneExampleByTitle(ctx, txRead, requestHeader.TenantCode, model.Title)
	if err != nil {
		return exception.Internal("error finding example by title", err)
	}
	if duplicateCheck != nil && duplicateCheck.ID != id {
		return exception.AlreadyExists("example with title :" + model.Title + " already exists")
	}

	checkUniquePending, err := s.PendingDataRepository.GetPending(ctx, txRead, requestHeader.TenantCode, (&entity.Example{}).TableName())
	if err != nil {
		auditRequest.DeclareAuditError("Failed to check duplicate", err)
		return exception.Internal("Failed to check duplicate", err)
	}
	if len(*checkUniquePending) > 0 {
		uniquePendingData := entity.Example{}
		for _, pending := range *checkUniquePending {
			if pending.NewValue != nil {
				if err := json.Unmarshal(pending.NewValue, &uniquePendingData); err != nil {
					auditRequest.DeclareAuditError("failed to unmarshal json", err)
					return exception.Internal("failed to unmarshal json", err)
				}
				if uniquePendingData.Title == model.Title && uniquePendingData.ID != id {
					auditRequest.DeclareAuditError("Title already exists", nil)
					return exception.AlreadyExists("Title already exists")
				}
			}
		}
	}

	if detailExample.SysRowStatus == constants.SYSROW_STATUS_RETURN_INSERT {
		// RETRY INSERT
		if err := structType.DeclareRetryInsert(body, requestHeader); err != nil {
			auditRequest.DeclareAuditError("Failed declare", err)
			return exception.Internal("declare error: ", err)
		}
		auditRequest.DeclareInsertRetry(id, detailExample, body)
		detailExample = body
	} else if detailExample.SysRowStatus == constants.SYSROW_STATUS_RETURN_UPDATE && detailExample.Pending.ActionType == constants.ACTION_TYPE_UPDATE {
		// RETRY UPDATE
		auditRequest.DeclareUpdateRetry(id, detailExample, body)
		// Update Pending
		jsonData, err := json.Marshal(body)
		if err != nil {
			auditRequest.DeclareAuditError("error marshalling old value", err)
			return exception.Internal("error marshalling old value", err)
		}
		requestPending := &pendingdatadomain.Pending{
			ID:          detailExample.Pending.ID,
			TenantCode:  requestHeader.TenantCode,
			TblName:     (&entity.Example{}).TableName(),
			ObjectId:    body.ID,
			ObjectName:  "title",
			ActionType:  constants.ACTION_TYPE_UPDATE,
			RowStatus:   constants.SYSROW_STATUS_PENDING_UPDATE,
			NewValue:    jsonData,
			PendingBy:   requestHeader.CreatedBy,
			PendingHost: requestHeader.CreatedHost,
		}
		// Update Pending
		err = s.PendingDataRepository.UpdatePending(ctx, tx, requestPending)
		if err != nil {
			return exception.Internal("update pending data", err)
		}

		// Update Master
		if err := structType.DeclarePendingUpdate(detailExample, requestHeader, &requestPending.ID); err != nil {
			auditRequest.DeclareAuditError("Failed declare", err)
			return exception.Internal("declare error: ", err)
		}
	} else {
		auditRequest.DeclareAuditOldValue(detailExample)
		auditRequest.DeclareAuditNewValue(body)

		// UPDATE Normal
		jsonData, err := json.Marshal(body)
		if err != nil {
			auditRequest.DeclareAuditError("error marshalling old value", err)
			return exception.Internal("error marshalling old value", err)
		}
		requestPending := &pendingdatadomain.Pending{
			TenantCode:  requestHeader.TenantCode,
			TblName:     (&entity.Example{}).TableName(),
			ObjectId:    body.ID,
			ObjectName:  "title",
			ActionType:  constants.ACTION_TYPE_UPDATE,
			RowStatus:   constants.SYSROW_STATUS_PENDING_UPDATE,
			NewValue:    jsonData,
			PendingBy:   requestHeader.CreatedBy,
			PendingHost: requestHeader.CreatedHost,
		}
		if err := s.PendingDataRepository.CreatePending(ctx, tx, requestPending); err != nil {
			auditRequest.DeclareAuditError("Failed insert pending data", err)
			return exception.Internal("Failed insert pending data", err)
		}
		if err := structType.DeclarePendingUpdate(detailExample, requestHeader, &requestPending.ID); err != nil {
			auditRequest.DeclareAuditError("Failed declare", err)
			return exception.Internal("declare error: ", err)
		}
	}

	if err := s.exampleRepo.UpdateExample(ctx, tx, detailExample); err != nil {
		return exception.Internal("error updating example", err)
	}

	if err := tx.Commit().Error; err != nil {
		return exception.Internal("commit transaction", err)
	}

	auditRequest.DeclareAuditTraceID(body.ID)
	return nil
}

```

---

### Delete (Pending Delete)

This function declares a pending delete.

#### Flow:

1. Check if data exists.
2. Check if `sys_row_status = 1` (active).
3. Initiate delete.

```go
func (s *ExampleServiceImpl) DeleteExample(
	ctx context.Context, id string, requestHeader *loggingdata.RequestHeader,
) *exception.Exception {
	tx := s.db.Begin()
	defer tx.Rollback()

	auditRequest := auditExternal.NewRequestAuditCreate(
		requestHeader.TenantCode,
		(&entity.Example{}).TableName(),
		requestHeader.CreatedHost,
		requestHeader.CreatedBy,
		"title",
		constants.ACTION_TYPE_DELETE,
		constants.APPROVAL_STATUS_PENDING,
		constants.AUDIT_STATUS_SUCCESS,
	)
	defer func() {
		auditRequest.DeclareAuditEndTime()
		if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
			slog.Error("failed to create audit", slog.Any("error", err))
		}
	}()

	detailExample, err := s.exampleRepo.FindOneExample(ctx, tx, id)
	if err != nil {
		return exception.Internal("error finding example", err)
	}
	if detailExample == nil {
		auditRequest.DeclareAuditError("example with id of "+id+" not found", nil)
		return exception.NotFound("example with id of " + id + " not found")
	}

	if detailExample.SysRowStatus != constants.SYSROW_STATUS_ACTIVE {
		auditRequest.DeclareAuditError("example with id of "+id+" still has pending status", nil)
		return exception.PermissionDenied("example with id of " + id + " still has pending status")
	}

	// Update Data
	if err := structType.DeclarePendingDelete(detailExample, requestHeader); err != nil {
		auditRequest.DeclareAuditError("Failed declare", err)
		return exception.Internal("declare error: ", err)
	}

	if err := s.exampleRepo.UpdateExample(ctx, tx, detailExample); err != nil {
		return exception.Internal("error deleting example", err)
	}

	if err := tx.Commit().Error; err != nil {
		return exception.Internal("commit transaction", err)
	}

	auditRequest.DeclareAuditOldValue(detailExample)
	auditRequest.DeclareAuditTraceID(id)
	return nil
}

```

---

## Approval Workflow

### Insert/Update/Delete Approval for Pending Data

This function approves pending inserts, updates, or deletes (based on `sys_row_status = 0`, `2`, or `3`).

#### Example of UpdateApproval Struct:

```go
type UpdateApproval struct {
  Id      []string `json:"id" example:"[\"123e4567-e89b-12d3-a456-426614174000\",\"987e6543-b21c-34f5-d678-123456789abc\"]"`
  Remarks string   `json:"remarks" example:"Approval needed for the following templates due to new updates"`
}
```

#### Flow:

1. Check if data exists.
2. Check `sys_row_status` (pending insert, pending update, pending delete).
3. Use the structType to declare the specific condition based on the current status.
4. Update and commit.

```go
func (s *ExampleServiceImpl) ApprovalExample(
	ctx context.Context, approval string, request *model.UpdateApproval, requestHeader *loggingdata.RequestHeader,
) *exception.Exception {
	// Init Transaction
	tx := s.db.Begin()
	defer tx.Rollback()

	txRead := s.db

	var auditRequests []*auditExternal.RequestAuditCreate
	approvalBool, err := strconv.ParseBool(approval)
	if err != nil {
		return exception.PermissionDenied("Input of approval must be true/false")
	}
	if approvalBool {
		for _, id := range request.Id {
			_, err := uuid.Parse(id)
			if err != nil {
				return exception.PermissionDenied("Input of id must be UUID, input: " + id)
			}
			auditRequest := auditExternal.NewRequestAuditCreate(
				requestHeader.TenantCode,
				(&entity.Example{}).TableName(),
				requestHeader.CreatedHost,
				requestHeader.CreatedBy,
				"title",
				constants.ACTION_TYPE_UPDATE,
				constants.APPROVAL_STATUS_PENDING,
				constants.AUDIT_STATUS_SUCCESS,
			)
			detailExample, err := s.exampleRepo.FindOneExample(ctx, txRead, id)
			if err != nil {
				auditRequest.DeclareAuditError("failed get detail example, id table: "+id, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("failed get example", err)
			}

			if detailExample == nil {
				continue
			}
			if detailExample.SysRowStatus != constants.SYSROW_STATUS_ACTIVE {
				// Check Approval Mode
				if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_INSERT {
					if err := structType.DeclareApproveUpsert(detailExample, requestHeader, request.Remarks); err != nil {
						auditRequest.DeclareAuditError("failed update pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("declare error: ", err)
					}
					auditRequest.DeclareInsertApprove(detailExample.ID, request.Remarks, detailExample)
				} else if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_UPDATE && detailExample.Pending.ActionType == constants.ACTION_TYPE_UPDATE {
					auditRequest.DeclareUpdateApprove(detailExample.ID, request.Remarks, detailExample, detailExample.Pending.NewValue)
					if err := json.Unmarshal(detailExample.Pending.NewValue, detailExample); err != nil {
						auditRequest.DeclareAuditError("failed to unmarshal json, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("failed to unmarshal json", err)
					}
					if err := structType.DeclareApproveUpsert(detailExample, requestHeader, request.Remarks); err != nil {
						auditRequest.DeclareAuditError("failed update pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("declare error: ", err)
					}
					err := s.PendingDataRepository.DeletePending(ctx, tx, detailExample.Pending.ID)
					if err != nil {
						auditRequest.DeclareAuditError("failed deleting pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("failed deleting pending", err)
					}
					detailExample.PendingId = nil
					detailExample.Pending = nil
				} else if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_DELETE {
					auditRequest.DeclareDeleteApprove(id, request.Remarks, detailExample)
					err = s.exampleRepo.DeleteExample(ctx, tx, detailExample.ID)
					if err != nil {
						auditRequest.DeclareAuditError("failed deleting example, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("failed deleting example", err)
					}
					auditRequests = append(auditRequests, auditRequest)
					continue
				}
				if err := s.exampleRepo.UpdateExample(ctx, tx, detailExample); err != nil {
					auditRequest.DeclareAuditError("failed update example, id table: "+detailExample.ID, err)
					go func(
						auditRequest *auditExternal.RequestAuditCreate,
					) {
						if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
							slog.Error("failed to create audit", slog.Any("error", err))
						}
					}(auditRequest)
					return exception.Internal("failed update example", err)
				}
				auditRequests = append(auditRequests, auditRequest)
			}
		}
	} else {
		for _, id := range request.Id {
			auditRequest := auditExternal.NewRequestAuditCreate(
				requestHeader.TenantCode,
				(&entity.Example{}).TableName(),
				requestHeader.CreatedHost,
				requestHeader.CreatedBy,
				"title",
				constants.ACTION_TYPE_UPDATE,
				constants.APPROVAL_STATUS_PENDING,
				constants.AUDIT_STATUS_SUCCESS,
			)
			detailExample, err := s.exampleRepo.FindOneExample(ctx, txRead, id)
			if err != nil {
				auditRequest.DeclareAuditError("failed get detail example, id table: "+id, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("failed get detail example", err)
			}
			if detailExample == nil {
				continue
			}
			if detailExample.SysRowStatus != constants.SYSROW_STATUS_ACTIVE {
				// Check Approval Mode
				if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_INSERT {
					// Reject Insert
					auditRequest.DeclareInsertReject(id, request.Remarks, detailExample)
					err = s.exampleRepo.DeleteExample(ctx, tx, detailExample.ID)
					if err != nil {
						auditRequest.DeclareAuditError("failed deleting example, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("failed deleting example", err)
					}
					auditRequests = append(auditRequests, auditRequest)
					continue
				} else if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_UPDATE && detailExample.Pending.ActionType == constants.ACTION_TYPE_UPDATE {
					auditRequest.DeclareUpdateReject(detailExample.ID, request.Remarks, detailExample, detailExample.Pending.NewValue)
					if err := structType.DeclareRejectDelUp(detailExample, requestHeader, request.Remarks); err != nil {
						auditRequest.DeclareAuditError("failed update pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("declare error: ", err)
					}
					err := s.PendingDataRepository.DeletePending(ctx, tx, detailExample.Pending.ID)
					if err != nil {
						auditRequest.DeclareAuditError("failed deleting pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("failed deleting pending", err)
					}
					detailExample.PendingId = nil
					detailExample.Pending = nil
				} else if detailExample.SysRowStatus == constants.SYSROW_STATUS_PENDING_DELETE {
					// Reject Delete
					if err := structType.DeclareRejectDelUp(detailExample, requestHeader, request.Remarks); err != nil {
						auditRequest.DeclareAuditError("failed update pending, id table: "+detailExample.ID, err)
						go func(
							auditRequest *auditExternal.RequestAuditCreate,
						) {
							if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
								slog.Error("failed to create audit", slog.Any("error", err))
							}
						}(auditRequest)
						return exception.Internal("declare error: ", err)
					}
					auditRequest.DeclareDeleteReject(detailExample.ID, request.Remarks, detailExample)
				}
				if err := s.exampleRepo.UpdateExample(ctx, tx, detailExample); err != nil {
					auditRequest.DeclareAuditError("failed update example, id table: "+detailExample.ID, err)
					go func(
						auditRequest *auditExternal.RequestAuditCreate,
					) {
						if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
							slog.Error("failed to create audit", slog.Any("error", err))
						}
					}(auditRequest)
					return exception.Internal("failed update example", err)
				}
				auditRequests = append(auditRequests, auditRequest)
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		return exception.Internal("failed commit transaction", err)
	}

	// Send audit logs after successful commit
	for _, auditRequest := range auditRequests {
		go func(auditRequest *auditExternal.RequestAuditCreate) {
			if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
				slog.Error("failed to create audit", slog.Any("error", err))
			}
		}(auditRequest)
	}
	return nil
}

```

---

## Return Pending Data

This function returns data marked with `sys_row_status` as pending (either insert or update).

#### Flow:

1. Check if data exists.
2. Check `sys_row_status = 0` (pending insert) or `2` (pending update).
3. Declare specific return conditions based on `sys_row_status`.
4. Update and commit.

```go
func (s *ExampleServiceImpl) ReturnExample(
	ctx context.Context, request *model.UpdateApproval, requestHeader *loggingdata.RequestHeader,
) *exception.Exception {
	// Init Transaction
	tx := s.db.Begin()
	defer tx.Rollback()
	var auditRequests []*auditExternal.RequestAuditCreate

	for _, id := range request.Id {
		auditRequest := auditExternal.NewRequestAuditCreate(
			requestHeader.TenantCode,
			(&entity.Example{}).TableName(),
			requestHeader.CreatedHost,
			requestHeader.CreatedBy,
			"title",
			constants.ACTION_TYPE_UPDATE,
			constants.APPROVAL_STATUS_RETURN,
			constants.AUDIT_STATUS_SUCCESS,
		)
		detailById, err := s.exampleRepo.FindOneExample(ctx, s.db, id)
		if err != nil {
			auditRequest.DeclareAuditError("failed get detail example, id table: "+id, err)
			go func(
				auditRequest *auditExternal.RequestAuditCreate,
			) {
				if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
					slog.Error("failed to create audit", slog.Any("error", err))
				}
			}(auditRequest)
			return exception.Internal("get detail", err)
		}
		if detailById == nil {
			continue
		}

		// Check Approval Mode
		if detailById.SysRowStatus == constants.SYSROW_STATUS_PENDING_INSERT {
			if err := structType.DeclareReturnInsert(detailById, requestHeader, request.Remarks); err != nil {
				auditRequest.DeclareAuditError("failed update pending, id table: "+detailById.ID, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("declare error: ", err)
			}
			auditRequest.DeclareInsertReturn(detailById.ID, request.Remarks, detailById)

		} else if detailById.SysRowStatus == constants.SYSROW_STATUS_PENDING_UPDATE && detailById.Pending.ActionType == constants.ACTION_TYPE_UPDATE {
			auditRequest.DeclareUpdateReturn(detailById.ID, request.Remarks, detailById, detailById.Pending.NewValue)
			if err := structType.DeclareReturnUpdate(detailById.Pending, requestHeader, request.Remarks); err != nil {
				auditRequest.DeclareAuditError("failed update pending, id table: "+detailById.ID, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("declare error: ", err)
			}
			err := s.PendingDataRepository.UpdatePending(ctx, tx, detailById.Pending)
			if err != nil {
				auditRequest.DeclareAuditError("failed update pending, id table: "+detailById.ID, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("update pending", err)
			}

			// Master
			if err := structType.DeclareReturnUpdate(detailById, requestHeader, request.Remarks); err != nil {
				auditRequest.DeclareAuditError("failed update pending, id table: "+detailById.ID, err)
				go func(
					auditRequest *auditExternal.RequestAuditCreate,
				) {
					if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
						slog.Error("failed to create audit", slog.Any("error", err))
					}
				}(auditRequest)
				return exception.Internal("declare error: ", err)
			}
		}

		// Update Data
		err = s.exampleRepo.UpdateExample(ctx, tx, detailById)
		if err != nil {
			auditRequest.DeclareAuditError("failed update example, id table: "+detailById.ID, err)
			go func(
				auditRequest *auditExternal.RequestAuditCreate,
			) {
				if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
					slog.Error("failed to create audit", slog.Any("error", err))
				}
			}(auditRequest)
			return exception.Internal("update example", err)
		}
		auditRequests = append(auditRequests, auditRequest)
	}

	// commit trans
	if err := tx.Commit().Error; err != nil {
		return exception.Internal("commit transaction", err)
	}
	// Send audit logs after successful commit
	for _, auditRequest := range auditRequests {
		go func(auditRequest *auditExternal.RequestAuditCreate) {
			if err := s.AuditSvcExternal.CreateAudit(ctx, auditRequest); err != nil {
				slog.Error("failed to create audit", slog.Any("error", err))
			}
		}(auditRequest)
	}
	return nil
}
```

---

## Conclusion

This Dual Approval System is built to ensure data integrity through a pending approval mechanism for inserts, updates, and deletes. It integrates closely with an audit service to track every change and rollback where necessary. For more information on the system design, refer to the [System Design Documentation](https://praisindo.getoutline.com/s/1d3acb03-4e66-4e7f-a401-9e0444c32bb5) and the [Audit Service Repository](https://github.com/Praisindo/cpm-audit-service).

