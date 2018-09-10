package stub

import (
	"context"
	"github.com/integr8ly/walkthrough-operator/pkg/apis/integreatly/v1alpha1"
	scclientset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
		case v1alpha1.PhaseUserRoleBindings:
			wtState, err := h.userRoleBindings(o)
			if err != nil {
				return errors.Wrap(err, "phase user role binding failed")
			}
			return sdk.Update(wtState)
		case v1alpha1.PhaseProvisionServices:
			wtState, err := h.provisionServices(o)
			if err != nil {
				return errors.Wrap(err, "phase provision services failed")
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

	labels := map[string]string{
		"aerogear.org/walkthrough-operator": "true",
	}
	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   wt.Spec.UserName + "-walkthroughs",
			Labels: labels,
		},
	}

	namespace, err := h.k8sClient.CoreV1().Namespaces().Create(nsSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create walkthrough namespace")
	}

	wtCopy.Status.Namespace = namespace.Name
	wtCopy.Status.Phase = v1alpha1.PhaseUserRoleBindings
	return wtCopy, nil
}

func (h *Handler) userRoleBindings(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()

	userRoles := []string{"edit"}

	for _, role := range userRoles {
		err := sdk.Create(newRoleBinding(wt.Spec.UserName, wt.Status.Namespace, role))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create user %s role binding", role)
		}
	}

	wtCopy.Status.Phase = v1alpha1.PhaseProvisionServices
	return wtCopy, nil
}

func (h *Handler) provisionServices(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()
	//ToDo Implement me
	return wtCopy, nil
}

func newRoleBinding(username, namsepace, roleName string) *rbacv1.RoleBinding {
	labels := map[string]string{
		"aerogear.org/walkthrough-operator": "true",
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: roleName + "-",
			Namespace:    namsepace,
			Labels:       labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     roleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				Name:     username,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}
}
