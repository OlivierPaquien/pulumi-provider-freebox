package main

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
)

func TestHashSpec(t *testing.T) {
	method, value := hashSpec("deadbeef")
	assert.Equal(t, string(freeboxTypes.HashTypeSHA256), method)
	assert.Equal(t, "deadbeef", value)

	method, value = hashSpec("sha512:abc123")
	assert.Equal(t, "sha512", method)
	assert.Equal(t, "abc123", value)

	method, value = hashSpec(":onlyvalue")
	assert.Equal(t, string(freeboxTypes.HashTypeSHA256), method)
	assert.Equal(t, "onlyvalue", value)
}
