package handler

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinderPageReplaceSlotCapMatchesLayoutMaximum(t *testing.T) {
	field, ok := reflect.TypeOf(binderPageReplaceBody{}).FieldByName("Slots")
	require.True(t, ok)
	assert.Equal(t, strconv.Itoa(entity.MaxBinderSlotsPerPage), field.Tag.Get("maxItems"))
}
