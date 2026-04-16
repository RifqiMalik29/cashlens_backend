package telegram

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFilterFixedCategories(t *testing.T) {
	cats := []*models.Category{
		{ID: uuid.New(), Name: "Makanan & Minuman"},
		{ID: uuid.New(), Name: "Transportasi"},
		{ID: uuid.New(), Name: "Custom Category"},
		{ID: uuid.New(), Name: "Lainnya"},
		{ID: uuid.New(), Name: "Another Custom"},
	}

	result := filterFixedCategories(cats)

	assert.Len(t, result, 3)
	names := make([]string, len(result))
	for i, c := range result {
		names[i] = c.Name
	}
	assert.Contains(t, names, "Makanan & Minuman")
	assert.Contains(t, names, "Transportasi")
	assert.Contains(t, names, "Lainnya")
	assert.NotContains(t, names, "Custom Category")
}
