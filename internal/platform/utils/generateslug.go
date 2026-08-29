package utils

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

func GenerateSlug(name string) string {
	return fmt.Sprintf(
		"%s-%s",
		slug.Make(name),
		uuid.NewString()[:8],
	)
}
