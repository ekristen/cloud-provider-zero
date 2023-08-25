package patch

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// CreatePatch accepts an old and a new object and creates a patch of the differences as
// specified in http://jsonpatch.com/ and updates the response accordingly.
// The old object should be the Raw object received in the original request
func CreatePatch(oldJSON []byte, newObj interface{}, response *v1.AdmissionResponse) ([]byte, *v1.PatchType, error) {
	newJSON, err := json.Marshal(newObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal newObj to JSON: %w", err)
	}

	patch := admission.PatchResponseFromRaw(oldJSON, newJSON)
	if len(patch.Patches) == 0 {
		return nil, nil, nil
	}
	patchJSON, err := json.Marshal(patch.Patches)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal generated patch to JSON: %w", err)
	}

	return patchJSON, patch.PatchType, nil
}
