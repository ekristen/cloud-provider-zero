package webhook

import (
	admissionv1 "k8s.io/api/admissionregistration/v1"
)

var rules = []admissionv1.RuleWithOperations{
	{
		Operations: []admissionv1.OperationType{
			admissionv1.Create,
			admissionv1.Update,
		},
		Rule: admissionv1.Rule{
			APIGroups:   []string{""},
			APIVersions: []string{"v1"},
			Resources:   []string{"nodes"},
			Scope:       &clusterScope,
		},
	},
}
