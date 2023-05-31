package common

import (
	"context"
)

type CP struct {
	Ctx        context.Context
	Kubeconfig string
	Name       string
}
