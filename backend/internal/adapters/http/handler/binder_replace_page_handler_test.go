package handler

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinderPageReplaceSlotCapMatchesLayoutMaximum(t *testing.T) {
	field, ok := reflect.TypeOf(binderPageReplaceBody{}).FieldByName("Slots")
	require.True(t, ok)
	assert.Equal(t, strconv.Itoa(dto.BinderMaxSlotsPerPage), field.Tag.Get("maxItems"))
}
