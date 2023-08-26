package node

import (
	"fmt"
	"github.com/ekristen/cloud-provider-zero/pkg/common"
	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/trace"
	"time"
)

const resourceName = "Node"

func NewMutation() webhook.Handler {
	return &mutator{}
}

type mutator struct{}

func (v *mutator) Admit(response *webhook.Response, request *webhook.Request) error {
	if request.DryRun != nil && *request.DryRun {
		response.Allowed = true
		return nil
	}

	listTrace := trace.New(fmt.Sprintf("%s Validator Admit", resourceName), trace.Field{Key: "user", Value: request.UserInfo.Username})
	defer listTrace.LogIfLong(2 * time.Second)

	switch request.Operation {
	case admissionv1.Create, admissionv1.Update:
		return v.admitModify(response, request)
	}

	response.Allowed = true
	return nil
}

func (v *mutator) admitModify(response *webhook.Response, request *webhook.Request) error {
	var err error
	var obj runtime.Object

	if request.Operation == admissionv1.Create {
		obj, err = request.DecodeObject()
	} else {
		obj, err = request.DecodeOldObject()
	}
	if err != nil {
		return err
	}

	response.Allowed = true

	object := obj.(*corev1.Node)

	if object.Spec.ProviderID != "" {
		logrus.Debug("provider id already set, cannot change, skipping")
		return nil
	}

	labels := object.GetLabels()
	newNode := object.DeepCopy()

	provider, providerOk := labels[common.ProviderLabel]
	instanceId, instanceOk := labels[common.InstanceIdLabel]

	if !providerOk {
		logrus.Debug("provider label not set, skipping")
		return nil
	}
	if !instanceOk {
		logrus.Debug("instance-id label not set, skipping")
		return nil
	}

	if provider == "aws" {
		zone, zoneOk := labels[corev1.LabelTopologyZone]

		if !zoneOk {
			logrus.Debug("topology.kubernetes.io/zone label missing, skipping")
			return nil
		}

		newNode.Spec.ProviderID = fmt.Sprintf("%s:///%s/%s", provider, zone, instanceId)
	}

	if err := response.CreatePatch(request, newNode); err != nil {
		return err
	}

	return nil
}
