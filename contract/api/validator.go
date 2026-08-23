package api

import (
	apiEntity "github.com/hecc-blot/framework/entity/api"
)

type IValidator interface {
	GetMessages() apiEntity.Messages
}
