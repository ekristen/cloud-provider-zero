package node

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/rancher/wrangler/pkg/webhook"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/trace"

	"github.com/ekristen/cloud-provider-zero/pkg/common"
	"github.com/ekristen/cloud-provider-zero/pkg/patch"
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

func (v *mutator) admitCreateUpdate(node *corev1.Node, response *webhook.Response, _ *webhook.Request) error {
	newNode := node.DeepCopy()

	logger := logrus.WithField("node", node.Name)

	// Note: this is here so that anytime we return the response is allowed through
	response.Allowed = true

	if newNode.Spec.ProviderID != "" {
		logger.Debug("provider id already set, cannot change, skipping")
		return nil
	}

	labels := newNode.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	provider, providerOK := labels[common.ProviderLabel]
	instanceID, instanceOK := labels[common.InstanceIDLabel]

	if !providerOK {
		logger.Info("provider label not set, skipping")
		return nil
	}
	if !instanceOK {
		logger.Info("instance-id label not set, skipping")
		return nil
	}

	if provider == "aws" {
		zone, zoneOk := labels[corev1.LabelTopologyZone]

		if !zoneOk {
			logger.Info("topology.kubernetes.io/zone label missing, skipping")
			return nil
		}

		newNode.Spec.ProviderID = fmt.Sprintf("%s:///%s/%s", provider, zone, instanceID)
	}

	if err := patch.CreatePatch(node, newNode, response); err != nil {
		logger.WithError(err).Error("unable to create patch for mutation")
		return err
	}

	logger.WithField("patch", string(response.Patch)).WithField("type", *response.PatchType).Info("patched")

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
