package create_server

import (
	"context"
	"os"
	mcvultrgov "sebekerga/vultr_minecraft_governor"
	"sebekerga/vultr_minecraft_governor/routines"
	"strconv"

	"github.com/melbahja/goph"
	"github.com/vultr/govultr/v3"
)

type CreatingServerContext struct {
	VCtx        context.Context
	VultrClient *govultr.Client

	SSHClient goph.Client

	TargetInstanceLabel  string
	TargetInstanceRegion string
	TargetInstancePlan   string
	TargetInstanceOSID   int
	TargetScriptID       string
	TargetBlockLabel     string

	CreatedInstanceID string
	TargetBlockID     string
}

func InitContext(vultrContext context.Context, vultrClient *govultr.Client, sshClient goph.Client) CreatingServerContext {
	instance_os_id, err := strconv.Atoi(os.Getenv(mcvultrgov.TARGET_INSTANCE_OS_ID_KEY))
	if err != nil {
		panic(err)
	}

	return CreatingServerContext{
		VCtx:        vultrContext,
		VultrClient: vultrClient,

		SSHClient: sshClient,

		TargetInstanceLabel:  os.Getenv(mcvultrgov.TARGET_INSTANCE_LABEL_KEY),
		TargetInstanceRegion: os.Getenv(mcvultrgov.TARGET_INSTANCE_REGION_KEY),
		TargetInstancePlan:   os.Getenv(mcvultrgov.TARGET_INSTANCE_PLAN_KEY),
		TargetInstanceOSID:   instance_os_id,
		TargetScriptID:       os.Getenv(mcvultrgov.TARGET_SCRIPT_ID_KEY),
		TargetBlockLabel:     os.Getenv(mcvultrgov.TARGET_BLOCK_LABEL_KEY),
	}
}

type Ctx = CreatingServerContext
type F = routines.RoutineFunc[Ctx]

func CreatingServerEntry(ctx *Ctx) (F, error) {
	return _CheckIfInstanceCreated, nil
}

func _CheckIfInstanceCreated(ctx *Ctx) (F, error) {
	instances, _, _, err := ctx.VultrClient.Instance.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, nil
	}

	for _, instance := range instances {
		if instance.Label == ctx.TargetInstanceLabel {
			ctx.CreatedInstanceID = instance.ID
			return _FindBlockStorage, nil
		}
	}

	return _CreateInstance, nil
}

func _CreateInstance(ctx *Ctx) (F, error) {
	instance, _, err := ctx.VultrClient.Instance.Create(ctx.VCtx, &govultr.InstanceCreateReq{
		Label:    ctx.TargetInstanceLabel,
		Region:   ctx.TargetInstanceRegion,
		Plan:     ctx.TargetInstancePlan,
		OsID:     ctx.TargetInstanceOSID,
		ScriptID: ctx.TargetScriptID,
	})
	if err != nil {
		return nil, err
	}
	ctx.CreatedInstanceID = instance.ID

	return _FindBlockStorage, nil
}

func _FindBlockStorage(ctx *Ctx) (F, error) {
	blocks, _, _, err := ctx.VultrClient.BlockStorage.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, block := range blocks {
		if block.Label == ctx.TargetBlockLabel {
			ctx.TargetBlockID = block.ID
			return nil, nil
		}
	}

	return _CreateBlockStorage, nil
}

func _CreateBlockStorage(ctx *Ctx) (F, error) {
	block, _, err := ctx.VultrClient.BlockStorage.Create(ctx.VCtx, &govultr.BlockStorageCreate{
		Region: ctx.TargetInstanceRegion,
		Label:  ctx.TargetBlockLabel,
		SizeGB: 10,
	})
	if err != nil {
		return nil, err
	}
	ctx.TargetBlockID = block.ID

	return nil, nil
}

// func _AttachBlockStorage(ctx *C) (F, error) {
// 	_, _, err := ctx.VultrClient.BlockStorage.Attach(ctx.VCtx, ctx.TargetBlockID, &govultr.BlockStorageAttach{
// 		InstanceID: ctx.CreatedInstanceID,
// 		Live:       govultr.Bool(true),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return nil, nil
// }
