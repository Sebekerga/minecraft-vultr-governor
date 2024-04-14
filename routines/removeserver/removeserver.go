package remove_server

import (
	"context"
	"os"
	mcvultrgov "sebekerga/vultr_minecraft_governor"
	"sebekerga/vultr_minecraft_governor/routines"

	"github.com/vultr/govultr/v3"
)

type CreatingServerContext struct {
	VCtx        context.Context
	VultrClient *govultr.Client

	TargetInstanceLabel  string
	TargetInstanceRegion string

	CreatedInstanceID string
	TargetBlockID     string
}

func InitContext(vultrContext context.Context, vultrClient *govultr.Client) CreatingServerContext {
	return CreatingServerContext{
		VCtx:        vultrContext,
		VultrClient: vultrClient,

		TargetInstanceLabel:  os.Getenv(mcvultrgov.TARGET_INSTANCE_LABEL_KEY),
		TargetInstanceRegion: os.Getenv(mcvultrgov.TARGET_INSTANCE_REGION_KEY),
	}
}

type Ctx = CreatingServerContext
type F = routines.RoutineFunc[Ctx]

func RemovingServerEntry(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	return _CheckIfInstanceExists, nil
}

func _CheckIfInstanceExists(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	instances, _, _, err := ctx.VultrClient.Instance.List(ctx.VCtx, &govultr.ListOptions{
		Label:  ctx.TargetInstanceLabel,
		Region: ctx.TargetInstanceRegion,
	})
	if err != nil {
		ph(routines.ERROR, "Error listing instances")
		return nil, err
	}

	if len(instances) == 0 {
		ph(routines.INFO, "Instance not found")
		return nil, nil
	}

	ctx.CreatedInstanceID = instances[0].ID
	return _RemoveInstance, nil
}

func _RemoveInstance(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	err := ctx.VultrClient.Instance.Delete(ctx.VCtx, ctx.CreatedInstanceID)
	if err != nil {
		ph(routines.ERROR, "Error deleting instance")
		return nil, err
	}

	ph(routines.INFO, "Instance deleted")
	return nil, nil
}
