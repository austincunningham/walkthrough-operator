package stub

import (
	"context"
	"github.com/integr8ly/walkthrough-operator/pkg/apis/integreatly/v1alpha1"
	scclientset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewHandler(cfg v1alpha1.Config, k8sclient kubernetes.Interface, svcCatalog scclientset.Interface) sdk.Handler {
	return &Handler{
		cfg:                  cfg,
		serviceCatalogClient: svcCatalog,
		k8sClient:            k8sclient,
	}
}

type Handler struct {
	cfg                  v1alpha1.Config
	k8sClient            kubernetes.Interface
	serviceCatalogClient scclientset.Interface
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Debug("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	switch o := event.Object.(type) {
	case *v1alpha1.Walkthrough:
		logrus.Infof("Walthrough: %v, Phase: %v", o.Name, o.Status.Phase)
		if event.Deleted {
			return nil
		}
		switch o.Status.Phase {
		case v1alpha1.NoPhase:
			wtState, err := h.initialise(o)
			if err != nil {
				return errors.Wrap(err, "failed to init resource")
			}
			return sdk.Update(wtState)
		case v1alpha1.PhaseProvisionNamespace:
			wtState, err := h.provisionNamespace(o)
			if err != nil {
				return errors.Wrap(err, "phase provision namespace failed")
			}
			return sdk.Update(wtState)
		case v1alpha1.PhaseProvisionServices:
			wtState, err := h.provisionServices(o)
			if err != nil {
				return errors.Wrap(err, "phase provision services failed")
			}
			return sdk.Update(wtState)
		case v1alpha1.PhaseUserRoleBindings:
			wtState, err := h.userRoleBindings(o)
			if err != nil {
				return errors.Wrap(err, "phase user role binding failed")
			}
			return sdk.Update(wtState)
		}
		return nil
	}
	return nil
}

func (h *Handler) initialise(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()
	wtCopy.Status.Phase = v1alpha1.PhaseProvisionNamespace
	return wtCopy, nil
}

func (h *Handler) provisionNamespace(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()

	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: wt.Spec.Namespace,
		},
	}

	namespace, err := h.k8sClient.CoreV1().Namespaces().Create(nsSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wlakthrouh namespace")
	}

	logrus.Debugf("namespace %+v", namespace)

	wtCopy.Status.Phase = v1alpha1.PhaseProvisionServices
	return wtCopy, nil
}

func (h *Handler) provisionServices(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()
	//ToDo Implement me
	return wtCopy, nil
}

func (h *Handler) userRoleBindings(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()
	//ToDo Implement me
	return wtCopy, nil
}
