package main

import (
	"context"
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/configuration"
)

type stubConfigurationRepository struct {
	value string
	err   error
}

func (repository stubConfigurationRepository) Get(context.Context, string) (string, error) {
	return repository.value, repository.err
}

func TestRequiredConfigurationValue(t *testing.T) {
	value, err := requiredConfigurationValue(context.Background(), stubConfigurationRepository{value: "token"}, configuration.AppTokenName)
	if err != nil || value != "token" {
		t.Fatalf("requiredConfigurationValue() = %q, %v", value, err)
	}
}

func TestRequiredConfigurationValueRejectsMissingValue(t *testing.T) {
	_, err := requiredConfigurationValue(context.Background(), stubConfigurationRepository{err: configuration.ErrNotFound}, configuration.AppTokenName)
	if !errors.Is(err, configuration.ErrNotFound) {
		t.Fatalf("requiredConfigurationValue() error = %v", err)
	}
}

func TestRequiredConfigurationValueRejectsEmptyValue(t *testing.T) {
	_, err := requiredConfigurationValue(context.Background(), stubConfigurationRepository{}, configuration.AppTokenName)
	if !errors.Is(err, errRequiredConfigurationEmpty) {
		t.Fatalf("requiredConfigurationValue() error = %v", err)
	}
}
