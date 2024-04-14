package create_server

import (
	"context"
	"fmt"
	"log"
	"os"
	mcvultrgov "sebekerga/vultr_minecraft_governor"
	routines "sebekerga/vultr_minecraft_governor/routines"
	"strconv"
	"time"

	"github.com/melbahja/goph"
	"github.com/vultr/govultr/v3"
)

const MAX_ATTEMPTS = 5

type CreatingServerContext struct {
	VCtx        context.Context
	VultrClient *govultr.Client

	SSHClient *goph.Client

	TargetInstanceLabel  string
	TargetInstanceRegion string
	TargetInstancePlan   string
	TargetInstanceOSID   int
	TargetScriptID       string
	TargetBlockLabel     string

	CreatedInstanceID string
	CreatedInstanceIP string
	TargetBlockID     string

	AttemptCount int
}

func InitContext(vultrContext context.Context, vultrClient *govultr.Client) CreatingServerContext {
	instance_os_id, err := strconv.Atoi(os.Getenv(mcvultrgov.TARGET_INSTANCE_OS_ID_KEY))
	if err != nil {
		panic(err)
	}

	return CreatingServerContext{
		VCtx:        vultrContext,
		VultrClient: vultrClient,

		TargetInstanceLabel:  os.Getenv(mcvultrgov.TARGET_INSTANCE_LABEL_KEY),
		TargetInstanceRegion: os.Getenv(mcvultrgov.TARGET_INSTANCE_REGION_KEY),
		TargetInstancePlan:   os.Getenv(mcvultrgov.TARGET_INSTANCE_PLAN_KEY),
		TargetInstanceOSID:   instance_os_id,
		TargetScriptID:       os.Getenv(mcvultrgov.TARGET_SCRIPT_ID_KEY),
		TargetBlockLabel:     os.Getenv(mcvultrgov.TARGET_BLOCK_LABEL_KEY),

		AttemptCount: 0,
	}
}

type Ctx = CreatingServerContext
type F = routines.RoutineFunc[Ctx]

func CreatingServerEntry(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	return _CheckIfInstanceCreated, nil
}

func _CheckIfInstanceCreated(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	instances, _, _, err := ctx.VultrClient.Instance.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		ph(routines.ERROR, "Error occurred while fetching instances")
		return nil, err
	}

	for _, instance := range instances {
		if instance.Label == ctx.TargetInstanceLabel {
			ctx.CreatedInstanceID = instance.ID
			ctx.CreatedInstanceIP = instance.MainIP
			ph(routines.INFO, fmt.Sprintf("Found existing instance, ID: %s", ctx.CreatedInstanceID))
			return _FindBlockStorage, nil
		}
	}

	ph(routines.INFO, "No existing instance found")
	return _CreateInstance, nil
}

func _CreateInstance(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Creating new instance")
	instance, _, err := ctx.VultrClient.Instance.Create(ctx.VCtx, &govultr.InstanceCreateReq{
		Label:    ctx.TargetInstanceLabel,
		Region:   ctx.TargetInstanceRegion,
		Plan:     ctx.TargetInstancePlan,
		OsID:     ctx.TargetInstanceOSID,
		ScriptID: ctx.TargetScriptID,
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating instance")
		return nil, err
	}
	ctx.CreatedInstanceID = instance.ID
	ctx.CreatedInstanceIP = instance.MainIP
	ph(routines.INFO, fmt.Sprintf("Instance created, ID: %s", ctx.CreatedInstanceID))

	return _FindBlockStorage, nil
}

func _FindBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Checking if block storage already created")
	blocks, _, _, err := ctx.VultrClient.BlockStorage.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		ph(routines.ERROR, "Error occurred while fetching block storage")
		return nil, err
	}

	for _, block := range blocks {
		if block.Label == ctx.TargetBlockLabel {
			ctx.TargetBlockID = block.ID
			ph(routines.INFO, fmt.Sprintf("Found existing block storage, ID: %s", ctx.TargetBlockID))
			return _AwaitServerSSH, nil
		}
	}

	return _CreateBlockStorage, nil
}

func _CreateBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Creating new block storage")
	block, _, err := ctx.VultrClient.BlockStorage.Create(ctx.VCtx, &govultr.BlockStorageCreate{
		Region: ctx.TargetInstanceRegion,
		Label:  ctx.TargetBlockLabel,
		SizeGB: 10,
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating block storage")
		return nil, err
	}
	ctx.TargetBlockID = block.ID
	ph(routines.INFO, fmt.Sprintf("Block storage created, ID: %s", ctx.TargetBlockID))

	return _AwaitServerSSH, nil
}

func _AwaitServerSSH(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	if ctx.AttemptCount <= 0 {
		ph(routines.INFO, fmt.Sprintf("Waiting for server SSH, IP: %s", ctx.CreatedInstanceIP))
	}

	keyPath := os.Getenv(mcvultrgov.INSTANCE_SSH_KEY_PATH_KEY)
	auth, err := goph.Key(keyPath, "")
	if err != nil {
		ph(routines.ERROR, "Unable to get SSH key")
		return nil, err
	}
	ph(routines.INFO, "Loaded key")

	// I cant know host key before server is booted
	sshClient, err := goph.NewUnknown("root", ctx.CreatedInstanceIP, auth)
	if err != nil {
		ph(routines.INFO, fmt.Sprintf("Attempt %d: Server not booted yet", ctx.AttemptCount+1))
		log.Printf("Error: %s", err)
		ctx.AttemptCount++
		if ctx.AttemptCount >= MAX_ATTEMPTS {
			ph(routines.ERROR, "Max attempts reached")
			return nil, err
		}
		time.Sleep(10 * time.Second)
		return _AwaitServerSSH, nil
	}

	ctx.SSHClient = sshClient

	ph(routines.INFO, "SSH connection established")
	ctx.AttemptCount = 0
	return _AttachBlockStorage, nil
}

func _AttachBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Attaching block storage")
	err := ctx.VultrClient.BlockStorage.Attach(ctx.VCtx, ctx.TargetBlockID, &govultr.BlockStorageAttach{
		InstanceID: ctx.CreatedInstanceID,
		Live:       govultr.BoolToBoolPtr(true),
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while attaching block storage")
		return nil, err
	}
	ph(routines.INFO, "Block storage attached")

	return _MountingBlockStorage, nil
}

func _MountingBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	_, err := ctx.SSHClient.Run("mkdir /mnt/minecraft")
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating directory")
		return nil, err
	}
	ph(routines.INFO, "Directory created")

	_, err = ctx.SSHClient.Run("mount /dev/sda /mnt/minecraft")
	if err != nil {
		ph(routines.ERROR, "Error occurred while mounting block storage")
		return nil, err
	}
	ph(routines.INFO, "Block storage mounted")

	return nil, nil
}
