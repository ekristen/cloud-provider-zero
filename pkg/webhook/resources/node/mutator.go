package node

import (
	"fmt"
	"github.com/ekristen/cloud-provider-zero/pkg/common"
	"github.com/ekristen/cloud-provider-zero/pkg/patch"
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

	node, err := nodeObject(request)
	if err != nil {
		logrus.WithError(err).Error("unable to decode object")
		return err
	}

	switch request.Operation {
	case admissionv1.Create, admissionv1.Update:
		return v.admitCreateUpdate(node, response, request)
	default:
		return fmt.Errorf("operation type %q not handled", request.Operation)
	}
}

func (v *mutator) admitCreateUpdate(node *corev1.Node, response *webhook.Response, request *webhook.Request) error {
	newNode := node.DeepCopy()

	response.Allowed = true

	if newNode.Spec.ProviderID != "" {
		logrus.Debug("provider id already set, cannot change, skipping")
		return nil
	}

	labels := newNode.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

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

	if err := patch.CreatePatch(newNode, request, response); err != nil {
		logrus.WithError(err).Error("unable to create patch for mutation")
		return err
	}

	logrus.WithField("patch", string(response.Patch)).WithField("type", *response.PatchType).Trace("patched")

	return nil
}

func nodeObject(request *webhook.Request) (*corev1.Node, error) {
	var node runtime.Object
	var err error
	if request.Operation == admissionv1.Create {
		node, err = request.DecodeObject()
	} else {
		node, err = request.DecodeOldObject()
	}
	return node.(*corev1.Node), err
}
