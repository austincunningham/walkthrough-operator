package stub

import (
	"context"

	"github.com/integr8ly/walkthrough-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Debug("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	switch o := event.Object.(type) {
	case *v1alpha1.Walkthrough:
		logrus.Debugf("handle walkthrough %v", o)
	}
	return nil
}
