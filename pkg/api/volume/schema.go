package volume

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/pkg/schemas"

	"github.com/harvester/harvester/pkg/config"
)

func RegisterSchema(scaled *config.Scaled, server *server.Server, options config.Options) error {
	t := schema.Template{
		ID: "harvesterhci.io.volumeimage",
		Customize: func(s *types.APISchema) {
			s.Formatter = Formatter
			s.ResourceActions = map[string]schemas.Action{
				actionExport: {},
			}
			s.ActionHandlers = map[string]http.Handler{
				actionExport: ExportActionHandler{
					httpClient:                  http.Client{},
					images:                      scaled.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineImage(),
					imageCache:                  scaled.HarvesterFactory.Harvesterhci().V1beta1().VirtualMachineImage().Cache(),
					backingImageDataSources:     scaled.LonghornFactory.Longhorn().V1beta1().BackingImageDataSource(),
					backingImageDataSourceCache: scaled.LonghornFactory.Longhorn().V1beta1().BackingImageDataSource().Cache(),
					pvcs:     					scaled.CoreFactory.Core().V1().PersistentVolumeClaim(),
					pvcCache: 					scaled.CoreFactory.Core().V1().PersistentVolumeClaim().Cache(),
				},
			}
		},
	}
	server.SchemaFactory.AddTemplate(t)
	return nil
}
