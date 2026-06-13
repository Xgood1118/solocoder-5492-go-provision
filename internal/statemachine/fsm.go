package statemachine

import (
	"errors"
	"provision-server/internal/models"
)

var transitions = map[models.DeviceStatus][]models.DeviceStatus{
	models.StatusPending:     {models.StatusProvisioned, models.StatusRetired},
	models.StatusProvisioned: {models.StatusActivated, models.StatusDeactivated, models.StatusRetired},
	models.StatusActivated:   {models.StatusDeactivated, models.StatusRetired},
	models.StatusDeactivated: {models.StatusActivated, models.StatusProvisioned, models.StatusRetired},
	models.StatusRetired:     {},
}

func CanTransition(from, to models.DeviceStatus) bool {
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func ValidateTransition(from, to models.DeviceStatus) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return errors.New("invalid state transition: " + string(from) + " -> " + string(to))
	}
	return nil
}
