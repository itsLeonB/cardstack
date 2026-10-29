package mapper

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSlugOf(t *testing.T) {
	id := uuid.MustParse("0192f3a4-7b6c-7d8e-9f01-23456789abcd")

	cases := map[string]struct {
		name string
		want string
	}{
		"lower-cases and hyphenates words": {"Ada Lovelace", "ada-lovelace-9abcd"},
		"collapses runs of punctuation":    {"  Grace -- Hopper!! ", "grace-hopper-9abcd"},
		"keeps digits":                     {"Player 1", "player-1-9abcd"},
		"emoji only falls back":            {"🔥🔥", "user-9abcd"},
		"non-Latin only falls back":        {"山田太郎", "user-9abcd"},
		"empty falls back":                 {"", "user-9abcd"},
	}
	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			assert.Equal(t, tc.want, SlugOf(tc.name, "user", id))
		})
	}
}
