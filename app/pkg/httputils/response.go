package httputils

import (
	"github.com/Praisindo/pkg-library/app/pkg/database"
	"github.com/Praisindo/pkg-library/app/pkg/exception"
)

// ErrorResponse is a struct to represent error http response
type ErrorResponse struct {
	StatusCode  int    `json:"status_code"`
	Message     any    `json:"message"`
	DetailError string `json:"error"`
}

func (r *ErrorResponse) Error() string {
	return r.DetailError
}

// GenErrorResponse is a function to generate error http response
func GenErrorResponse(statusCode int, message any, err error) *ErrorResponse {
	if err == nil {
		return &ErrorResponse{
			StatusCode: statusCode,
			Message:    message,
		}
	}
	return &ErrorResponse{
		StatusCode:  statusCode,
		Message:     message,
		DetailError: err.Error(),
	}
}

// GenErrorResponseException is a function to generate error response from exception
func GenErrorResponseException(exc *exception.Exception) *ErrorResponse {
	switch exc.Code {
	case exception.InvalidArgumentCode:
		return GenErrorResponse(400, exc.Message, exc.Error)
	case exception.NotFoundCode:
		return GenErrorResponse(404, exc.Message, exc.Error)
	case exception.AlreadyExistsCode:
		return GenErrorResponse(409, exc.Message, exc.Error)
	case exception.PermissionDeniedCode:
		return GenErrorResponse(403, exc.Message, exc.Error)
	case exception.UnauthenticatedCode:
		return GenErrorResponse(401, exc.Message, exc.Error)
	case exception.InternalErrorCode:
		return GenErrorResponse(500, exc.Message, exc.Error)
	case exception.UnprocessableEntityCode:
		return GenErrorResponse(422, exc.Message, exc.Error)
	default:
		return GenErrorResponse(500, exc.Message, exc.Error)
	}
}

type ExceptionResponse struct {
	StatusCode int `json:"status_code"`
	Message    any `json:"message"`
}

type SuccessResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type DataSuccessResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
}

type PaginationDataSuccessResponse struct {
	StatusCode int               `json:"status_code"`
	Message    string            `json:"message"`
	Pagination database.Paginate `json:"pagination"`
	Data       any               `json:"data"`
}
