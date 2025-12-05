package structType

import (
	"errors"
	"reflect"
	"time"

	"github.com/Praisindo/pkg-library/app/pkg/constants"
	"github.com/Praisindo/pkg-library/app/pkg/loggingdata"
)

// RETURN 1 for json where, RETURN 2 for column
func GetType(dbType string, x interface{}, dst []string) ([]string, []string) {
	v := reflect.ValueOf(x)
	s := reflect.TypeOf(x)
	selected := []string{}
	selected2 := []string{}

	for i := 0; i < v.NumField(); i++ {
		for _, field := range dst {
			if field == s.Field(i).Tag.Get("json") {
				fieldType := s.Field(i).Type

				// Handle pointer types
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}

				if dbType == "postgres" {

					// PG
					switch fieldType.Kind() {
					case reflect.String:
						selected = append(selected, "(new_value->>'"+field+"')::text AS "+field)
						selected2 = append(selected2, field+"::text")
					case reflect.Int:
						selected = append(selected, "(new_value->>'"+field+"')::int AS "+field)
						selected2 = append(selected2, field+"::int")
					case reflect.Int64:
						selected = append(selected, "(new_value->>'"+field+"')::bigint AS "+field)
						selected2 = append(selected2, field+"::bigint")
					default:
						selected = append(selected, "(new_value->>'"+field+"')::"+fieldType.Name()+" AS "+field)
						selected2 = append(selected2, field+"::"+fieldType.Name())
					}
				} else if dbType == "mysql" {
					selected = append(selected, "JSON_VALUE(new_value, '$."+field+"') AS "+field)
					selected2 = append(selected2, field)
				} else {

					// MYSQL
					selected = append(selected, "JSON_VALUE(new_value, '$."+field+"') AS "+field)
					selected2 = append(selected2, field)
				}
			}
		}
	}

	return selected, selected2
}

func DeclarePendingInsert(x interface{}, requestHeader *loggingdata.RequestHeader) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.PENDING_INSERT_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_PENDING_INSERT) // Replace with appropriate constant
			case "sys_created_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_created_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_pending_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_pending_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_pending_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclarePendingUpdate(x interface{}, requestHeader *loggingdata.RequestHeader, pendingId *int) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.PENDING_UPDATE_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_PENDING_UPDATE) // Replace with appropriate constant
			case "sys_last_pending_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_approve_by":
				field.SetZero()
			case "sys_last_approve_host":
				field.SetZero()
			case "sys_last_approval_notes":
				field.SetZero()
			case "sys_last_pending_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_pending_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			case "sys_last_approve_time":

				field.SetZero()

			case "pending_id":
				{
					if field.Kind() == reflect.Ptr {
						switch field.Type().Elem().Kind() {
						case reflect.Int:
							value := *pendingId
							field.Set(reflect.ValueOf(&value))
						case reflect.Int64:
							value := int64(*pendingId)
							field.Set(reflect.ValueOf(&value))
						}
					}
				}
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclarePendingDelete(x interface{}, requestHeader *loggingdata.RequestHeader) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.PENDING_DELETE_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_PENDING_DELETE) // Replace with appropriate constant
			case "sys_last_pending_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_approve_by":
				field.SetZero()
			case "sys_last_approve_host":
				field.SetZero()
			case "sys_last_approval_notes":
				field.SetZero()
			case "sys_last_pending_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_pending_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			case "sys_last_approve_time":
				field.SetZero()

			case "pending_id":
				field.SetZero()
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclareApproveUpsert(x interface{}, requestHeader *loggingdata.RequestHeader, remarks string) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.APPROVE_UPSERT_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_ACTIVE) // Replace with appropriate constant
			case "sys_last_approve_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_approve_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_approval_notes":
				field.SetString(remarks)
			case "sys_last_pending_time":
				if field.Kind() == reflect.Ptr {
					field.SetZero()
				}
			case "sys_last_approve_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			case "pending_id":
				field.SetZero()
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclareReturnInsert(x interface{}, requestHeader *loggingdata.RequestHeader, remarks string) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.RETURN_UPSERT_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_RETURN_INSERT) // Replace with appropriate constant
			case "sys_last_approval_notes":
				field.SetString(remarks)
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclareReturnUpdate(x interface{}, requestHeader *loggingdata.RequestHeader, remarks string) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.RETURN_UPSERT_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		switch jsonTag {
		case "sys_row_status", "row_status":
			fieldsFound["sys_row_status"] = true
			field.SetInt(constants.SYSROW_STATUS_RETURN_UPDATE) // Replace with appropriate constant
		case "sys_last_approval_notes", "return_notes":
			fieldsFound["sys_last_approval_notes"] = true
			field.SetString(remarks)
		}

	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclareRejectDelUp(x interface{}, requestHeader *loggingdata.RequestHeader, remarks string) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.REJECT_DELUP_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_ACTIVE) // Replace with appropriate constant
			case "sys_last_approve_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_approve_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			case "sys_last_approve_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_approval_notes":
				field.SetString(remarks)
			case "pending_id":
				field.SetZero()
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}

func DeclareRetryInsert(x interface{}, requestHeader *loggingdata.RequestHeader) error {
	body := reflect.ValueOf(x).Elem()
	bodyType := reflect.TypeOf(x).Elem()
	now := time.Now()
	fieldsFound := make(map[string]bool)
	for _, field := range constants.RETRY_INSERT_FIELD {
		fieldsFound[field] = false
	}
	for i := 0; i < body.NumField(); i++ {
		field := body.Field(i)
		fieldType := bodyType.Field(i)
		jsonTag := fieldType.Tag.Get("json")

		if _, exists := fieldsFound[jsonTag]; exists {
			fieldsFound[jsonTag] = true
			switch jsonTag {
			case "sys_row_status":
				field.SetInt(constants.SYSROW_STATUS_PENDING_INSERT) // Replace with appropriate constant
			case "sys_created_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_created_host":
				field.SetString(requestHeader.CreatedHost)
			case "sys_last_pending_by":
				field.SetString(requestHeader.CreatedBy)
			case "sys_last_pending_time":
				if field.Kind() == reflect.Ptr {
					field.Set(reflect.ValueOf(&now))
				}
			case "sys_last_pending_host":
				field.SetString(requestHeader.CreatedHost)
			}
		}
	}
	// Check if all required fields are found
	for field, found := range fieldsFound {
		if !found {
			return errors.New("missing field: " + field)
		}
	}
	return nil
}
