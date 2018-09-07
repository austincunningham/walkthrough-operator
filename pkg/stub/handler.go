package stub

import (
	"context"
	"encoding/json"
	"github.com/integr8ly/walkthrough-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	scclientset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		case v1alpha1.PhaseProvisionedServices:
			wtState, err := h.provisionedServices(o)
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
	wtCopy.Status.Ready = false
	wtCopy.Status.Services = map[string]string{}
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

	requiredServices := wt.Spec.Services
	logrus.Debugf("provision services: required=%v", requiredServices)
	csc, err := h.serviceCatalogClient.Servicecatalog().ClusterServiceClasses().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service classes")
	}

	serviceClassMap := map[string]v1beta1.ClusterServiceClass{}
	for i, _ := range requiredServices {
		for i2, _ := range csc.Items {
			if csc.Items[i2].Spec.ExternalName == requiredServices[i] {
				serviceClassMap[requiredServices[i]] = csc.Items[i2]
			}
		}
	}

	if len(serviceClassMap) != len(wt.Spec.Services) {
		return nil, errors.Errorf("Unable to get all service classes for required services: %v", requiredServices)
	}

	//ToDO This is not currently used, but we know we will need it for enmaase
	decodedParams := map[string]string{}
	parameters, err := json.Marshal(decodedParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal decoded parameters")
	}
	//

	for serviceName, serviceClass := range serviceClassMap {
		logrus.Debugf("required services, servicename: %s, serviceClass: %+v", serviceName, serviceClass.Spec.ExternalID)
		if _, ok := wtCopy.Status.Services[serviceName]; !ok {
			logrus.Debugf("creating service %s", serviceName)
			si := h.newServiceInstance(wt.Status.Namespace, parameters, serviceClass)
			serviceInstance, err := h.serviceCatalogClient.ServicecatalogV1beta1().ServiceInstances(wt.Status.Namespace).Create(&si)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create service instance for %s", serviceName)
			}
			wtCopy.Status.Services[serviceName] = serviceInstance.GetName()
			logrus.Infof("created service: %s, %s ", serviceName, serviceInstance.GetName())
		}
	}

	wtCopy.Status.Phase = v1alpha1.PhaseProvisionedServices
	return wtCopy, nil
}

func (h *Handler) provisionedServices(wt *v1alpha1.Walkthrough) (*v1alpha1.Walkthrough, error) {
	wtCopy := wt.DeepCopy()
	allServicesReady := true

	for _, svcInstName := range wtCopy.Status.Services {
		si, err := h.serviceCatalogClient.ServicecatalogV1beta1().ServiceInstances(wtCopy.Status.Namespace).Get(svcInstName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get service instance %s", svcInstName)
		}

		serviceReady := false
		for _, c := range si.Status.Conditions {
			if c.Type == v1beta1.ServiceInstanceConditionReady && c.Status == v1beta1.ConditionTrue {
				serviceReady = true
				break
			}
		}

		if !serviceReady {
			allServicesReady = false
		}
		logrus.Debugf("service %s, ready: %v", svcInstName, serviceReady)
	}

	if allServicesReady {
		logrus.Debug("All services ready!!!")
		wtCopy.Status.Ready = true
		wtCopy.Status.Phase = v1alpha1.PhaseComplete
	}
	return wtCopy, nil
}

func newRoleBinding(username, namespace, roleName string) *rbacv1.RoleBinding {
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
			Namespace:    namespace,
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

func (h *Handler) newServiceInstance(namespace string, parameters []byte, sc v1beta1.ClusterServiceClass) v1beta1.ServiceInstance {
	return v1beta1.ServiceInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "servicecatalog.k8s.io/v1beta1",
			Kind:       "ServiceInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: sc.Spec.ExternalName + "-",
		},
		Spec: v1beta1.ServiceInstanceSpec{
			PlanReference: v1beta1.PlanReference{
				ClusterServiceClassExternalName: sc.Spec.ExternalName,
			},
			ClusterServiceClassRef: &v1beta1.ClusterObjectReference{
				Name: sc.Name,
			},
			Parameters: &runtime.RawExtension{Raw: parameters},
		},
	}
}
