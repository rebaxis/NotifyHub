package model

import "errors"

var (
	// ErrAgentNotFound возвращается когда агент не найден
	ErrAgentNotFound = errors.New("agent not found")

	// ErrUnauthorized возвращается при ошибке аутентификации
	ErrUnauthorized = errors.New("unauthorized")

	// ErrAgentAlreadyExists возвращается при попытке зарегистрировать агента с существующим ID
	ErrAgentAlreadyExists = errors.New("agent already exists")
)
