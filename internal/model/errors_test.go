package model

import (
	"errors"
	"testing"
)

func TestErrorConstants(t *testing.T) {
	// Проверяем что ошибки инициализированы и не nil
	if ErrAgentNotFound == nil {
		t.Error("ErrAgentNotFound не должна быть nil")
	}
	if ErrAgentAlreadyExists == nil {
		t.Error("ErrAgentAlreadyExists не должна быть nil")
	}
	if ErrUnauthorized == nil {
		t.Error("ErrUnauthorized не должна быть nil")
	}

	// Проверяем корректность сообщений
	if ErrAgentNotFound.Error() != "agent not found" {
		t.Errorf("ErrAgentNotFound.Error() = %v", ErrAgentNotFound.Error())
	}
	if ErrAgentAlreadyExists.Error() != "agent already exists" {
		t.Errorf("ErrAgentAlreadyExists.Error() = %v", ErrAgentAlreadyExists.Error())
	}
	if ErrUnauthorized.Error() != "unauthorized" {
		t.Errorf("ErrUnauthorized.Error() = %v", ErrUnauthorized.Error())
	}
}

func TestErrorsAreUnique(t *testing.T) {
	// Проверяем что все ошибки уникальны
	if ErrAgentNotFound == ErrAgentAlreadyExists {
		t.Error("ErrAgentNotFound и ErrAgentAlreadyExists не должны быть одинаковыми")
	}
	if ErrAgentNotFound == ErrUnauthorized {
		t.Error("ErrAgentNotFound и ErrUnauthorized не должны быть одинаковыми")
	}
	if ErrAgentAlreadyExists == ErrUnauthorized {
		t.Error("ErrAgentAlreadyExists и ErrUnauthorized не должны быть одинаковыми")
	}
}

func TestErrorsIsComparison(t *testing.T) {
	// Проверяем errors.Is для наших ошибок
	if !errors.Is(ErrAgentNotFound, ErrAgentNotFound) {
		t.Error("errors.Is() должен вернуть true для одинаковых ошибок")
	}

	if errors.Is(ErrAgentNotFound, ErrUnauthorized) {
		t.Error("errors.Is() должен вернуть false для разных ошибок")
	}
}

