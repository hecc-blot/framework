package api

import (
	apiEntity "github.com/hecc-blot/hecc-blot-core/entity/api"
)

type IValidator interface {
	GetMessages() apiEntity.Messages
}
