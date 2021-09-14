package volume

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/schemas/validation"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/harvester/pkg/api/vm"
	apisv1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	lhv1beta1 "github.com/harvester/harvester/pkg/generated/controllers/longhorn.io/v1beta1"
	"github.com/harvester/harvester/pkg/util"
	lhmv1beta1 "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta1"
	lhmtypes "github.com/longhorn/longhorn-manager/types"
)

const (
	actionExport = "export"
)

func Formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	if resource.APIObject.Data().String("spec", "sourceType") == actionExport {
		resource.AddAction(request, actionExport)
	}
}

type ExportActionHandler struct {
	//images                      v1beta1.VirtualMachineImageClient
	//imageCache                  v1beta1.VirtualMachineImageCache
	backingImages     			lhv1beta1.BackingImageClient
	pvcCache 					v1.PersistentVolumeClaimCache
}

func (h ExportActionHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.do(rw, req); err != nil {
		status := http.StatusInternalServerError
		if e, ok := err.(*apierror.APIError); ok {
			status = e.Code.Status
		}
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (h ExportActionHandler) do(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	action := vars["action"]
	namespace := vars["namespace"]
	name := vars["name"]

	switch action {
	case actionExport:
		var input vm.ExportVolumeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			return apierror.NewAPIError(validation.InvalidBodyContent, "Failed to decode request body: %v "+err.Error())
		}
		if input.DiskName == "" {
			return apierror.NewAPIError(validation.InvalidBodyContent, "Parameter `volumeName` are required")
		}
		return h.exportVolume(r.Context(), namespace, name, input)
	default:
		return apierror.NewAPIError(validation.InvalidAction, "Unsupported action")
	}
}

func (h ExportActionHandler) exportVolume(ctx context.Context, namespace, name string, input vm.ExportVolumeInput) error {
	// We only permit volume source from existing PersistentVolumeClaim at this moment.
	// KubeVirt won't check PVC existence so we validate it on our own.
	pvc, err := h.pvcCache.Get(namespace, input.VolumeSourceName)
	if err != nil {
		return fmt.Errorf("failed to get pvc %s/%s, error: %s", name, namespace, err.Error())
	}

	bi := &lhmv1beta1.BackingImage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvc.Spec.VolumeName,
			Namespace: util.LonghornSystemNamespaceName,
		},
		Spec: lhmtypes.BackingImageSpec {
			Disks:            map[string]struct{}{},
			SourceType:       lhmtypes.BackingImageDataSourceType("export-from-volume"),
			SourceParameters: map[string]string{
				lhmtypes.DataSourceTypeExportFromVolumeParameterVolumeName: pvc.Spec.VolumeName,
				lhmtypes.DataSourceTypeExportFromVolumeParameterExportType: lhmtypes.DataSourceTypeExportFromVolumeParameterExportTypeRAW,
			},
		},
	}

	backingImage, err = h.backingImages.Create(bi)
	if err != nil {
		logrus.Errorf("fail to create backing image for volume %s", pvc.Spec.VolumeName)
		return err
	}


	return nil
}