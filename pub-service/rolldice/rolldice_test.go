package rolldice

import (
	"bytes"
	"context"
	"encoding/json"
	"go-sandbox/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRollDice(t *testing.T) {
	requestBody, _ := json.Marshal(map[string]int8{
		"sides": 6,
		"rolls": 3,
	})

	req, err := http.NewRequest("POST", "/rolldice", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RollDice)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "RollDiceResponse should be OK")

	var response RollDiceResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int8(6), response.Sides, "Sides should be equal to 6")
	assert.Equal(t, int8(3), response.Rolls, "Rolls should be equal to 3")
}

func TestRollDiceInvalidInput(t *testing.T) {
	_, err := rollDice(context.Background(), 1, 1)
	assert.Error(t, err, "Should return an error for invalid input")

	_, err = rollDice(context.Background(), 101, 1)
	assert.Error(t, err, "Should return an error for invalid input")
}

func TestRollDiceDistribution(t *testing.T) {
	distribution, err := rollDice(context.Background(), 6, 100)
	if err != nil {
		t.Fatal(err)
	}

	log := logger.Get()
	log.Debug("distribution", zap.Any("distribution", distribution))

	total := int32(0)
	for _, count := range distribution {
		total += count
	}

	assert.Equal(t, int32(100), total, "Total rolls should be equal to 100")
}
