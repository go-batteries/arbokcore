package api

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type ErrorResponse struct {
	Message    string `json:"message"`
	HttpStatus int    `json:"-"`
	Code       int    `json:"code"`
}

type Response struct {
	Success bool           `json:"success"`
	Data    any            `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

// Parse error message in the format
// err_key:err_code:status_code

func ParserErrorResponse(err error) *ErrorResponse {
	errStr := err.Error()

	splits := strings.Split(errStr, ":")
	if len(splits) < 3 {
		return &ErrorResponse{
			Message:    "terrible",
			HttpStatus: 501,
			Code:       9,
		}
	}

	key, code, status := splits[0], splits[1], splits[2]

	// use key for internationalization later
	log.Error().Err(err).
		Str("error_reason", key).Msg("failed")

	statusCode, _ := strconv.Atoi(status)
	errorCode, _ := strconv.Atoi(code)

	return &ErrorResponse{
		Message:    key,
		HttpStatus: statusCode,
		Code:       errorCode,
	}
}

func BuildResponse(err error, data any) Response {
	if err != nil {
		return Response{
			Success: false,
			Error:   ParserErrorResponse(err),
		}
	}

	return Response{
		Success: true,
		Error:   nil,
		Data:    data,
	}
}
