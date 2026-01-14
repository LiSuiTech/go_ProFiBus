package errors

import (
	"errors"
	"testing"
)

func TestErrorCreation(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		err := New(ErrPortNotOpen, nil)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err.Code != ErrPortNotOpen {
			t.Errorf("Expected code %d, got %d", ErrPortNotOpen, err.Code)
		}
	})

	t.Run("Newf", func(t *testing.T) {
		err := Newf(ErrInvalidParam, "测试参数: %s", "test")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err.Message != "测试参数: test" {
			t.Errorf("Unexpected message: %s", err.Message)
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := Wrap(ErrPortOpenFailed, baseErr)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err.Err != baseErr {
			t.Error("Expected wrapped error")
		}
	})
}

func TestErrorIs(t *testing.T) {
	err := New(ErrCRCCheckFailed, nil)

	if !Is(err, ErrCRCCheckFailed) {
		t.Error("Is() should return true")
	}

	if Is(err, ErrPortNotOpen) {
		t.Error("Is() should return false")
	}
}

func TestErrorMessage(t *testing.T) {
	err := New(ErrPortNotOpen, nil)
	msg := err.Error()

	if msg == "" {
		t.Error("Error message should not be empty")
	}

	t.Logf("Error message: %s", msg)
}
