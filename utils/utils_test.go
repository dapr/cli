package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateString(t *testing.T) {
	truncateStrings := []struct {
		str       string
		maxLen    int
		exceptStr string
	}{
		{"dapr-injector", 4, "d..."},
		{"dapr-injector", 13, "dapr-injector"},
		{"dapr-injector", 0, "dapr-injector"},
	}
	for _, s := range truncateStrings {
		result := TruncateString(s.str, s.maxLen)
		assert.Equal(t, s.exceptStr, result, "Expected str not equal")
	}
}

func TestCreateContainerName(t *testing.T) {
	serviceContainerNames := []struct {
		serviceContainerName string
		dockerNetwork        string
		exceptName           string
	}{
		{"dapr-injector", "default", "dapr-injector_default"},
		{"dapr-injector", "", "dapr-injector"},
	}

	for _, s := range serviceContainerNames {
		containerName := CreateContainerName(s.serviceContainerName, s.dockerNetwork)
		assert.Equal(t, s.exceptName, containerName, "Expected name not equal")
	}
}

func TestIsAddressLegal(t *testing.T) {
	addresses := []struct {
		address string
		except  bool
	}{
		{"0.0.0.0", true},
		{"localhost", true},
		{"127.0.0.1", true},
		{"192.168.0.1", true},
		{"300.0.0.1", false},
		{"IP Address", false},
	}

	for _, a := range addresses {
		isLegal := IsAddressLegal(a.address)
		assert.Equal(t, a.except, isLegal, "Expected status not equal")
	}
}
